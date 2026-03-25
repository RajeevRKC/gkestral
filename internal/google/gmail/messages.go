package gmail

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"golang.org/x/time/rate"

	"gkestral/internal/google/transport"
)

// MessageContent holds extracted message body content.
type MessageContent struct {
	PlainText   string            // Plain text body (preferred).
	HTMLText    string            // Stripped HTML body (fallback).
	Attachments []GmailAttachment // Attachment metadata.
}

// GmailFullMessage is the raw API response for messages.get with full format.
type GmailFullMessage struct {
	ID           string       `json:"id"`
	ThreadID     string       `json:"threadId"`
	Snippet      string       `json:"snippet"`
	LabelIDs     []string     `json:"labelIds"`
	SizeEstimate int          `json:"sizeEstimate"`
	Payload      gmailPayload `json:"payload"`
}

// GetFullMessage retrieves a message with full content (body + attachments).
func (g *GmailClient) GetFullMessage(ctx context.Context, messageID string) (*GmailFullMessage, error) {
	path := fmt.Sprintf("users/%s/messages/%s", g.userID, messageID)
	var raw GmailFullMessage
	err := g.client.DoJSON(ctx, http.MethodGet, path, nil, &raw,
		transport.WithQuery(map[string][]string{"format": {"full"}}))
	if err != nil {
		return nil, err
	}
	return &raw, nil
}

// ExtractContent walks the MIME tree and extracts plain text, HTML, and attachments.
func ExtractContent(msg *GmailFullMessage) (*MessageContent, error) {
	content := &MessageContent{}

	// Walk the payload.
	walkParts(msg.Payload.Parts, content)

	// If no parts, check the top-level body.
	if len(msg.Payload.Parts) == 0 && msg.Payload.Body.Data != "" {
		text, err := decodeBase64URL(msg.Payload.Body.Data)
		if err == nil {
			if strings.HasPrefix(msg.Payload.MIMEType, "text/plain") {
				content.PlainText = text
			} else if strings.HasPrefix(msg.Payload.MIMEType, "text/html") {
				content.HTMLText = stripHTML(text)
			}
		}
	}

	return content, nil
}

func walkParts(parts []gmailPart, content *MessageContent) {
	for _, part := range parts {
		switch {
		case part.MIMEType == "text/plain" && part.Body.Data != "" && content.PlainText == "":
			if text, err := decodeBase64URL(part.Body.Data); err == nil {
				content.PlainText = text
			}
		case part.MIMEType == "text/html" && part.Body.Data != "" && content.HTMLText == "":
			if text, err := decodeBase64URL(part.Body.Data); err == nil {
				content.HTMLText = stripHTML(text)
			}
		case part.Filename != "" && part.Body.AttachmentID != "":
			content.Attachments = append(content.Attachments, GmailAttachment{
				Name:         part.Filename,
				MIMEType:     part.MIMEType,
				Size:         part.Body.Size,
				AttachmentID: part.Body.AttachmentID,
			})
		}
		// Recurse into nested parts (multipart/alternative, multipart/mixed, etc.).
		if len(part.Parts) > 0 {
			walkParts(part.Parts, content)
		}
	}
}

// FormatOption configures context formatting.
type FormatOption func(*formatConfig)

type formatConfig struct {
	maxLength      int
	includeAttList bool
}

// WithMaxLength truncates the body to approximately this many characters.
func WithMaxLength(n int) FormatOption {
	return func(c *formatConfig) { c.maxLength = n }
}

// WithAttachmentList includes attachment metadata in the formatted output.
func WithAttachmentList(include bool) FormatOption {
	return func(c *formatConfig) { c.includeAttList = include }
}

// FormatForContext produces clean text suitable for Gemini context injection.
func FormatForContext(msg *GmailMessage, content *MessageContent, opts ...FormatOption) string {
	cfg := formatConfig{maxLength: 0, includeAttList: true}
	for _, opt := range opts {
		opt(&cfg)
	}

	var b strings.Builder
	b.WriteString("From: " + msg.From + "\n")
	if msg.To != "" {
		b.WriteString("To: " + msg.To + "\n")
	}
	if !msg.Date.IsZero() {
		b.WriteString("Date: " + msg.Date.Format("2006-01-02 15:04") + "\n")
	}
	b.WriteString("Subject: " + msg.Subject + "\n\n")

	// Prefer plain text, fall back to stripped HTML.
	body := content.PlainText
	if body == "" {
		body = content.HTMLText
	}

	if cfg.maxLength > 0 && len(body) > cfg.maxLength {
		body = body[:cfg.maxLength] + "\n[...truncated]"
	}
	b.WriteString(body)

	if cfg.includeAttList && len(content.Attachments) > 0 {
		b.WriteString("\n\nAttachments:\n")
		for _, att := range content.Attachments {
			b.WriteString(fmt.Sprintf("- %s (%s, %d bytes)\n", att.Name, att.MIMEType, att.Size))
		}
	}

	return b.String()
}

// BatchExtract concurrently extracts content from multiple messages.
// Uses a Token Bucket rate limiter to respect Gmail's 250 quota units/second
// (messages.get costs 5 units, so ~40 requests/second with headroom).
func (g *GmailClient) BatchExtract(ctx context.Context, messageIDs []string, maxConcurrent int) ([]MessageContent, error) {
	if maxConcurrent <= 0 {
		maxConcurrent = 5
	}

	limiter := rate.NewLimiter(rate.Limit(40), 10) // 40 req/s, burst of 10.
	results := make([]MessageContent, len(messageIDs))
	errs := make([]error, len(messageIDs))

	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrent)

	for i, id := range messageIDs {
		wg.Add(1)
		go func(idx int, msgID string) {
			defer wg.Done()

			// Acquire semaphore slot, but respect context cancellation.
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				errs[idx] = ctx.Err()
				return
			}

			if err := limiter.Wait(ctx); err != nil {
				errs[idx] = err
				return
			}

			full, err := g.GetFullMessage(ctx, msgID)
			if err != nil {
				errs[idx] = err
				return
			}

			content, err := ExtractContent(full)
			if err != nil {
				errs[idx] = err
				return
			}
			results[idx] = *content
		}(i, id)
	}
	wg.Wait()

	// Return first error encountered.
	for _, err := range errs {
		if err != nil {
			return results, err
		}
	}
	return results, nil
}

// DownloadAttachment retrieves attachment content by message and attachment ID.
func (g *GmailClient) DownloadAttachment(ctx context.Context, messageID, attachmentID string) ([]byte, error) {
	path := fmt.Sprintf("users/%s/messages/%s/attachments/%s", g.userID, messageID, attachmentID)
	var resp struct {
		Data string `json:"data"` // Base64url-encoded.
		Size int    `json:"size"`
	}
	err := g.client.DoJSON(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(resp.Data)
}

// --- Helpers ---

// decodeBase64URL decodes Gmail's URL-safe base64 encoded content.
func decodeBase64URL(s string) (string, error) {
	// Gmail uses base64url without padding.
	data, err := base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// stripHTML removes HTML tags and decodes common entities.
var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

func stripHTML(html string) string {
	// Replace block-level tags with newlines.
	for _, tag := range []string{"</p>", "</div>", "</tr>", "<br>", "<br/>", "<br />"} {
		html = strings.ReplaceAll(html, tag, "\n")
	}
	// Replace list items with bullets.
	html = strings.ReplaceAll(html, "<li>", "- ")
	html = strings.ReplaceAll(html, "</li>", "\n")

	// Strip all remaining tags.
	text := htmlTagRe.ReplaceAllString(html, "")

	// Decode common HTML entities.
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", `"`)
	text = strings.ReplaceAll(text, "&#39;", "'")
	text = strings.ReplaceAll(text, "&nbsp;", " ")

	// Collapse multiple newlines.
	for strings.Contains(text, "\n\n\n") {
		text = strings.ReplaceAll(text, "\n\n\n", "\n\n")
	}

	return strings.TrimSpace(text)
}
