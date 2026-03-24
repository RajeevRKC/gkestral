package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrNoCredentials is returned when no OAuth client credentials are configured.
var ErrNoCredentials = errors.New("auth: no OAuth client credentials configured")

// ClientCredentials holds the OAuth 2.0 client ID and secret.
type ClientCredentials struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// Validate checks that required fields are present.
func (c ClientCredentials) Validate() error {
	if c.ClientID == "" {
		return errors.New("auth: client_id is required")
	}
	if c.ClientSecret == "" {
		return errors.New("auth: client_secret is required")
	}
	return nil
}

// BundledCredentials are compiled into the binary for production use.
// End users never need to create a GCP project -- they just sign in.
// These are placeholder values during development; replaced before release.
var BundledCredentials = ClientCredentials{
	ClientID:     "", // Set before shipping
	ClientSecret: "", // Set before shipping
}

// LoadCredentials resolves OAuth client credentials from multiple sources.
// Priority order:
//  1. Environment variables (GKESTRAL_CLIENT_ID, GKESTRAL_CLIENT_SECRET)
//  2. Config file (~/.gkestral/credentials.json)
//  3. Bundled credentials (compiled into binary for production)
//
// Returns ErrNoCredentials if all sources are empty.
func LoadCredentials() (ClientCredentials, error) {
	// 1. Environment variables (highest priority -- developer override).
	if id, secret := os.Getenv("GKESTRAL_CLIENT_ID"), os.Getenv("GKESTRAL_CLIENT_SECRET"); id != "" && secret != "" {
		return ClientCredentials{ClientID: id, ClientSecret: secret}, nil
	}

	// 2. Config file.
	if creds, err := loadCredentialsFile(); err == nil {
		return creds, nil
	}

	// 3. Bundled credentials (production).
	if BundledCredentials.ClientID != "" && BundledCredentials.ClientSecret != "" {
		return BundledCredentials, nil
	}

	return ClientCredentials{}, ErrNoCredentials
}

// CredentialsFilePath returns the path to the credentials config file.
func CredentialsFilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".gkestral", "credentials.json")
}

func loadCredentialsFile() (ClientCredentials, error) {
	path := CredentialsFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		return ClientCredentials{}, err
	}
	var creds ClientCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return ClientCredentials{}, fmt.Errorf("auth: parse credentials file: %w", err)
	}
	if err := creds.Validate(); err != nil {
		return ClientCredentials{}, err
	}
	return creds, nil
}

// SaveCredentials writes client credentials to the config file.
func SaveCredentials(creds ClientCredentials) error {
	if err := creds.Validate(); err != nil {
		return err
	}
	path := CredentialsFilePath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("auth: create config dir: %w", err)
	}
	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("auth: marshal credentials: %w", err)
	}
	return os.WriteFile(path, data, 0600)
}
