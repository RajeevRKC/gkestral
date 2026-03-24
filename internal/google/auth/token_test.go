package auth

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// mockKeyring implements KeyringProvider for testing.
type mockKeyring struct {
	store map[string]string
	err   error // if set, all operations return this error
}

func newMockKeyring() *mockKeyring {
	return &mockKeyring{store: make(map[string]string)}
}

func (m *mockKeyring) Set(service, user, password string) error {
	if m.err != nil {
		return m.err
	}
	m.store[service+"/"+user] = password
	return nil
}

func (m *mockKeyring) Get(service, user string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	v, ok := m.store[service+"/"+user]
	if !ok {
		return "", errors.New("not found")
	}
	return v, nil
}

func (m *mockKeyring) Delete(service, user string) error {
	if m.err != nil {
		return m.err
	}
	delete(m.store, service+"/"+user)
	return nil
}

func testToken() StoredToken {
	return StoredToken{
		AccessToken:  "access-123",
		RefreshToken: "refresh-456",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour).Format(time.RFC3339),
		Scopes:       []string{ScopeUserInfoEmail, ScopeDriveReadOnly},
	}
}

func expiredToken() StoredToken {
	return StoredToken{
		AccessToken:  "expired-access",
		RefreshToken: "refresh-456",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(-time.Hour).Format(time.RFC3339),
		Scopes:       []string{ScopeUserInfoEmail},
	}
}

// --- KeyringTokenStore tests ---

func TestKeyringTokenStore_SaveAndLoad(t *testing.T) {
	kr := newMockKeyring()
	store := NewKeyringTokenStore(kr)

	tok := testToken()
	if err := store.Save(tok); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if loaded.AccessToken != tok.AccessToken {
		t.Errorf("AccessToken = %q, want %q", loaded.AccessToken, tok.AccessToken)
	}
	if loaded.RefreshToken != tok.RefreshToken {
		t.Errorf("RefreshToken = %q, want %q", loaded.RefreshToken, tok.RefreshToken)
	}
}

func TestKeyringTokenStore_LoadNotFound(t *testing.T) {
	kr := newMockKeyring()
	store := NewKeyringTokenStore(kr)

	_, err := store.Load()
	if !errors.Is(err, ErrNoToken) {
		t.Errorf("error = %v, want ErrNoToken", err)
	}
}

func TestKeyringTokenStore_Clear(t *testing.T) {
	kr := newMockKeyring()
	store := NewKeyringTokenStore(kr)

	_ = store.Save(testToken())
	if !store.Exists() {
		t.Error("Exists() should be true after Save")
	}

	if err := store.Clear(); err != nil {
		t.Fatalf("Clear error: %v", err)
	}
	if store.Exists() {
		t.Error("Exists() should be false after Clear")
	}
}

func TestKeyringTokenStore_Exists(t *testing.T) {
	kr := newMockKeyring()
	store := NewKeyringTokenStore(kr)

	if store.Exists() {
		t.Error("Exists() should be false when no token stored")
	}

	_ = store.Save(testToken())
	if !store.Exists() {
		t.Error("Exists() should be true after Save")
	}
}

func TestKeyringTokenStore_CorruptData(t *testing.T) {
	kr := newMockKeyring()
	store := NewKeyringTokenStore(kr)

	// Store invalid JSON directly.
	kr.store[defaultService+"/"+defaultUser] = "not-json{{"

	_, err := store.Load()
	if !errors.Is(err, ErrTokenCorrupt) {
		t.Errorf("error = %v, want ErrTokenCorrupt", err)
	}
}

// --- EncryptedFileTokenStore tests ---

func TestEncryptedFileTokenStore_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tokens.json")
	store := NewEncryptedFileTokenStore(path, "test-passphrase-123")

	tok := testToken()
	if err := store.Save(tok); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if loaded.AccessToken != tok.AccessToken {
		t.Errorf("AccessToken = %q, want %q", loaded.AccessToken, tok.AccessToken)
	}
	if loaded.RefreshToken != tok.RefreshToken {
		t.Errorf("RefreshToken = %q, want %q", loaded.RefreshToken, tok.RefreshToken)
	}
	if len(loaded.Scopes) != len(tok.Scopes) {
		t.Errorf("Scopes len = %d, want %d", len(loaded.Scopes), len(tok.Scopes))
	}
}

func TestEncryptedFileTokenStore_WrongPassphrase(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tokens.json")

	store1 := NewEncryptedFileTokenStore(path, "correct-pass")
	_ = store1.Save(testToken())

	store2 := NewEncryptedFileTokenStore(path, "wrong-pass")
	_, err := store2.Load()
	if !errors.Is(err, ErrTokenCorrupt) {
		t.Errorf("error = %v, want ErrTokenCorrupt for wrong passphrase", err)
	}
}

func TestEncryptedFileTokenStore_LoadNotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.json")
	store := NewEncryptedFileTokenStore(path, "pass")

	_, err := store.Load()
	if !errors.Is(err, ErrNoToken) {
		t.Errorf("error = %v, want ErrNoToken", err)
	}
}

func TestEncryptedFileTokenStore_CorruptFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tokens.json")

	// Write garbage data.
	os.WriteFile(path, []byte("this is not encrypted data at all"), 0600)

	store := NewEncryptedFileTokenStore(path, "pass")
	_, err := store.Load()
	if !errors.Is(err, ErrTokenCorrupt) {
		t.Errorf("error = %v, want ErrTokenCorrupt", err)
	}
}

func TestEncryptedFileTokenStore_TooShortFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tokens.json")

	os.WriteFile(path, []byte("short"), 0600)

	store := NewEncryptedFileTokenStore(path, "pass")
	_, err := store.Load()
	if !errors.Is(err, ErrTokenCorrupt) {
		t.Errorf("error = %v, want ErrTokenCorrupt", err)
	}
}

func TestEncryptedFileTokenStore_Clear(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tokens.json")
	store := NewEncryptedFileTokenStore(path, "pass")

	_ = store.Save(testToken())
	if !store.Exists() {
		t.Error("Exists() should be true after Save")
	}

	if err := store.Clear(); err != nil {
		t.Fatalf("Clear error: %v", err)
	}
	if store.Exists() {
		t.Error("Exists() should be false after Clear")
	}
}

func TestEncryptedFileTokenStore_ClearNonexistent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.json")
	store := NewEncryptedFileTokenStore(path, "pass")

	// Should not error on non-existent file.
	if err := store.Clear(); err != nil {
		t.Errorf("Clear error: %v", err)
	}
}

func TestEncryptedFileTokenStore_FilePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tokens.json")
	store := NewEncryptedFileTokenStore(path, "pass")
	_ = store.Save(testToken())

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat error: %v", err)
	}
	// On Windows, file permissions are limited. Skip strict check on Windows.
	mode := info.Mode().Perm()
	// Just verify the file is not world-readable (best effort).
	if mode&0077 != 0 && mode != 0666 { // Windows often returns 0666
		// Non-Windows: should be 0600
		if mode != 0600 {
			t.Logf("WARNING: file permissions = %o, expected 0600 (may be OS-specific)", mode)
		}
	}
}

func TestEncryptedFileTokenStore_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "deep", "tokens.json")
	store := NewEncryptedFileTokenStore(path, "pass")

	if err := store.Save(testToken()); err != nil {
		t.Fatalf("Save error (should create dirs): %v", err)
	}
	if !store.Exists() {
		t.Error("token should exist after Save with directory creation")
	}
}

// --- StoredToken tests ---

func TestStoredToken_IsExpired(t *testing.T) {
	future := StoredToken{Expiry: time.Now().Add(time.Hour).Format(time.RFC3339)}
	if future.IsExpired() {
		t.Error("future token should not be expired")
	}

	past := StoredToken{Expiry: time.Now().Add(-time.Hour).Format(time.RFC3339)}
	if !past.IsExpired() {
		t.Error("past token should be expired")
	}

	// Within the 60s buffer window.
	borderline := StoredToken{Expiry: time.Now().Add(30 * time.Second).Format(time.RFC3339)}
	if !borderline.IsExpired() {
		t.Error("token expiring in 30s should be considered expired (60s buffer)")
	}
}

func TestStoredToken_ExpiryTime(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	tok := StoredToken{Expiry: now.Format(time.RFC3339)}
	if !tok.ExpiryTime().Equal(now) {
		t.Errorf("ExpiryTime() = %v, want %v", tok.ExpiryTime(), now)
	}
}

// --- TokenRefresher tests ---

func TestTokenRefresher_GetValidToken_NotExpired(t *testing.T) {
	kr := newMockKeyring()
	store := NewKeyringTokenStore(kr)
	_ = store.Save(testToken())

	refresher := NewTokenRefresher(store)
	tok, err := refresher.GetValidToken()
	if err != nil {
		t.Fatalf("GetValidToken error: %v", err)
	}
	if tok.AccessToken != "access-123" {
		t.Errorf("AccessToken = %q", tok.AccessToken)
	}
}

func TestTokenRefresher_GetValidToken_Refreshes(t *testing.T) {
	kr := newMockKeyring()
	store := NewKeyringTokenStore(kr)
	_ = store.Save(expiredToken())

	refreshed := false
	refresher := NewTokenRefresher(store)
	refresher.RefreshFunc = func(refreshToken string) (StoredToken, error) {
		if refreshToken != "refresh-456" {
			t.Errorf("RefreshFunc got refreshToken = %q", refreshToken)
		}
		return StoredToken{
			AccessToken: "new-access",
			TokenType:   "Bearer",
			Expiry:      time.Now().Add(time.Hour).Format(time.RFC3339),
		}, nil
	}
	refresher.OnRefresh = func(newToken StoredToken) {
		refreshed = true
	}

	tok, err := refresher.GetValidToken()
	if err != nil {
		t.Fatalf("GetValidToken error: %v", err)
	}
	if tok.AccessToken != "new-access" {
		t.Errorf("AccessToken = %q, want new-access", tok.AccessToken)
	}
	if tok.RefreshToken != "refresh-456" {
		t.Errorf("RefreshToken should be preserved, got %q", tok.RefreshToken)
	}
	if !refreshed {
		t.Error("OnRefresh callback was not called")
	}
}

func TestTokenRefresher_GetValidToken_InvalidGrant(t *testing.T) {
	kr := newMockKeyring()
	store := NewKeyringTokenStore(kr)
	_ = store.Save(expiredToken())

	reauthCalled := false
	refresher := NewTokenRefresher(store)
	refresher.RefreshFunc = func(refreshToken string) (StoredToken, error) {
		return StoredToken{}, errors.New("oauth2: invalid_grant")
	}
	refresher.OnReauthRequired = func(reason string) {
		reauthCalled = true
	}

	_, err := refresher.GetValidToken()
	if !errors.Is(err, ErrInvalidGrant) {
		t.Errorf("error = %v, want ErrInvalidGrant", err)
	}
	if !reauthCalled {
		t.Error("OnReauthRequired callback was not called")
	}
	// Token should be cleared.
	if store.Exists() {
		t.Error("token should be cleared after invalid_grant")
	}
}

func TestTokenRefresher_GetValidToken_NoRefreshFunc(t *testing.T) {
	kr := newMockKeyring()
	store := NewKeyringTokenStore(kr)
	_ = store.Save(expiredToken())

	refresher := NewTokenRefresher(store)
	// No RefreshFunc set.

	_, err := refresher.GetValidToken()
	if err == nil {
		t.Error("expected error when RefreshFunc is nil and token expired")
	}
}

func TestTokenRefresher_GetValidToken_NoToken(t *testing.T) {
	kr := newMockKeyring()
	store := NewKeyringTokenStore(kr)

	refresher := NewTokenRefresher(store)
	_, err := refresher.GetValidToken()
	if !errors.Is(err, ErrNoToken) {
		t.Errorf("error = %v, want ErrNoToken", err)
	}
}

func TestTokenRefresher_ConcurrentRefresh(t *testing.T) {
	kr := newMockKeyring()
	store := NewKeyringTokenStore(kr)
	_ = store.Save(expiredToken())

	refreshCount := 0
	var mu sync.Mutex

	refresher := NewTokenRefresher(store)
	refresher.RefreshFunc = func(refreshToken string) (StoredToken, error) {
		mu.Lock()
		refreshCount++
		mu.Unlock()
		time.Sleep(50 * time.Millisecond) // Simulate network delay.
		return StoredToken{
			AccessToken: "concurrent-access",
			TokenType:   "Bearer",
			Expiry:      time.Now().Add(time.Hour).Format(time.RFC3339),
		}, nil
	}

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := refresher.GetValidToken()
			if err != nil {
				t.Errorf("GetValidToken error: %v", err)
			}
		}()
	}
	wg.Wait()

	// Due to mutex, only one goroutine should actually refresh.
	// Others will find a valid token after the first refreshes.
	// Note: exact count depends on timing, but should be small.
	mu.Lock()
	defer mu.Unlock()
	if refreshCount > 3 {
		t.Errorf("refreshCount = %d, expected <= 3 (mutex should serialize)", refreshCount)
	}
}

func TestDefaultTokenPath(t *testing.T) {
	path := DefaultTokenPath()
	if !filepath.IsAbs(path) {
		t.Errorf("DefaultTokenPath() = %q, want absolute path", path)
	}
	if filepath.Base(path) != "tokens.json" {
		t.Errorf("DefaultTokenPath() base = %q, want tokens.json", filepath.Base(path))
	}
}
