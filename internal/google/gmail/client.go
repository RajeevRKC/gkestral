package gmail

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"gkestral/internal/google/transport"
)

const (
	// GmailBaseURL is the Gmail API v1 base URL.
	GmailBaseURL = "https://gmail.googleapis.com/gmail/v1"

	// DefaultMaxResults is the default number of messages per search page.
	DefaultMaxResults = 20

	// DefaultMaxSearchPages is the max number of pages for search results.
	DefaultMaxSearchPages = 5
)

// GmailClient provides access to Gmail API v1.
type GmailClient struct {
	client *transport.GoogleClient
	userID string
}

// GmailOption configures a GmailClient.
type GmailOption func(*GmailClient)

// NewGmailClient creates a Gmail client using the given authenticated HTTP client.
func NewGmailClient(httpClient *http.Client, opts ...GmailOption) *GmailClient {
	g := &GmailClient{
		client: transport.NewGoogleClient(httpClient, GmailBaseURL),
		userID: "me",
	}
	for _, opt := range opts {
		opt(g)
	}
	return g
}

// WithGmailBaseURL overrides the Gmail API base URL (for testing).
func WithGmailBaseURL(baseURL string) GmailOption {
	return func(g *GmailClient) {
		g.client = transport.NewGoogleClient(g.client.HTTPClient, baseURL)
	}
}

// WithUserID sets the Gmail user ID (default "me" for authenticated user).
func WithUserID(userID string) GmailOption {
	return func(g *GmailClient) { g.userID = userID }
}

// SearchOption configures a message search request.
type SearchOption func(*searchConfig)

type searchConfig struct {
	maxResults int
	maxPages   int
	labelIDs   []string
}

func defaultSearchConfig() searchConfig {
	return searchConfig{
		maxResults: DefaultMaxResults,
		maxPages:   DefaultMaxSearchPages,
	}
}

// WithMaxResults sets the number of results per page.
func WithMaxResults(n int) SearchOption {
	return func(c *searchConfig) { c.maxResults = n }
}

// WithSearchMaxPages sets the maximum number of search pages.
func WithSearchMaxPages(n int) SearchOption {
	return func(c *searchConfig) { c.maxPages = n }
}

// WithLabelIDs filters messages by label IDs.
func WithLabelIDs(labels ...string) SearchOption {
	return func(c *searchConfig) { c.labelIDs = labels }
}

// SearchMessages searches Gmail using Gmail search syntax with bounded pagination.
// Returns enriched messages with header metadata (subject, from, date).
func (g *GmailClient) SearchMessages(ctx context.Context, query string, opts ...SearchOption) ([]GmailMessage, error) {
	cfg := defaultSearchConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	// Step 1: List message IDs matching the query.
	params := url.Values{
		"q":          {query},
		"maxResults": {fmt.Sprintf("%d", cfg.maxResults)},
	}
	for _, label := range cfg.labelIDs {
		params.Add("labelIds", label)
	}

	path := fmt.Sprintf("users/%s/messages", g.userID)

	extract := func(data json.RawMessage) ([]gmailMessageRef, string, error) {
		var resp gmailMessageListResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, "", err
		}
		return resp.Messages, resp.NextPageToken, nil
	}

	refs, err := transport.PaginatedList(ctx, g.client, path, extract, cfg.maxPages, params)
	if err != nil {
		return nil, err
	}

	// Step 2: Fetch metadata for each message.
	messages := make([]GmailMessage, 0, len(refs))
	for _, ref := range refs {
		msg, err := g.GetMessage(ctx, ref.ID)
		if err != nil {
			return messages, fmt.Errorf("gmail: get message %s: %w", ref.ID, err)
		}
		messages = append(messages, *msg)
	}

	return messages, nil
}

// GetMessage retrieves a single message with metadata (headers + snippet, no body).
func (g *GmailClient) GetMessage(ctx context.Context, messageID string) (*GmailMessage, error) {
	path := fmt.Sprintf("users/%s/messages/%s", g.userID, messageID)
	params := url.Values{
		"format": {"metadata"},
		"metadataHeaders": {"Subject", "From", "To", "Date"},
	}

	var raw gmailMessageResponse
	err := g.client.DoJSON(ctx, http.MethodGet, path, nil, &raw,
		transport.WithQuery(params))
	if err != nil {
		return nil, err
	}

	msg := &GmailMessage{
		ID:           raw.ID,
		ThreadID:     raw.ThreadID,
		Snippet:      raw.Snippet,
		LabelIDs:     raw.LabelIDs,
		SizeEstimate: raw.SizeEstimate,
	}
	extractHeaders(msg, raw.Payload.Headers)

	// Check for attachments.
	msg.HasAttachments = hasAttachments(raw.Payload.Parts)

	return msg, nil
}

// ListLabels returns all Gmail labels for the authenticated user.
func (g *GmailClient) ListLabels(ctx context.Context) ([]GmailLabel, error) {
	path := fmt.Sprintf("users/%s/labels", g.userID)
	var resp struct {
		Labels []GmailLabel `json:"labels"`
	}
	err := g.client.DoJSON(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Labels, nil
}

func hasAttachments(parts []gmailPart) bool {
	for _, p := range parts {
		if p.Filename != "" && p.Body.AttachmentID != "" {
			return true
		}
		if hasAttachments(p.Parts) {
			return true
		}
	}
	return false
}
