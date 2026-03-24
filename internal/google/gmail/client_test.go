package gmail

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gkestral/internal/google/transport"
)

func testGmailServer(handler http.HandlerFunc) (*GmailClient, *httptest.Server) {
	srv := httptest.NewServer(handler)
	client := NewGmailClient(srv.Client(), WithGmailBaseURL(srv.URL))
	return client, srv
}

func TestSearchMessages_Basic(t *testing.T) {
	callCount := 0
	client, srv := testGmailServer(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		path := r.URL.Path

		if strings.Contains(path, "/messages/msg1") {
			fmt.Fprint(w, `{
				"id":"msg1","threadId":"t1","snippet":"Hello world",
				"labelIds":["INBOX"],"sizeEstimate":1234,
				"payload":{"headers":[
					{"name":"Subject","value":"Test Subject"},
					{"name":"From","value":"sender@example.com"},
					{"name":"To","value":"me@example.com"},
					{"name":"Date","value":"Mon, 25 Mar 2026 10:00:00 +0000"}
				]}
			}`)
			return
		}
		if strings.Contains(path, "/messages/msg2") {
			fmt.Fprint(w, `{
				"id":"msg2","threadId":"t2","snippet":"Second message",
				"payload":{"headers":[{"name":"Subject","value":"Another"}]}
			}`)
			return
		}
		// List endpoint
		q := r.URL.Query().Get("q")
		if q != "from:boss subject:report" {
			t.Errorf("q = %q, want from:boss subject:report", q)
		}
		fmt.Fprint(w, `{"messages":[{"id":"msg1","threadId":"t1"},{"id":"msg2","threadId":"t2"}]}`)
	})
	defer srv.Close()

	messages, err := client.SearchMessages(context.Background(), "from:boss subject:report")
	if err != nil {
		t.Fatalf("SearchMessages error: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("messages len = %d, want 2", len(messages))
	}
	if messages[0].Subject != "Test Subject" {
		t.Errorf("messages[0].Subject = %q", messages[0].Subject)
	}
	if messages[0].From != "sender@example.com" {
		t.Errorf("messages[0].From = %q", messages[0].From)
	}
	if messages[0].Snippet != "Hello world" {
		t.Errorf("messages[0].Snippet = %q", messages[0].Snippet)
	}
	if messages[0].SizeEstimate != 1234 {
		t.Errorf("messages[0].SizeEstimate = %d", messages[0].SizeEstimate)
	}
}

func TestSearchMessages_Pagination(t *testing.T) {
	page := 0
	client, srv := testGmailServer(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasSuffix(path, "/msg1") || strings.HasSuffix(path, "/msg2") || strings.HasSuffix(path, "/msg3") {
			id := path[strings.LastIndex(path, "/")+1:]
			fmt.Fprintf(w, `{"id":"%s","snippet":"msg","payload":{"headers":[{"name":"Subject","value":"Subj %s"}]}}`, id, id)
			return
		}
		page++
		switch page {
		case 1:
			fmt.Fprint(w, `{"messages":[{"id":"msg1"}],"nextPageToken":"p2"}`)
		case 2:
			fmt.Fprint(w, `{"messages":[{"id":"msg2"}],"nextPageToken":"p3"}`)
		case 3:
			fmt.Fprint(w, `{"messages":[{"id":"msg3"}]}`)
		}
	})
	defer srv.Close()

	messages, err := client.SearchMessages(context.Background(), "test")
	if err != nil {
		t.Fatalf("SearchMessages error: %v", err)
	}
	if len(messages) != 3 {
		t.Errorf("messages len = %d, want 3", len(messages))
	}
}

func TestSearchMessages_MaxPages(t *testing.T) {
	listCalls := 0
	client, srv := testGmailServer(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.Contains(path, "/messages/") && !strings.HasSuffix(path, "/messages") {
			id := path[strings.LastIndex(path, "/")+1:]
			fmt.Fprintf(w, `{"id":"%s","snippet":"s","payload":{"headers":[]}}`, id)
			return
		}
		listCalls++
		fmt.Fprintf(w, `{"messages":[{"id":"m%d"}],"nextPageToken":"tok%d"}`, listCalls, listCalls+1)
	})
	defer srv.Close()

	messages, err := client.SearchMessages(context.Background(), "test", WithSearchMaxPages(2))
	if err != nil {
		t.Fatalf("SearchMessages error: %v", err)
	}
	if len(messages) != 2 {
		t.Errorf("messages len = %d, want 2 (maxPages=2)", len(messages))
	}
	if listCalls != 2 {
		t.Errorf("listCalls = %d, want 2", listCalls)
	}
}

func TestSearchMessages_Empty(t *testing.T) {
	client, srv := testGmailServer(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"messages":null,"resultSizeEstimate":0}`)
	})
	defer srv.Close()

	messages, err := client.SearchMessages(context.Background(), "nonexistent-query-xyz")
	if err != nil {
		t.Fatalf("SearchMessages error: %v", err)
	}
	if len(messages) != 0 {
		t.Errorf("messages len = %d, want 0", len(messages))
	}
}

func TestSearchMessages_WithLabels(t *testing.T) {
	client, srv := testGmailServer(func(w http.ResponseWriter, r *http.Request) {
		labels := r.URL.Query()["labelIds"]
		if len(labels) != 1 || labels[0] != "INBOX" {
			t.Errorf("labelIds = %v, want [INBOX]", labels)
		}
		fmt.Fprint(w, `{"messages":[]}`)
	})
	defer srv.Close()

	_, err := client.SearchMessages(context.Background(), "test", WithLabelIDs("INBOX"))
	if err != nil {
		t.Fatalf("SearchMessages error: %v", err)
	}
}

func TestSearchMessages_Unauthorized(t *testing.T) {
	client, srv := testGmailServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	defer srv.Close()

	_, err := client.SearchMessages(context.Background(), "test")
	if !errors.Is(err, transport.ErrUnauthorized) {
		t.Errorf("error = %v, want ErrUnauthorized", err)
	}
}

func TestGetMessage(t *testing.T) {
	client, srv := testGmailServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("format") != "metadata" {
			t.Errorf("format = %q, want metadata", r.URL.Query().Get("format"))
		}
		fmt.Fprint(w, `{
			"id":"msg123","threadId":"t1","snippet":"Preview text",
			"labelIds":["INBOX","UNREAD"],"sizeEstimate":5678,
			"payload":{
				"headers":[
					{"name":"Subject","value":"Important Update"},
					{"name":"From","value":"boss@company.com"},
					{"name":"To","value":"me@company.com"},
					{"name":"Date","value":"Mon, 25 Mar 2026 14:30:00 +0300"}
				],
				"parts":[
					{"mimeType":"text/plain","body":{"size":100}},
					{"mimeType":"application/pdf","filename":"report.pdf","body":{"attachmentId":"att1","size":50000}}
				]
			}
		}`)
	})
	defer srv.Close()

	msg, err := client.GetMessage(context.Background(), "msg123")
	if err != nil {
		t.Fatalf("GetMessage error: %v", err)
	}
	if msg.Subject != "Important Update" {
		t.Errorf("Subject = %q", msg.Subject)
	}
	if msg.From != "boss@company.com" {
		t.Errorf("From = %q", msg.From)
	}
	if !msg.HasAttachments {
		t.Error("HasAttachments should be true")
	}
	if len(msg.LabelIDs) != 2 {
		t.Errorf("LabelIDs len = %d, want 2", len(msg.LabelIDs))
	}
}

func TestGetMessage_NoAttachments(t *testing.T) {
	client, srv := testGmailServer(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{
			"id":"msg456","snippet":"Plain message",
			"payload":{"headers":[{"name":"Subject","value":"Hello"}],"parts":[
				{"mimeType":"text/plain","body":{"size":50}}
			]}
		}`)
	})
	defer srv.Close()

	msg, err := client.GetMessage(context.Background(), "msg456")
	if err != nil {
		t.Fatalf("GetMessage error: %v", err)
	}
	if msg.HasAttachments {
		t.Error("HasAttachments should be false")
	}
}

func TestGetMessage_NotFound(t *testing.T) {
	client, srv := testGmailServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	defer srv.Close()

	_, err := client.GetMessage(context.Background(), "nonexistent")
	if !errors.Is(err, transport.ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound", err)
	}
}

func TestListLabels(t *testing.T) {
	client, srv := testGmailServer(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/labels") {
			t.Errorf("path = %q, want to end with /labels", r.URL.Path)
		}
		fmt.Fprint(w, `{"labels":[
			{"id":"INBOX","name":"INBOX","type":"system"},
			{"id":"Label_1","name":"Work","type":"user"}
		]}`)
	})
	defer srv.Close()

	labels, err := client.ListLabels(context.Background())
	if err != nil {
		t.Fatalf("ListLabels error: %v", err)
	}
	if len(labels) != 2 {
		t.Errorf("labels len = %d, want 2", len(labels))
	}
	if labels[0].Name != "INBOX" {
		t.Errorf("labels[0].Name = %q", labels[0].Name)
	}
}

// --- Type tests ---

func TestExtractHeaders(t *testing.T) {
	msg := &GmailMessage{}
	headers := []gmailHeader{
		{Name: "Subject", Value: "Test"},
		{Name: "From", Value: "a@b.com"},
		{Name: "To", Value: "c@d.com"},
		{Name: "Date", Value: "Mon, 25 Mar 2026 10:00:00 +0000"},
	}
	extractHeaders(msg, headers)
	if msg.Subject != "Test" {
		t.Errorf("Subject = %q", msg.Subject)
	}
	if msg.From != "a@b.com" {
		t.Errorf("From = %q", msg.From)
	}
	if msg.To != "c@d.com" {
		t.Errorf("To = %q", msg.To)
	}
	if msg.Date.IsZero() {
		t.Error("Date should be parsed")
	}
}

func TestExtractHeaders_BadDate(t *testing.T) {
	msg := &GmailMessage{}
	extractHeaders(msg, []gmailHeader{{Name: "Date", Value: "not-a-date"}})
	if !msg.Date.IsZero() {
		t.Error("Date should be zero for unparseable date")
	}
}

func TestHasAttachments_Nested(t *testing.T) {
	parts := []gmailPart{
		{MIMEType: "multipart/mixed", Parts: []gmailPart{
			{MIMEType: "text/plain", Body: gmailBody{Size: 100}},
			{MIMEType: "application/pdf", Filename: "doc.pdf", Body: gmailBody{AttachmentID: "a1", Size: 5000}},
		}},
	}
	if !hasAttachments(parts) {
		t.Error("should detect nested attachment")
	}
}

func TestHasAttachments_None(t *testing.T) {
	parts := []gmailPart{
		{MIMEType: "text/plain", Body: gmailBody{Size: 100}},
		{MIMEType: "text/html", Body: gmailBody{Size: 200}},
	}
	if hasAttachments(parts) {
		t.Error("should not detect attachments")
	}
}
