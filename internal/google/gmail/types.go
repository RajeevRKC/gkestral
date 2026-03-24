// Package gmail provides a client for the Gmail API v1.
// Built on the shared transport layer -- no google-api-go-client dependency.
package gmail

import "time"

// GmailMessage represents a message with header-level metadata.
type GmailMessage struct {
	ID             string    `json:"id"`
	ThreadID       string    `json:"threadId"`
	Subject        string    `json:"-"` // Extracted from headers.
	From           string    `json:"-"` // Extracted from headers.
	To             string    `json:"-"` // Extracted from headers.
	Date           time.Time `json:"-"` // Extracted from headers.
	Snippet        string    `json:"snippet"`
	LabelIDs       []string  `json:"labelIds"`
	HasAttachments bool      `json:"-"` // Derived from payload.
	SizeEstimate   int       `json:"sizeEstimate"`
}

// GmailLabel represents a Gmail label.
type GmailLabel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"` // "system" or "user"
}

// GmailAttachment represents a message attachment.
type GmailAttachment struct {
	Name         string `json:"filename"`
	MIMEType     string `json:"mimeType"`
	Size         int    `json:"size"`
	AttachmentID string `json:"attachmentId"`
}

// gmailMessageListResponse is the raw API response for messages.list.
type gmailMessageListResponse struct {
	Messages          []gmailMessageRef `json:"messages"`
	NextPageToken     string            `json:"nextPageToken"`
	ResultSizeEstimate int              `json:"resultSizeEstimate"`
}

type gmailMessageRef struct {
	ID       string `json:"id"`
	ThreadID string `json:"threadId"`
}

// gmailMessageResponse is the raw API response for messages.get with metadata format.
type gmailMessageResponse struct {
	ID           string       `json:"id"`
	ThreadID     string       `json:"threadId"`
	Snippet      string       `json:"snippet"`
	LabelIDs     []string     `json:"labelIds"`
	SizeEstimate int          `json:"sizeEstimate"`
	Payload      gmailPayload `json:"payload"`
}

type gmailPayload struct {
	MIMEType string        `json:"mimeType"`
	Headers  []gmailHeader `json:"headers"`
	Parts    []gmailPart   `json:"parts"`
	Body     gmailBody     `json:"body"`
}

type gmailHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type gmailPart struct {
	MIMEType string    `json:"mimeType"`
	Filename string    `json:"filename"`
	Body     gmailBody `json:"body"`
	Parts    []gmailPart `json:"parts"` // Recursive for multipart.
}

type gmailBody struct {
	AttachmentID string `json:"attachmentId"`
	Size         int    `json:"size"`
	Data         string `json:"data"` // Base64url-encoded content.
}

// extractHeaders populates GmailMessage fields from raw headers.
func extractHeaders(msg *GmailMessage, headers []gmailHeader) {
	for _, h := range headers {
		switch h.Name {
		case "Subject":
			msg.Subject = h.Value
		case "From":
			msg.From = h.Value
		case "To":
			msg.To = h.Value
		case "Date":
			if t, err := time.Parse(time.RFC1123Z, h.Value); err == nil {
				msg.Date = t
			} else if t, err := time.Parse("Mon, 2 Jan 2006 15:04:05 -0700", h.Value); err == nil {
				msg.Date = t
			}
		}
	}
}
