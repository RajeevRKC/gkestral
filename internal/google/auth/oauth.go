package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Sentinel errors for OAuth flow.
var (
	ErrConsentDenied = errors.New("auth: user denied consent")
	ErrStateMismatch = errors.New("auth: OAuth state parameter mismatch (possible CSRF)")
	ErrPortInUse     = errors.New("auth: callback port already in use")
	ErrFlowTimeout   = errors.New("auth: OAuth flow timed out waiting for callback")
	ErrFlowCancelled = errors.New("auth: OAuth flow was cancelled")
)

// Google OAuth 2.0 endpoints.
const (
	GoogleAuthURL  = "https://accounts.google.com/o/oauth2/v2/auth"
	GoogleTokenURL = "https://oauth2.googleapis.com/token"
)

// OAuthClient handles the OAuth 2.0 desktop flow with PKCE.
type OAuthClient struct {
	clientID     string
	clientSecret string
	scopes       ScopeSet
	callbackPort int
	tokenStore   TokenStore
	authURL      string
	tokenURL     string
	flowTimeout  time.Duration

	// HTTPClient is used for token exchange requests. Defaults to http.DefaultClient.
	HTTPClient *http.Client

	// OpenBrowserFunc is called to open the auth URL in the user's browser.
	// Defaults to OpenBrowser. Override for testing.
	OpenBrowserFunc func(url string) error
}

// OAuthOption configures an OAuthClient.
type OAuthOption func(*OAuthClient)

// WithClientID sets the OAuth client ID.
func WithClientID(id string) OAuthOption {
	return func(o *OAuthClient) { o.clientID = id }
}

// WithClientSecret sets the OAuth client secret.
func WithClientSecret(secret string) OAuthOption {
	return func(o *OAuthClient) { o.clientSecret = secret }
}

// WithScopes sets the requested OAuth scopes.
func WithScopes(scopes ScopeSet) OAuthOption {
	return func(o *OAuthClient) { o.scopes = scopes }
}

// WithCallbackPort sets the localhost port for the OAuth callback.
func WithCallbackPort(port int) OAuthOption {
	return func(o *OAuthClient) { o.callbackPort = port }
}

// WithTokenStore sets the token persistence backend.
func WithTokenStore(store TokenStore) OAuthOption {
	return func(o *OAuthClient) { o.tokenStore = store }
}

// WithAuthURL overrides the authorization endpoint (for testing).
func WithAuthURL(url string) OAuthOption {
	return func(o *OAuthClient) { o.authURL = url }
}

// WithTokenURL overrides the token exchange endpoint (for testing).
func WithTokenURL(url string) OAuthOption {
	return func(o *OAuthClient) { o.tokenURL = url }
}

// WithFlowTimeout sets the maximum time to wait for user consent.
func WithFlowTimeout(d time.Duration) OAuthOption {
	return func(o *OAuthClient) { o.flowTimeout = d }
}

// NewOAuthClient creates an OAuth client with the given options.
func NewOAuthClient(opts ...OAuthOption) *OAuthClient {
	o := &OAuthClient{
		callbackPort:    8085,
		authURL:         GoogleAuthURL,
		tokenURL:        GoogleTokenURL,
		flowTimeout:     5 * time.Minute,
		HTTPClient:      http.DefaultClient,
		OpenBrowserFunc: OpenBrowser,
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// DesktopFlow runs the full OAuth 2.0 desktop flow with PKCE:
//  1. Generate PKCE verifier + challenge
//  2. Generate cryptographic state parameter
//  3. Start local callback server
//  4. Open browser to Google consent page
//  5. Wait for callback with auth code
//  6. Validate state, exchange code for tokens
//  7. Store tokens and return
func (o *OAuthClient) DesktopFlow(ctx context.Context) (StoredToken, error) {
	// Validate credentials before starting the flow.
	if o.clientID == "" || o.clientSecret == "" {
		return StoredToken{}, errors.New("auth: client ID and secret must be configured before starting OAuth flow")
	}

	// Generate PKCE code verifier (43-128 unreserved characters).
	verifier, err := generateCodeVerifier()
	if err != nil {
		return StoredToken{}, fmt.Errorf("auth: generate PKCE verifier: %w", err)
	}
	challenge := computeCodeChallenge(verifier)

	// Generate state parameter for CSRF protection.
	state, err := generateState()
	if err != nil {
		return StoredToken{}, fmt.Errorf("auth: generate state: %w", err)
	}

	// Start local callback server.
	callbackResult := make(chan callbackData, 1)
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", o.callbackPort))
	if err != nil {
		return StoredToken{}, ErrPortInUse
	}
	defer listener.Close() // Ensure listener is closed on ALL exit paths.

	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", o.callbackPort)

	var accepted sync.Once
	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			q := r.URL.Query()

			// Only accept requests that carry an OAuth state or error param.
			// Rejects stray requests (port scanners, favicon fetches) that
			// would otherwise consume the sync.Once.
			if q.Get("state") == "" && q.Get("error") == "" {
				http.Error(w, "Not an OAuth callback", http.StatusBadRequest)
				return
			}

			accepted.Do(func() {
				callbackResult <- callbackData{
					code:  q.Get("code"),
					state: q.Get("state"),
					err:   q.Get("error"),
				}
			})
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<html><body><h2>Authentication complete</h2><p>You can close this window and return to Gkestral.</p><script>window.close()</script></body></html>`)
		}),
	}

	go srv.Serve(listener)

	// Ensure server is shut down on all exit paths.
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	// Build authorization URL.
	authURL := o.buildAuthURL(state, challenge, redirectURI)

	// Open browser.
	if o.OpenBrowserFunc != nil {
		if err := o.OpenBrowserFunc(authURL); err != nil {
			return StoredToken{}, fmt.Errorf("auth: open browser: %w", err)
		}
	}

	// Wait for callback, timeout, or cancellation.
	var cb callbackData
	select {
	case cb = <-callbackResult:
		// Got callback.
	case <-time.After(o.flowTimeout):
		return StoredToken{}, ErrFlowTimeout
	case <-ctx.Done():
		return StoredToken{}, ErrFlowCancelled
	}

	// Handle error from Google (user denied, etc.).
	if cb.err != "" {
		if cb.err == "access_denied" {
			return StoredToken{}, ErrConsentDenied
		}
		return StoredToken{}, fmt.Errorf("auth: OAuth error: %s", cb.err)
	}

	// Validate state parameter.
	if cb.state != state {
		return StoredToken{}, ErrStateMismatch
	}

	if cb.code == "" {
		return StoredToken{}, errors.New("auth: no authorization code in callback")
	}

	// Exchange authorization code for tokens.
	token, err := o.exchangeCode(ctx, cb.code, verifier, redirectURI)
	if err != nil {
		return StoredToken{}, err
	}

	// Store token if store is configured.
	if o.tokenStore != nil {
		if err := o.tokenStore.Save(token); err != nil {
			return StoredToken{}, fmt.Errorf("auth: save token: %w", err)
		}
	}

	return token, nil
}

// AuthURL returns the authorization URL without starting the flow.
// Useful for testing or custom flow implementations.
func (o *OAuthClient) AuthURL(state, challenge, redirectURI string) string {
	return o.buildAuthURL(state, challenge, redirectURI)
}

func (o *OAuthClient) buildAuthURL(state, challenge, redirectURI string) string {
	params := url.Values{
		"client_id":             {o.clientID},
		"redirect_uri":         {redirectURI},
		"response_type":        {"code"},
		"scope":                {strings.Join(o.scopes.Slice(), " ")},
		"state":                {state},
		"code_challenge":       {challenge},
		"code_challenge_method": {"S256"},
		"access_type":          {"offline"}, // Request refresh token.
		"prompt":               {"consent"}, // Force consent to get refresh token.
		"include_granted_scopes": {"true"},  // Progressive permissioning.
	}
	return o.authURL + "?" + params.Encode()
}

type callbackData struct {
	code  string
	state string
	err   string
}

// exchangeCode trades the authorization code for access + refresh tokens.
func (o *OAuthClient) exchangeCode(ctx context.Context, code, verifier, redirectURI string) (StoredToken, error) {
	data := url.Values{
		"client_id":     {o.clientID},
		"client_secret": {o.clientSecret},
		"code":          {code},
		"code_verifier": {verifier},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {redirectURI},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.tokenURL,
		strings.NewReader(data.Encode()))
	if err != nil {
		return StoredToken{}, fmt.Errorf("auth: create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := o.HTTPClient.Do(req)
	if err != nil {
		return StoredToken{}, fmt.Errorf("auth: token exchange: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return StoredToken{}, fmt.Errorf("auth: read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return StoredToken{}, fmt.Errorf("auth: token exchange failed (%d): %s",
			resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		Scope        string `json:"scope"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return StoredToken{}, fmt.Errorf("auth: parse token response: %w", err)
	}

	expiresIn := tokenResp.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 3600 // Default to 1 hour if server returns zero/negative.
	}
	expiry := time.Now().Add(time.Duration(expiresIn) * time.Second)

	return StoredToken{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		TokenType:    tokenResp.TokenType,
		Expiry:       expiry.Format(time.RFC3339),
		Scopes:       strings.Fields(tokenResp.Scope),
	}, nil
}

// --- PKCE helpers ---

// generateCodeVerifier creates a cryptographically random PKCE code verifier.
// Length: 64 characters from the unreserved character set [A-Za-z0-9-._~].
func generateCodeVerifier() (string, error) {
	b := make([]byte, 48) // 48 bytes -> 64 base64url chars
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// computeCodeChallenge computes S256 code challenge from verifier.
func computeCodeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// generateState creates a cryptographically random state parameter.
func generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
