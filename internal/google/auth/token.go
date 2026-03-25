package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/pbkdf2"
)

// Sentinel errors for token operations.
var (
	ErrNoToken             = errors.New("auth: no stored token")
	ErrPassphraseRequired  = errors.New("auth: OS keyring unavailable, passphrase required")
	ErrTokenCorrupt        = errors.New("auth: stored token data is corrupt")
	ErrInvalidGrant        = errors.New("auth: refresh token revoked or expired (invalid_grant)")
)

// StoredToken is the on-disk representation of an OAuth token + granted scopes.
type StoredToken struct {
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	TokenType    string   `json:"token_type"`
	Expiry       string   `json:"expiry"` // RFC3339
	Scopes       []string `json:"scopes"`
}

// Token returns the expiry as a time.Time.
func (st StoredToken) ExpiryTime() time.Time {
	t, _ := time.Parse(time.RFC3339, st.Expiry)
	return t
}

// IsExpired reports whether the access token has expired (with 60s buffer).
func (st StoredToken) IsExpired() bool {
	return time.Now().After(st.ExpiryTime().Add(-60 * time.Second))
}

// TokenStore is the interface for persisting and retrieving OAuth tokens.
type TokenStore interface {
	Save(token StoredToken) error
	Load() (StoredToken, error)
	Clear() error
	Exists() bool
}

// KeyringProvider abstracts OS-level keyring operations for testing.
type KeyringProvider interface {
	Set(service, user, password string) error
	Get(service, user string) (string, error)
	Delete(service, user string) error
}

// KeyringTokenStore stores tokens in the OS-level credential manager.
type KeyringTokenStore struct {
	provider KeyringProvider
	service  string
	user     string
}

const (
	defaultService = "gkestral-oauth"
	defaultUser    = "default"
)

// NewKeyringTokenStore creates a token store backed by the OS keyring.
func NewKeyringTokenStore(provider KeyringProvider) *KeyringTokenStore {
	return &KeyringTokenStore{
		provider: provider,
		service:  defaultService,
		user:     defaultUser,
	}
}

func (k *KeyringTokenStore) Save(token StoredToken) error {
	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("auth: marshal token: %w", err)
	}
	return k.provider.Set(k.service, k.user, string(data))
}

func (k *KeyringTokenStore) Load() (StoredToken, error) {
	data, err := k.provider.Get(k.service, k.user)
	if err != nil {
		return StoredToken{}, ErrNoToken
	}
	var token StoredToken
	if err := json.Unmarshal([]byte(data), &token); err != nil {
		return StoredToken{}, ErrTokenCorrupt
	}
	return token, nil
}

func (k *KeyringTokenStore) Clear() error {
	return k.provider.Delete(k.service, k.user)
}

func (k *KeyringTokenStore) Exists() bool {
	_, err := k.provider.Get(k.service, k.user)
	return err == nil
}

// EncryptedFileTokenStore stores tokens in an AES-256-GCM encrypted JSON file.
// Used as fallback when OS keyring is unavailable.
type EncryptedFileTokenStore struct {
	path       string
	passphrase string
}

// PBKDF2 parameters.
const (
	pbkdf2Iterations = 100_000
	pbkdf2KeyLen     = 32 // AES-256
	saltLen          = 16
)

// NewEncryptedFileTokenStore creates a file-based token store with PBKDF2-derived
// AES-256-GCM encryption. The passphrase is provided by the application layer.
func NewEncryptedFileTokenStore(path, passphrase string) *EncryptedFileTokenStore {
	return &EncryptedFileTokenStore{path: path, passphrase: passphrase}
}

// DefaultTokenPath returns the default token storage location.
func DefaultTokenPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".gkestral", "tokens.json")
}

func (f *EncryptedFileTokenStore) deriveKey(salt []byte) []byte {
	return pbkdf2.Key([]byte(f.passphrase), salt, pbkdf2Iterations, pbkdf2KeyLen, sha256.New)
}

func (f *EncryptedFileTokenStore) Save(token StoredToken) error {
	plaintext, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("auth: marshal token: %w", err)
	}

	salt := make([]byte, saltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return fmt.Errorf("auth: generate salt: %w", err)
	}

	key := f.deriveKey(salt)
	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("auth: create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("auth: create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return fmt.Errorf("auth: generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	// File format: salt (16 bytes) + nonce+ciphertext
	data := append(salt, ciphertext...)

	dir := filepath.Dir(f.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("auth: create directory: %w", err)
	}
	return os.WriteFile(f.path, data, 0600)
}

func (f *EncryptedFileTokenStore) Load() (StoredToken, error) {
	data, err := os.ReadFile(f.path)
	if err != nil {
		if os.IsNotExist(err) {
			return StoredToken{}, ErrNoToken
		}
		return StoredToken{}, fmt.Errorf("auth: read token file: %w", err)
	}

	if len(data) < saltLen+12 { // salt + minimum GCM nonce
		return StoredToken{}, ErrTokenCorrupt
	}

	salt := data[:saltLen]
	ciphertext := data[saltLen:]

	key := f.deriveKey(salt)
	block, err := aes.NewCipher(key)
	if err != nil {
		return StoredToken{}, ErrTokenCorrupt
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return StoredToken{}, ErrTokenCorrupt
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return StoredToken{}, ErrTokenCorrupt
	}

	nonce := ciphertext[:nonceSize]
	ct := ciphertext[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return StoredToken{}, ErrTokenCorrupt
	}

	var token StoredToken
	if err := json.Unmarshal(plaintext, &token); err != nil {
		return StoredToken{}, ErrTokenCorrupt
	}
	return token, nil
}

func (f *EncryptedFileTokenStore) Clear() error {
	if err := os.Remove(f.path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("auth: remove token file: %w", err)
	}
	return nil
}

func (f *EncryptedFileTokenStore) Exists() bool {
	_, err := os.Stat(f.path)
	return err == nil
}

// TokenRefresher wraps a TokenStore and provides thread-safe auto-refresh.
type TokenRefresher struct {
	store TokenStore
	mu    sync.Mutex

	// RefreshFunc is called to obtain a new access token using the refresh token.
	// The caller provides this — typically wrapping the OAuth token endpoint.
	RefreshFunc func(refreshToken string) (StoredToken, error)

	// OnRefresh is called after a successful token refresh.
	OnRefresh func(newToken StoredToken)

	// OnReauthRequired is called when the refresh token is permanently invalid
	// (e.g., invalid_grant). The caller should re-trigger the OAuth desktop flow.
	OnReauthRequired func(reason string)
}

// NewTokenRefresher creates a TokenRefresher that manages auto-refresh lifecycle.
func NewTokenRefresher(store TokenStore) *TokenRefresher {
	return &TokenRefresher{store: store}
}

// GetValidToken returns a non-expired token, refreshing if necessary.
// Thread-safe: only one goroutine refreshes at a time. Callbacks are
// invoked AFTER the mutex is released to prevent deadlock.
func (r *TokenRefresher) GetValidToken() (StoredToken, error) {
	r.mu.Lock()

	token, err := r.store.Load()
	if err != nil {
		r.mu.Unlock()
		return StoredToken{}, err
	}

	if !token.IsExpired() {
		r.mu.Unlock()
		return token, nil
	}

	if r.RefreshFunc == nil {
		r.mu.Unlock()
		return StoredToken{}, errors.New("auth: token expired and no RefreshFunc configured")
	}

	newToken, err := r.RefreshFunc(token.RefreshToken)
	if err != nil {
		// Check for invalid_grant — permanent failure.
		if isInvalidGrant(err) {
			_ = r.store.Clear()
			// Capture callback before unlocking.
			reauthCb := r.OnReauthRequired
			r.mu.Unlock()
			if reauthCb != nil {
				reauthCb("refresh token revoked or expired")
			}
			return StoredToken{}, ErrInvalidGrant
		}
		r.mu.Unlock()
		return StoredToken{}, fmt.Errorf("auth: refresh failed: %w", err)
	}

	// Preserve refresh token if not rotated.
	if newToken.RefreshToken == "" {
		newToken.RefreshToken = token.RefreshToken
	}

	if err := r.store.Save(newToken); err != nil {
		r.mu.Unlock()
		return StoredToken{}, fmt.Errorf("auth: persist refreshed token: %w", err)
	}

	// Capture callback before unlocking.
	refreshCb := r.OnRefresh
	r.mu.Unlock()

	if refreshCb != nil {
		refreshCb(newToken)
	}

	return newToken, nil
}

// isInvalidGrant checks if the error represents an OAuth invalid_grant response.
func isInvalidGrant(err error) bool {
	return errors.Is(err, ErrInvalidGrant) ||
		(err != nil && strings.Contains(err.Error(), "invalid_grant"))
}
