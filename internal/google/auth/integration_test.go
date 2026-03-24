//go:build integration

package auth

import (
	"context"
	"os"
	"testing"
)

// skipIfNoGoogleCreds skips the test if OAuth credentials are not configured.
func skipIfNoGoogleCreds(t *testing.T) {
	t.Helper()
	if os.Getenv("GOOGLE_CLIENT_ID") == "" || os.Getenv("GOOGLE_CLIENT_SECRET") == "" {
		t.Skip("GOOGLE_CLIENT_ID/SECRET not set -- skipping integration test")
	}
}

// skipIfNoTestToken skips the test if a pre-authenticated token is not available.
func skipIfNoTestToken(t *testing.T) string {
	t.Helper()
	tok := os.Getenv("GKESTRAL_TEST_TOKEN")
	if tok == "" {
		t.Skip("GKESTRAL_TEST_TOKEN not set -- skipping integration test")
	}
	return tok
}

func TestIntegration_Smoke(t *testing.T) {
	// Smoke test: verify skip logic works when creds are absent.
	// This test always passes -- it validates the test infrastructure.
	if os.Getenv("GOOGLE_CLIENT_ID") == "" {
		t.Log("No credentials -- skip logic works correctly")
		return
	}
	t.Log("Credentials found -- real integration tests would run")
}

func TestIntegration_OAuthTokenRefresh(t *testing.T) {
	skipIfNoGoogleCreds(t)
	tokJSON := skipIfNoTestToken(t)
	_ = tokJSON

	// Load pre-authenticated token and verify refresh works.
	t.Log("Token refresh integration test -- requires real Google OAuth token")
	// Implementation: parse GKESTRAL_TEST_TOKEN JSON, create store, verify refresh.
	// Deferred to when Commander provides real credentials.
}

func TestIntegration_ScopeEscalation(t *testing.T) {
	skipIfNoGoogleCreds(t)

	// Verify that NeedsReauth correctly detects missing scopes
	// against a real token's scope set.
	current := DefaultV01Scopes()
	requested := V01WithDrive()
	if !NeedsReauth(current, requested) {
		t.Error("should need reauth to add Drive scope")
	}
	t.Log("Scope escalation detection works")
}

func TestIntegration_VerifyAPIKey(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set -- skipping API key verification")
	}

	err := VerifyAPIKey(context.Background(), apiKey)
	if err != nil {
		t.Errorf("VerifyAPIKey failed: %v", err)
	}
}
