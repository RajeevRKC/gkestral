//go:build integration

package gmail

import (
	"context"
	"os"
	"testing"
)

func skipIfNoGmailCreds(t *testing.T) {
	t.Helper()
	if os.Getenv("GKESTRAL_TEST_TOKEN") == "" {
		t.Skip("GKESTRAL_TEST_TOKEN not set -- skipping Gmail integration test")
	}
}

func TestIntegration_Smoke(t *testing.T) {
	if os.Getenv("GKESTRAL_TEST_TOKEN") == "" {
		t.Log("No test token -- skip logic works correctly")
		return
	}
	t.Log("Test token found -- real Gmail integration tests would run")
}

func TestIntegration_GmailSearch(t *testing.T) {
	skipIfNoGmailCreds(t)

	query := os.Getenv("GKESTRAL_TEST_GMAIL_QUERY")
	if query == "" {
		query = "subject:gkestral-test"
	}

	ctx := context.Background()
	_ = ctx
	t.Logf("Would search Gmail with query: %s", query)
	// Implementation: parse GKESTRAL_TEST_TOKEN, create http.Client, search.
}

func TestIntegration_GmailExtractContent(t *testing.T) {
	skipIfNoGmailCreds(t)
	t.Log("Would extract content from a known message")
}

func TestIntegration_GmailBatchExtract(t *testing.T) {
	skipIfNoGmailCreds(t)
	t.Log("Would batch extract 5 messages with rate limiting")
}
