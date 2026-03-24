package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// SetupStatus describes the result of checking the current auth configuration.
type SetupStatus struct {
	HasCredentials  bool   // Client ID + secret available
	HasToken        bool   // Stored OAuth token exists
	TokenExpired    bool   // Token exists but is expired
	CredentialSource string // "env", "file", "bundled", or ""
	ScopesGranted   ScopeSet
}

// CheckSetup inspects the current auth configuration and returns its status.
func CheckSetup(store TokenStore) SetupStatus {
	status := SetupStatus{}

	creds, err := LoadCredentials()
	if err == nil && creds.Validate() == nil {
		status.HasCredentials = true
		switch {
		case creds.ClientID == BundledCredentials.ClientID && BundledCredentials.ClientID != "":
			status.CredentialSource = "bundled"
		default:
			// Check env vs file
			if id := envOrEmpty("GKESTRAL_CLIENT_ID"); id == creds.ClientID {
				status.CredentialSource = "env"
			} else {
				status.CredentialSource = "file"
			}
		}
	}

	if store != nil && store.Exists() {
		status.HasToken = true
		if tok, err := store.Load(); err == nil {
			status.TokenExpired = tok.IsExpired()
			status.ScopesGranted = NewScopeSet(tok.Scopes...)
		}
	}

	return status
}

func envOrEmpty(key string) string {
	return os.Getenv(key)
}

// GCloudAvailable reports whether the gcloud CLI is installed and accessible.
func GCloudAvailable() bool {
	_, err := exec.LookPath("gcloud")
	return err == nil
}

// GCloudProject holds metadata about a Google Cloud project.
type GCloudProject struct {
	ProjectID   string `json:"projectId"`
	ProjectName string `json:"name"`
}

// GCloudListProjects returns the user's GCP projects (requires gcloud auth).
func GCloudListProjects(ctx context.Context) ([]GCloudProject, error) {
	cmd := exec.CommandContext(ctx, "gcloud", "projects", "list", "--format=json", "--limit=20")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gcloud projects list: %w", err)
	}
	var projects []GCloudProject
	if err := json.Unmarshal(out, &projects); err != nil {
		return nil, fmt.Errorf("parse gcloud output: %w", err)
	}
	return projects, nil
}

// GCloudCreateProject creates a new GCP project.
func GCloudCreateProject(ctx context.Context, projectID, projectName string) error {
	cmd := exec.CommandContext(ctx, "gcloud", "projects", "create", projectID,
		"--name="+projectName, "--format=json")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("gcloud projects create: %s: %w", string(out), err)
	}
	return nil
}

// GCloudEnableAPIs enables the required Google APIs for Gkestral.
func GCloudEnableAPIs(ctx context.Context, projectID string) error {
	apis := []string{
		"generativelanguage.googleapis.com",
		"drive.googleapis.com",
		"gmail.googleapis.com",
		"oauth2.googleapis.com",
	}
	args := append([]string{"services", "enable", "--project=" + projectID}, apis...)
	cmd := exec.CommandContext(ctx, "gcloud", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("gcloud services enable: %s: %w", string(out), err)
	}
	return nil
}

// ConsoleURLs returns the Google Cloud Console URLs for manual setup.
type ConsoleURLs struct {
	CreateProject   string
	EnableAPIs      string
	OAuthConsent    string
	CreateOAuthCred string
}

// GetConsoleURLs returns the Cloud Console URLs, optionally scoped to a project.
func GetConsoleURLs(projectID string) ConsoleURLs {
	urls := ConsoleURLs{
		CreateProject: "https://console.cloud.google.com/projectcreate",
	}
	if projectID != "" {
		urls.EnableAPIs = fmt.Sprintf("https://console.cloud.google.com/apis/library?project=%s", projectID)
		urls.OAuthConsent = fmt.Sprintf("https://console.cloud.google.com/apis/credentials/consent?project=%s", projectID)
		urls.CreateOAuthCred = fmt.Sprintf("https://console.cloud.google.com/apis/credentials/oauthclient?project=%s", projectID)
	} else {
		urls.EnableAPIs = "https://console.cloud.google.com/apis/library"
		urls.OAuthConsent = "https://console.cloud.google.com/apis/credentials/consent"
		urls.CreateOAuthCred = "https://console.cloud.google.com/apis/credentials/oauthclient"
	}
	return urls
}

// OpenBrowser opens a URL in the user's default browser.
func OpenBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default: // linux, freebsd, etc.
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

// VerifyAPIKey checks that a Gemini API key is valid by making a test request.
func VerifyAPIKey(ctx context.Context, apiKey string) error {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models?key=%s&pageSize=1", apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("verify API key: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	return fmt.Errorf("API key verification failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
}
