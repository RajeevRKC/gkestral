package auth

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// mockTokenServer creates a test server that handles token exchange.
func mockTokenServer(t *testing.T, wantVerifier bool) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("token endpoint: method = %s, want POST", r.Method)
		}
		r.ParseForm()
		if r.Form.Get("grant_type") != "authorization_code" {
			t.Errorf("grant_type = %q", r.Form.Get("grant_type"))
		}
		if wantVerifier && r.Form.Get("code_verifier") == "" {
			t.Error("code_verifier missing from token request")
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"access_token": "mock-access-token",
			"refresh_token": "mock-refresh-token",
			"token_type": "Bearer",
			"expires_in": 3600,
			"scope": "https://www.googleapis.com/auth/userinfo.email"
		}`)
	}))
}

func findFreePort() int {
	l, _ := net.Listen("tcp", "127.0.0.1:0")
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}

func TestDesktopFlow_Success(t *testing.T) {
	tokenSrv := mockTokenServer(t, true)
	defer tokenSrv.Close()

	port := findFreePort()

	kr := newMockKeyring()
	store := NewKeyringTokenStore(kr)

	// Mock auth server that redirects to our callback immediately.
	authSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		state := r.URL.Query().Get("state")
		challenge := r.URL.Query().Get("code_challenge")
		if challenge == "" {
			t.Error("code_challenge missing from auth URL")
		}
		if r.URL.Query().Get("code_challenge_method") != "S256" {
			t.Error("code_challenge_method != S256")
		}
		if r.URL.Query().Get("include_granted_scopes") != "true" {
			t.Error("include_granted_scopes not set")
		}
		if r.URL.Query().Get("access_type") != "offline" {
			t.Error("access_type != offline")
		}
		// Simulate Google redirecting back with auth code.
		redirectURI := r.URL.Query().Get("redirect_uri")
		http.Redirect(w, r, redirectURI+"?code=mock-auth-code&state="+state, http.StatusFound)
	}))
	defer authSrv.Close()

	client := NewOAuthClient(
		WithClientID("test-client-id"),
		WithClientSecret("test-client-secret"),
		WithScopes(DefaultV01Scopes()),
		WithCallbackPort(port),
		WithTokenStore(store),
		WithAuthURL(authSrv.URL),
		WithTokenURL(tokenSrv.URL),
		WithFlowTimeout(10*time.Second),
	)
	// Override browser open to make HTTP request to auth server instead.
	client.OpenBrowserFunc = func(authURL string) error {
		go http.Get(authURL)
		return nil
	}

	token, err := client.DesktopFlow(context.Background())
	if err != nil {
		t.Fatalf("DesktopFlow error: %v", err)
	}

	if token.AccessToken != "mock-access-token" {
		t.Errorf("AccessToken = %q", token.AccessToken)
	}
	if token.RefreshToken != "mock-refresh-token" {
		t.Errorf("RefreshToken = %q", token.RefreshToken)
	}

	// Verify token was stored.
	if !store.Exists() {
		t.Error("token should be stored after successful flow")
	}
}

func TestDesktopFlow_ConsentDenied(t *testing.T) {
	port := findFreePort()

	authSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirectURI := r.URL.Query().Get("redirect_uri")
		http.Redirect(w, r, redirectURI+"?error=access_denied", http.StatusFound)
	}))
	defer authSrv.Close()

	client := NewOAuthClient(
		WithClientID("test-id"),
		WithClientSecret("test-secret"),
		WithScopes(DefaultV01Scopes()),
		WithCallbackPort(port),
		WithAuthURL(authSrv.URL),
		WithTokenURL("http://unused"),
		WithFlowTimeout(10*time.Second),
	)
	client.OpenBrowserFunc = func(authURL string) error {
		go http.Get(authURL)
		return nil
	}

	_, err := client.DesktopFlow(context.Background())
	if !errors.Is(err, ErrConsentDenied) {
		t.Errorf("error = %v, want ErrConsentDenied", err)
	}
}

func TestDesktopFlow_StateMismatch(t *testing.T) {
	port := findFreePort()

	authSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirectURI := r.URL.Query().Get("redirect_uri")
		// Send back a WRONG state parameter.
		http.Redirect(w, r, redirectURI+"?code=mock-code&state=wrong-state", http.StatusFound)
	}))
	defer authSrv.Close()

	client := NewOAuthClient(
		WithClientID("test-id"),
		WithClientSecret("test-secret"),
		WithScopes(DefaultV01Scopes()),
		WithCallbackPort(port),
		WithAuthURL(authSrv.URL),
		WithTokenURL("http://unused"),
		WithFlowTimeout(10*time.Second),
	)
	client.OpenBrowserFunc = func(authURL string) error {
		go http.Get(authURL)
		return nil
	}

	_, err := client.DesktopFlow(context.Background())
	if !errors.Is(err, ErrStateMismatch) {
		t.Errorf("error = %v, want ErrStateMismatch", err)
	}
}

func TestDesktopFlow_Timeout(t *testing.T) {
	port := findFreePort()

	client := NewOAuthClient(
		WithClientID("test-id"),
		WithClientSecret("test-secret"),
		WithScopes(DefaultV01Scopes()),
		WithCallbackPort(port),
		WithAuthURL("http://localhost:1/unused"),
		WithTokenURL("http://localhost:1/unused"),
		WithFlowTimeout(200*time.Millisecond),
	)
	// Don't open browser -- let it time out.
	client.OpenBrowserFunc = func(url string) error { return nil }

	_, err := client.DesktopFlow(context.Background())
	if !errors.Is(err, ErrFlowTimeout) {
		t.Errorf("error = %v, want ErrFlowTimeout", err)
	}
}

func TestDesktopFlow_ContextCancelled(t *testing.T) {
	port := findFreePort()

	ctx, cancel := context.WithCancel(context.Background())

	client := NewOAuthClient(
		WithClientID("test-id"),
		WithClientSecret("test-secret"),
		WithScopes(DefaultV01Scopes()),
		WithCallbackPort(port),
		WithAuthURL("http://localhost:1/unused"),
		WithTokenURL("http://localhost:1/unused"),
		WithFlowTimeout(30*time.Second),
	)
	client.OpenBrowserFunc = func(url string) error {
		// Cancel context shortly after browser "opens".
		go func() {
			time.Sleep(100 * time.Millisecond)
			cancel()
		}()
		return nil
	}

	_, err := client.DesktopFlow(ctx)
	if !errors.Is(err, ErrFlowCancelled) {
		t.Errorf("error = %v, want ErrFlowCancelled", err)
	}
}

func TestDesktopFlow_PortInUse(t *testing.T) {
	// Bind a port first.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port

	client := NewOAuthClient(
		WithClientID("test-id"),
		WithClientSecret("test-secret"),
		WithScopes(DefaultV01Scopes()),
		WithCallbackPort(port),
	)
	client.OpenBrowserFunc = func(url string) error { return nil }

	_, err = client.DesktopFlow(context.Background())
	if !errors.Is(err, ErrPortInUse) {
		t.Errorf("error = %v, want ErrPortInUse", err)
	}
}

func TestDesktopFlow_DuplicateCallback(t *testing.T) {
	tokenSrv := mockTokenServer(t, true)
	defer tokenSrv.Close()

	port := findFreePort()

	authSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		state := r.URL.Query().Get("state")
		redirectURI := r.URL.Query().Get("redirect_uri")
		callbackURL := redirectURI + "?code=mock-code&state=" + state
		// Send TWO redirects (simulating duplicate callback).
		go func() {
			http.Get(callbackURL)
			http.Get(callbackURL) // duplicate -- should be ignored
		}()
		w.WriteHeader(http.StatusOK)
	}))
	defer authSrv.Close()

	client := NewOAuthClient(
		WithClientID("test-id"),
		WithClientSecret("test-secret"),
		WithScopes(DefaultV01Scopes()),
		WithCallbackPort(port),
		WithAuthURL(authSrv.URL),
		WithTokenURL(tokenSrv.URL),
		WithFlowTimeout(10*time.Second),
	)
	client.OpenBrowserFunc = func(authURL string) error {
		go http.Get(authURL)
		return nil
	}

	// Should succeed despite duplicate callback.
	token, err := client.DesktopFlow(context.Background())
	if err != nil {
		t.Fatalf("DesktopFlow error (duplicate callback): %v", err)
	}
	if token.AccessToken != "mock-access-token" {
		t.Errorf("AccessToken = %q", token.AccessToken)
	}
}

func TestDesktopFlow_TokenExchangeError(t *testing.T) {
	port := findFreePort()

	badTokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":"invalid_client"}`)
	}))
	defer badTokenSrv.Close()

	authSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		state := r.URL.Query().Get("state")
		redirectURI := r.URL.Query().Get("redirect_uri")
		http.Redirect(w, r, redirectURI+"?code=mock-code&state="+state, http.StatusFound)
	}))
	defer authSrv.Close()

	client := NewOAuthClient(
		WithClientID("test-id"),
		WithClientSecret("wrong-secret"),
		WithScopes(DefaultV01Scopes()),
		WithCallbackPort(port),
		WithAuthURL(authSrv.URL),
		WithTokenURL(badTokenSrv.URL),
		WithFlowTimeout(10*time.Second),
	)
	client.OpenBrowserFunc = func(authURL string) error {
		go http.Get(authURL)
		return nil
	}

	_, err := client.DesktopFlow(context.Background())
	if err == nil {
		t.Error("expected error for bad token exchange")
	}
	if !strings.Contains(err.Error(), "invalid_client") {
		t.Errorf("error = %v, want to contain invalid_client", err)
	}
}

func TestDesktopFlow_OtherError(t *testing.T) {
	port := findFreePort()

	authSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirectURI := r.URL.Query().Get("redirect_uri")
		http.Redirect(w, r, redirectURI+"?error=server_error", http.StatusFound)
	}))
	defer authSrv.Close()

	client := NewOAuthClient(
		WithClientID("test-id"),
		WithClientSecret("test-secret"),
		WithScopes(DefaultV01Scopes()),
		WithCallbackPort(port),
		WithAuthURL(authSrv.URL),
		WithTokenURL("http://unused"),
		WithFlowTimeout(10*time.Second),
	)
	client.OpenBrowserFunc = func(authURL string) error {
		go http.Get(authURL)
		return nil
	}

	_, err := client.DesktopFlow(context.Background())
	if err == nil {
		t.Error("expected error for server_error")
	}
	if !strings.Contains(err.Error(), "server_error") {
		t.Errorf("error = %v, want to contain server_error", err)
	}
}

// --- PKCE tests ---

func TestGenerateCodeVerifier(t *testing.T) {
	v, err := generateCodeVerifier()
	if err != nil {
		t.Fatalf("generateCodeVerifier error: %v", err)
	}
	// 48 bytes -> 64 base64url chars
	if len(v) != 64 {
		t.Errorf("verifier length = %d, want 64", len(v))
	}
	// Should be URL-safe (no +, /, =).
	if strings.ContainsAny(v, "+/=") {
		t.Errorf("verifier contains non-URL-safe characters: %q", v)
	}
}

func TestGenerateCodeVerifier_Unique(t *testing.T) {
	v1, _ := generateCodeVerifier()
	v2, _ := generateCodeVerifier()
	if v1 == v2 {
		t.Error("two verifiers should not be identical")
	}
}

func TestComputeCodeChallenge(t *testing.T) {
	// Known test vector: verifier "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	// Challenge should be base64url(SHA256(verifier))
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	challenge := computeCodeChallenge(verifier)

	if challenge == "" {
		t.Error("challenge should not be empty")
	}
	if strings.ContainsAny(challenge, "+/=") {
		t.Errorf("challenge contains non-URL-safe characters: %q", challenge)
	}
	// S256 challenge is always 43 characters (32 bytes base64url encoded).
	if len(challenge) != 43 {
		t.Errorf("challenge length = %d, want 43", len(challenge))
	}
}

func TestGenerateState(t *testing.T) {
	s, err := generateState()
	if err != nil {
		t.Fatalf("generateState error: %v", err)
	}
	// 32 bytes -> 43 base64url chars
	if len(s) != 43 {
		t.Errorf("state length = %d, want 43", len(s))
	}
}

func TestGenerateState_Unique(t *testing.T) {
	s1, _ := generateState()
	s2, _ := generateState()
	if s1 == s2 {
		t.Error("two state values should not be identical")
	}
}

// --- Auth URL tests ---

func TestAuthURL_ContainsRequiredParams(t *testing.T) {
	client := NewOAuthClient(
		WithClientID("my-client-id"),
		WithClientSecret("my-secret"),
		WithScopes(V01WithDrive()),
	)

	authURL := client.AuthURL("test-state", "test-challenge", "http://localhost:8085/callback")
	u, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("parse auth URL: %v", err)
	}

	checks := map[string]string{
		"client_id":              "my-client-id",
		"response_type":         "code",
		"state":                 "test-state",
		"code_challenge":        "test-challenge",
		"code_challenge_method": "S256",
		"access_type":           "offline",
		"include_granted_scopes": "true",
	}
	for key, want := range checks {
		got := u.Query().Get(key)
		if got != want {
			t.Errorf("param %s = %q, want %q", key, got, want)
		}
	}

	// Check scope contains both expected scopes.
	scope := u.Query().Get("scope")
	if !strings.Contains(scope, ScopeUserInfoEmail) {
		t.Errorf("scope missing userinfo.email: %q", scope)
	}
	if !strings.Contains(scope, ScopeDriveReadOnly) {
		t.Errorf("scope missing drive.readonly: %q", scope)
	}
}
