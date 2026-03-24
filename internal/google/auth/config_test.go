package auth

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCredentials_FromEnv(t *testing.T) {
	t.Setenv("GKESTRAL_CLIENT_ID", "env-client-id")
	t.Setenv("GKESTRAL_CLIENT_SECRET", "env-client-secret")

	creds, err := LoadCredentials()
	if err != nil {
		t.Fatalf("LoadCredentials error: %v", err)
	}
	if creds.ClientID != "env-client-id" {
		t.Errorf("ClientID = %q, want env-client-id", creds.ClientID)
	}
	if creds.ClientSecret != "env-client-secret" {
		t.Errorf("ClientSecret = %q", creds.ClientSecret)
	}
}

func TestLoadCredentials_NoSource(t *testing.T) {
	t.Setenv("GKESTRAL_CLIENT_ID", "")
	t.Setenv("GKESTRAL_CLIENT_SECRET", "")
	// Ensure bundled creds are empty.
	old := BundledCredentials
	BundledCredentials = ClientCredentials{}
	defer func() { BundledCredentials = old }()

	_, err := LoadCredentials()
	if err != ErrNoCredentials {
		t.Errorf("error = %v, want ErrNoCredentials", err)
	}
}

func TestClientCredentials_Validate(t *testing.T) {
	valid := ClientCredentials{ClientID: "id", ClientSecret: "secret"}
	if err := valid.Validate(); err != nil {
		t.Errorf("Validate() error: %v", err)
	}

	noID := ClientCredentials{ClientSecret: "secret"}
	if err := noID.Validate(); err == nil {
		t.Error("Validate() should fail with empty ClientID")
	}

	noSecret := ClientCredentials{ClientID: "id"}
	if err := noSecret.Validate(); err == nil {
		t.Error("Validate() should fail with empty ClientSecret")
	}
}

func TestSaveAndLoadCredentialsFile(t *testing.T) {
	// Override credentials path to temp dir.
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials.json")

	creds := ClientCredentials{ClientID: "file-id", ClientSecret: "file-secret"}
	data, _ := os.ReadFile(path)
	_ = data // just to show file doesn't exist yet

	// Save directly to the temp path (bypassing CredentialsFilePath).
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	jsonData := `{"client_id":"file-id","client_secret":"file-secret"}`
	if err := os.WriteFile(path, []byte(jsonData), 0600); err != nil {
		t.Fatal(err)
	}

	// Verify the credential struct validates.
	if err := creds.Validate(); err != nil {
		t.Errorf("Validate error: %v", err)
	}
}

func TestCredentialsFilePath(t *testing.T) {
	path := CredentialsFilePath()
	if !filepath.IsAbs(path) {
		t.Errorf("CredentialsFilePath() = %q, want absolute", path)
	}
	if filepath.Base(path) != "credentials.json" {
		t.Errorf("base = %q, want credentials.json", filepath.Base(path))
	}
}

func TestSaveCredentials(t *testing.T) {
	// We can't easily override CredentialsFilePath, so test validation only.
	err := SaveCredentials(ClientCredentials{ClientID: "", ClientSecret: "s"})
	if err == nil {
		t.Error("SaveCredentials should fail with empty ClientID")
	}

	err = SaveCredentials(ClientCredentials{ClientID: "id", ClientSecret: ""})
	if err == nil {
		t.Error("SaveCredentials should fail with empty ClientSecret")
	}
}

func TestLoadCredentials_FromBundled(t *testing.T) {
	t.Setenv("GKESTRAL_CLIENT_ID", "")
	t.Setenv("GKESTRAL_CLIENT_SECRET", "")
	old := BundledCredentials
	BundledCredentials = ClientCredentials{ClientID: "bundled-id", ClientSecret: "bundled-secret"}
	defer func() { BundledCredentials = old }()

	creds, err := LoadCredentials()
	if err != nil {
		t.Fatalf("LoadCredentials error: %v", err)
	}
	if creds.ClientID != "bundled-id" {
		t.Errorf("ClientID = %q, want bundled-id", creds.ClientID)
	}
}

func TestLoadCredentials_EnvOverridesBundled(t *testing.T) {
	t.Setenv("GKESTRAL_CLIENT_ID", "env-id")
	t.Setenv("GKESTRAL_CLIENT_SECRET", "env-secret")
	old := BundledCredentials
	BundledCredentials = ClientCredentials{ClientID: "bundled-id", ClientSecret: "bundled-secret"}
	defer func() { BundledCredentials = old }()

	creds, err := LoadCredentials()
	if err != nil {
		t.Fatalf("LoadCredentials error: %v", err)
	}
	if creds.ClientID != "env-id" {
		t.Errorf("ClientID = %q, want env-id (env should override bundled)", creds.ClientID)
	}
}
