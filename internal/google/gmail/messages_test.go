package gmail

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

func b64url(s string) string {
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString([]byte(s))
}

func TestExtractContent_PlainTextOnly(t *testing.T) {
	msg := &GmailFullMessage{
		Payload: gmailPayload{
			Parts: []gmailPart{
				{MIMEType: "text/plain", Body: gmailBody{Data: b64url("Hello, World!")}},
			},
		},
	}
	content, err := ExtractContent(msg)
	if err != nil {
		t.Fatalf("ExtractContent error: %v", err)
	}
	if content.PlainText != "Hello, World!" {
		t.Errorf("PlainText = %q", content.PlainText)
	}
}

func TestExtractContent_HTMLOnly(t *testing.T) {
	html := "<html><body><p>Hello</p><p>World</p></body></html>"
	msg := &GmailFullMessage{
		Payload: gmailPayload{
			Parts: []gmailPart{
				{MIMEType: "text/html", Body: gmailBody{Data: b64url(html)}},
			},
		},
	}
	content, err := ExtractContent(msg)
	if err != nil {
		t.Fatalf("ExtractContent error: %v", err)
	}
	if content.PlainText != "" {
		t.Errorf("PlainText should be empty, got %q", content.PlainText)
	}
	if !strings.Contains(content.HTMLText, "Hello") {
		t.Errorf("HTMLText = %q, should contain Hello", content.HTMLText)
	}
	if strings.Contains(content.HTMLText, "<") {
		t.Errorf("HTMLText should be stripped of tags: %q", content.HTMLText)
	}
}

func TestExtractContent_MultipartAlternative(t *testing.T) {
	msg := &GmailFullMessage{
		Payload: gmailPayload{
			Parts: []gmailPart{
				{MIMEType: "text/plain", Body: gmailBody{Data: b64url("Plain version")}},
				{MIMEType: "text/html", Body: gmailBody{Data: b64url("<b>HTML version</b>")}},
			},
		},
	}
	content, err := ExtractContent(msg)
	if err != nil {
		t.Fatalf("ExtractContent error: %v", err)
	}
	if content.PlainText != "Plain version" {
		t.Errorf("PlainText = %q, should prefer plain over HTML", content.PlainText)
	}
}

func TestExtractContent_NestedMultipart(t *testing.T) {
	msg := &GmailFullMessage{
		Payload: gmailPayload{
			Parts: []gmailPart{
				{MIMEType: "multipart/alternative", Parts: []gmailPart{
					{MIMEType: "text/plain", Body: gmailBody{Data: b64url("Nested plain")}},
					{MIMEType: "text/html", Body: gmailBody{Data: b64url("<p>Nested HTML</p>")}},
				}},
				{MIMEType: "application/pdf", Filename: "report.pdf", Body: gmailBody{AttachmentID: "att1", Size: 5000}},
			},
		},
	}
	content, err := ExtractContent(msg)
	if err != nil {
		t.Fatalf("ExtractContent error: %v", err)
	}
	if content.PlainText != "Nested plain" {
		t.Errorf("PlainText = %q", content.PlainText)
	}
	if len(content.Attachments) != 1 {
		t.Fatalf("Attachments len = %d, want 1", len(content.Attachments))
	}
	if content.Attachments[0].Name != "report.pdf" {
		t.Errorf("Attachment name = %q", content.Attachments[0].Name)
	}
}

func TestExtractContent_TopLevelBody(t *testing.T) {
	msg := &GmailFullMessage{
		Payload: gmailPayload{
			MIMEType: "text/plain",
			Body:     gmailBody{Data: b64url("Top-level body content")},
		},
	}
	content, err := ExtractContent(msg)
	if err != nil {
		t.Fatalf("ExtractContent error: %v", err)
	}
	if content.PlainText != "Top-level body content" {
		t.Errorf("PlainText = %q", content.PlainText)
	}
}

func TestExtractContent_TopLevelHTML(t *testing.T) {
	msg := &GmailFullMessage{
		Payload: gmailPayload{
			MIMEType: "text/html",
			Body:     gmailBody{Data: b64url("<p>HTML only</p>")},
		},
	}
	content, err := ExtractContent(msg)
	if err != nil {
		t.Fatalf("ExtractContent error: %v", err)
	}
	if !strings.Contains(content.HTMLText, "HTML only") {
		t.Errorf("HTMLText = %q", content.HTMLText)
	}
}

func TestExtractContent_EmptyMessage(t *testing.T) {
	msg := &GmailFullMessage{Payload: gmailPayload{}}
	content, err := ExtractContent(msg)
	if err != nil {
		t.Fatalf("ExtractContent error: %v", err)
	}
	if content.PlainText != "" || content.HTMLText != "" {
		t.Error("empty message should produce empty content")
	}
}

func TestFormatForContext(t *testing.T) {
	msg := &GmailMessage{
		From:    "boss@company.com",
		To:      "me@company.com",
		Date:    time.Date(2026, 3, 25, 14, 30, 0, 0, time.UTC),
		Subject: "Q1 Report",
	}
	content := &MessageContent{
		PlainText: "Please review the attached Q1 report.",
		Attachments: []GmailAttachment{
			{Name: "q1-report.pdf", MIMEType: "application/pdf", Size: 50000},
		},
	}

	formatted := FormatForContext(msg, content)
	if !strings.Contains(formatted, "From: boss@company.com") {
		t.Error("should contain From")
	}
	if !strings.Contains(formatted, "Subject: Q1 Report") {
		t.Error("should contain Subject")
	}
	if !strings.Contains(formatted, "Please review") {
		t.Error("should contain body")
	}
	if !strings.Contains(formatted, "q1-report.pdf") {
		t.Error("should contain attachment list")
	}
}

func TestFormatForContext_FallsBackToHTML(t *testing.T) {
	msg := &GmailMessage{From: "a@b.com", Subject: "Test"}
	content := &MessageContent{HTMLText: "Stripped HTML content"}

	formatted := FormatForContext(msg, content)
	if !strings.Contains(formatted, "Stripped HTML content") {
		t.Error("should fall back to HTML when plain text is empty")
	}
}

func TestFormatForContext_Truncation(t *testing.T) {
	msg := &GmailMessage{From: "a@b.com", Subject: "Long"}
	content := &MessageContent{PlainText: strings.Repeat("x", 1000)}

	formatted := FormatForContext(msg, content, WithMaxLength(100))
	if !strings.Contains(formatted, "[...truncated]") {
		t.Error("should contain truncation marker")
	}
}

func TestFormatForContext_NoAttachments(t *testing.T) {
	msg := &GmailMessage{From: "a@b.com", Subject: "Test"}
	content := &MessageContent{PlainText: "Body"}

	formatted := FormatForContext(msg, content)
	if strings.Contains(formatted, "Attachments:") {
		t.Error("should not contain Attachments section when none present")
	}
}

func TestFormatForContext_HideAttachments(t *testing.T) {
	msg := &GmailMessage{From: "a@b.com", Subject: "Test"}
	content := &MessageContent{
		PlainText:   "Body",
		Attachments: []GmailAttachment{{Name: "file.pdf"}},
	}

	formatted := FormatForContext(msg, content, WithAttachmentList(false))
	if strings.Contains(formatted, "Attachments:") {
		t.Error("should not show attachments when WithAttachmentList(false)")
	}
}

func TestStripHTML(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"<p>Hello</p><p>World</p>", "Hello\nWorld"},
		{"<b>Bold</b> text", "Bold text"},
		{"<ul><li>One</li><li>Two</li></ul>", "- One\n- Two"},
		{"Line1<br>Line2", "Line1\nLine2"},
		{"&amp; &lt; &gt; &quot; &#39;", `& < > " '`},
		{"No tags here", "No tags here"},
		{"<div>Block</div><div>Level</div>", "Block\nLevel"},
	}
	for _, tt := range tests {
		got := stripHTML(tt.input)
		got = strings.TrimSpace(got)
		want := strings.TrimSpace(tt.want)
		if got != want {
			t.Errorf("stripHTML(%q) = %q, want %q", tt.input, got, want)
		}
	}
}

func TestDecodeBase64URL(t *testing.T) {
	input := b64url("Hello, World!")
	got, err := decodeBase64URL(input)
	if err != nil {
		t.Fatalf("decodeBase64URL error: %v", err)
	}
	if got != "Hello, World!" {
		t.Errorf("got = %q", got)
	}
}

func TestDecodeBase64URL_Invalid(t *testing.T) {
	_, err := decodeBase64URL("not-valid-base64!!!")
	if err == nil {
		t.Error("expected error for invalid base64")
	}
}

func TestBatchExtract(t *testing.T) {
	client, srv := testGmailServer(func(w http.ResponseWriter, r *http.Request) {
		// Extract message ID from path.
		parts := strings.Split(r.URL.Path, "/")
		msgID := parts[len(parts)-1]
		fmt.Fprintf(w, `{
			"id":"%s","payload":{
				"mimeType":"text/plain",
				"body":{"data":"%s"}
			}
		}`, msgID, b64url("Content of "+msgID))
	})
	defer srv.Close()

	ids := []string{"m1", "m2", "m3"}
	results, err := client.BatchExtract(context.Background(), ids, 2)
	if err != nil {
		t.Fatalf("BatchExtract error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("results len = %d, want 3", len(results))
	}
	for i, r := range results {
		expected := fmt.Sprintf("Content of m%d", i+1)
		if r.PlainText != expected {
			t.Errorf("results[%d].PlainText = %q, want %q", i, r.PlainText, expected)
		}
	}
}

func TestDownloadAttachment(t *testing.T) {
	encodedData := base64.URLEncoding.EncodeToString([]byte("attachment content"))
	client, srv := testGmailServer(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"data":"%s","size":18}`, encodedData)
	})
	defer srv.Close()

	data, err := client.DownloadAttachment(context.Background(), "msg1", "att1")
	if err != nil {
		t.Fatalf("DownloadAttachment error: %v", err)
	}
	if string(data) != "attachment content" {
		t.Errorf("data = %q", string(data))
	}
}

func TestGetFullMessage(t *testing.T) {
	client, srv := testGmailServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("format") != "full" {
			t.Errorf("format = %q, want full", r.URL.Query().Get("format"))
		}
		fmt.Fprint(w, `{
			"id":"msg1","snippet":"Test",
			"payload":{"mimeType":"text/plain","body":{"data":"`+b64url("Full body")+`"}}
		}`)
	})
	defer srv.Close()

	msg, err := client.GetFullMessage(context.Background(), "msg1")
	if err != nil {
		t.Fatalf("GetFullMessage error: %v", err)
	}
	if msg.ID != "msg1" {
		t.Errorf("ID = %q", msg.ID)
	}
}
