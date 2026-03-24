//go:build integration

package drive

import (
	"context"
	"os"
	"testing"
)

func skipIfNoDriveCreds(t *testing.T) {
	t.Helper()
	if os.Getenv("GKESTRAL_TEST_TOKEN") == "" {
		t.Skip("GKESTRAL_TEST_TOKEN not set -- skipping Drive integration test")
	}
	if os.Getenv("GKESTRAL_TEST_DRIVE_FOLDER") == "" {
		t.Skip("GKESTRAL_TEST_DRIVE_FOLDER not set -- skipping Drive integration test")
	}
}

func TestIntegration_Smoke(t *testing.T) {
	if os.Getenv("GKESTRAL_TEST_TOKEN") == "" {
		t.Log("No test token -- skip logic works correctly")
		return
	}
	t.Log("Test token found -- real Drive integration tests would run")
}

func TestIntegration_DriveListFiles(t *testing.T) {
	skipIfNoDriveCreds(t)
	folderID := os.Getenv("GKESTRAL_TEST_DRIVE_FOLDER")

	// Create authenticated client from test token and list files.
	t.Logf("Would list files in folder %s", folderID)
	// Implementation: parse GKESTRAL_TEST_TOKEN, create http.Client, list files.
}

func TestIntegration_DriveDownloadText(t *testing.T) {
	skipIfNoDriveCreds(t)
	t.Log("Would download a Google Doc as plain text")
}

func TestIntegration_DriveUploadFile(t *testing.T) {
	skipIfNoDriveCreds(t)
	t.Log("Would upload a test file (manual cleanup required)")
}

func TestIntegration_DriveSearchFiles(t *testing.T) {
	skipIfNoDriveCreds(t)

	ctx := context.Background()
	_ = ctx
	t.Log("Would search Drive for test files")
}
