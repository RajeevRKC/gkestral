package drive

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDownload_BinaryFile(t *testing.T) {
	callPaths := []string{}
	client, srv := testDriveServer(func(w http.ResponseWriter, r *http.Request) {
		callPaths = append(callPaths, r.URL.Path)
		if strings.Contains(r.URL.Path, "/export") {
			t.Error("should not call export for binary file")
		}
		if r.URL.Query().Get("alt") == "media" {
			fmt.Fprint(w, "binary-content-here")
			return
		}
		// GetFile metadata call.
		if r.URL.Query().Get("fields") != "" {
			fmt.Fprint(w, `{"id":"f1","name":"photo.jpg","mimeType":"image/jpeg"}`)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
	})
	defer srv.Close()

	rc, err := client.Download(context.Background(), "f1")
	if err != nil {
		t.Fatalf("Download error: %v", err)
	}
	defer rc.Close()
	data, _ := io.ReadAll(rc)
	if string(data) != "binary-content-here" {
		t.Errorf("content = %q", string(data))
	}
}

func TestDownload_GoogleDocExport(t *testing.T) {
	client, srv := testDriveServer(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/export") {
			if r.URL.Query().Get("mimeType") != "text/plain" {
				t.Errorf("export mimeType = %q, want text/plain", r.URL.Query().Get("mimeType"))
			}
			fmt.Fprint(w, "exported plain text content")
			return
		}
		// GetFile metadata call.
		fmt.Fprint(w, `{"id":"d1","name":"My Doc","mimeType":"application/vnd.google-apps.document"}`)
	})
	defer srv.Close()

	rc, err := client.Download(context.Background(), "d1")
	if err != nil {
		t.Fatalf("Download error: %v", err)
	}
	defer rc.Close()
	data, _ := io.ReadAll(rc)
	if string(data) != "exported plain text content" {
		t.Errorf("content = %q", string(data))
	}
}

func TestDownload_ExplicitExportFormat(t *testing.T) {
	client, srv := testDriveServer(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/export") {
			if r.URL.Query().Get("mimeType") != "application/pdf" {
				t.Errorf("export mimeType = %q, want application/pdf", r.URL.Query().Get("mimeType"))
			}
			fmt.Fprint(w, "pdf-bytes")
			return
		}
		// Should not even call GetFile when explicit format is given.
		w.WriteHeader(http.StatusBadRequest)
	})
	defer srv.Close()

	rc, err := client.Download(context.Background(), "d1", WithExportFormat("application/pdf"))
	if err != nil {
		t.Fatalf("Download error: %v", err)
	}
	defer rc.Close()
	data, _ := io.ReadAll(rc)
	if string(data) != "pdf-bytes" {
		t.Errorf("content = %q", string(data))
	}
}

func TestUpload_Basic(t *testing.T) {
	client := NewDriveClient(http.DefaultClient)

	// Create a test server that validates multipart/related.
	uploadSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ct := r.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "multipart/related") {
			t.Errorf("Content-Type = %q, want multipart/related", ct)
		}
		body, _ := io.ReadAll(r.Body)
		bodyStr := string(body)
		if !strings.Contains(bodyStr, `"name":"test.txt"`) {
			t.Error("body should contain file name in metadata")
		}
		if !strings.Contains(bodyStr, "hello world") {
			t.Error("body should contain file content")
		}
		if !strings.Contains(bodyStr, "application/json; charset=UTF-8") {
			t.Error("metadata part should have JSON content type")
		}
		fmt.Fprint(w, `{"id":"new-file-id","name":"test.txt","mimeType":"text/plain"}`)
	}))
	defer uploadSrv.Close()

	// Override client to point at test upload server.
	client.client.BaseURL = strings.Replace(uploadSrv.URL, uploadSrv.URL, uploadSrv.URL+"/drive/v3", 1)
	// Since our Upload constructs the URL from baseURL, we need a proper mock.
	// Let's use a simpler approach: override HTTPClient to intercept.
	originalClient := client.client.HTTPClient
	client.client.HTTPClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "/upload/") {
				// Redirect to test server.
				req.URL.Scheme = "http"
				req.URL.Host = strings.TrimPrefix(uploadSrv.URL, "http://")
				return http.DefaultTransport.RoundTrip(req)
			}
			return originalClient.Transport.RoundTrip(req)
		}),
	}

	file, err := client.Upload(context.Background(), "test.txt",
		strings.NewReader("hello world"),
		WithParentFolder("folder123"),
		WithDescription("A test file"))
	if err != nil {
		t.Fatalf("Upload error: %v", err)
	}
	if file.ID != "new-file-id" {
		t.Errorf("ID = %q", file.ID)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestUpload_WithMIMEType(t *testing.T) {
	uploadSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "application/pdf") {
			t.Error("content part should have application/pdf MIME type")
		}
		fmt.Fprint(w, `{"id":"f1","name":"doc.pdf"}`)
	}))
	defer uploadSrv.Close()

	client := NewDriveClient(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			req.URL.Scheme = "http"
			req.URL.Host = strings.TrimPrefix(uploadSrv.URL, "http://")
			return http.DefaultTransport.RoundTrip(req)
		}),
	})

	_, err := client.Upload(context.Background(), "doc.pdf",
		strings.NewReader("pdf content"), WithMIMEType("application/pdf"))
	if err != nil {
		t.Fatalf("Upload error: %v", err)
	}
}

func TestBuildMultipartRelated(t *testing.T) {
	metadata := []byte(`{"name":"test.txt"}`)
	content := []byte("file content")
	body, ct := buildMultipartRelated(metadata, content, "text/plain")

	if !strings.HasPrefix(ct, "multipart/related; boundary=") {
		t.Errorf("Content-Type = %q", ct)
	}

	data, _ := io.ReadAll(body)
	s := string(data)
	if !strings.Contains(s, "application/json; charset=UTF-8") {
		t.Error("missing JSON content type in metadata part")
	}
	if !strings.Contains(s, `{"name":"test.txt"}`) {
		t.Error("missing metadata JSON")
	}
	if !strings.Contains(s, "text/plain") {
		t.Error("missing content MIME type")
	}
	if !strings.Contains(s, "file content") {
		t.Error("missing file content")
	}
}

func TestDetectMIME(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"file.txt", "text/plain"},
		{"file.pdf", "application/pdf"},
		{"file.json", "application/json"},
		{"file", "application/octet-stream"},
		{"", "application/octet-stream"},
	}
	for _, tt := range tests {
		got := detectMIME(tt.name)
		// MIME detection is OS-dependent, so check prefix for known types.
		if tt.name == "file" || tt.name == "" {
			if got != "application/octet-stream" {
				t.Errorf("detectMIME(%q) = %q, want application/octet-stream", tt.name, got)
			}
		} else if got == "" {
			t.Errorf("detectMIME(%q) returned empty", tt.name)
		}
	}
}
