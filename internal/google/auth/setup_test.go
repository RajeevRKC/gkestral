package auth

import (
	"testing"
)

func TestCheckSetup_NoCredentials(t *testing.T) {
	t.Setenv("GKESTRAL_CLIENT_ID", "")
	t.Setenv("GKESTRAL_CLIENT_SECRET", "")
	old := BundledCredentials
	BundledCredentials = ClientCredentials{}
	defer func() { BundledCredentials = old }()

	status := CheckSetup(nil)
	if status.HasCredentials {
		t.Error("HasCredentials should be false")
	}
	if status.HasToken {
		t.Error("HasToken should be false with nil store")
	}
}

func TestCheckSetup_WithEnvCredentials(t *testing.T) {
	t.Setenv("GKESTRAL_CLIENT_ID", "env-id")
	t.Setenv("GKESTRAL_CLIENT_SECRET", "env-secret")

	status := CheckSetup(nil)
	if !status.HasCredentials {
		t.Error("HasCredentials should be true")
	}
}

func TestCheckSetup_WithTokenStore(t *testing.T) {
	t.Setenv("GKESTRAL_CLIENT_ID", "id")
	t.Setenv("GKESTRAL_CLIENT_SECRET", "secret")

	kr := newMockKeyring()
	store := NewKeyringTokenStore(kr)
	_ = store.Save(testToken())

	status := CheckSetup(store)
	if !status.HasToken {
		t.Error("HasToken should be true")
	}
	if status.TokenExpired {
		t.Error("TokenExpired should be false for future token")
	}
	if status.ScopesGranted.Len() != 2 {
		t.Errorf("ScopesGranted.Len() = %d, want 2", status.ScopesGranted.Len())
	}
}

func TestCheckSetup_ExpiredToken(t *testing.T) {
	t.Setenv("GKESTRAL_CLIENT_ID", "id")
	t.Setenv("GKESTRAL_CLIENT_SECRET", "secret")

	kr := newMockKeyring()
	store := NewKeyringTokenStore(kr)
	_ = store.Save(expiredToken())

	status := CheckSetup(store)
	if !status.TokenExpired {
		t.Error("TokenExpired should be true for expired token")
	}
}

func TestGCloudAvailable(t *testing.T) {
	// Just verify it doesn't panic -- actual result depends on system.
	_ = GCloudAvailable()
}

func TestGetConsoleURLs_WithProject(t *testing.T) {
	urls := GetConsoleURLs("my-project")
	if urls.CreateProject == "" {
		t.Error("CreateProject URL should not be empty")
	}
	if urls.EnableAPIs == "" {
		t.Error("EnableAPIs URL should not be empty")
	}
	if urls.OAuthConsent == "" {
		t.Error("OAuthConsent URL should not be empty")
	}
	if urls.CreateOAuthCred == "" {
		t.Error("CreateOAuthCred URL should not be empty")
	}
}

func TestGetConsoleURLs_NoProject(t *testing.T) {
	urls := GetConsoleURLs("")
	if urls.CreateProject == "" {
		t.Error("CreateProject URL should not be empty")
	}
}

func TestCheckSetup_CredentialSourceEnv(t *testing.T) {
	t.Setenv("GKESTRAL_CLIENT_ID", "env-id")
	t.Setenv("GKESTRAL_CLIENT_SECRET", "env-secret")

	kr := newMockKeyring()
	store := NewKeyringTokenStore(kr)
	status := CheckSetup(store)
	if !status.HasCredentials {
		t.Error("HasCredentials should be true")
	}
	if status.HasToken {
		t.Error("HasToken should be false (no token saved)")
	}
}

func TestCheckSetup_BundledCredentials(t *testing.T) {
	t.Setenv("GKESTRAL_CLIENT_ID", "")
	t.Setenv("GKESTRAL_CLIENT_SECRET", "")
	old := BundledCredentials
	BundledCredentials = ClientCredentials{ClientID: "bundled-id", ClientSecret: "bundled-secret"}
	defer func() { BundledCredentials = old }()

	status := CheckSetup(nil)
	if !status.HasCredentials {
		t.Error("HasCredentials should be true with bundled creds")
	}
	if status.CredentialSource != "bundled" {
		t.Errorf("CredentialSource = %q, want bundled", status.CredentialSource)
	}
}

func TestCheckSetup_NilStore(t *testing.T) {
	t.Setenv("GKESTRAL_CLIENT_ID", "")
	t.Setenv("GKESTRAL_CLIENT_SECRET", "")
	old := BundledCredentials
	BundledCredentials = ClientCredentials{}
	defer func() { BundledCredentials = old }()

	status := CheckSetup(nil)
	if status.HasCredentials {
		t.Error("HasCredentials should be false")
	}
	if status.HasToken {
		t.Error("HasToken should be false")
	}
	if status.CredentialSource != "" {
		t.Errorf("CredentialSource = %q, want empty", status.CredentialSource)
	}
}
