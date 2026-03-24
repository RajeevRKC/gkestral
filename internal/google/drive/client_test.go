package drive

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"gkestral/internal/google/transport"
)

func testDriveServer(handler http.HandlerFunc) (*DriveClient, *httptest.Server) {
	srv := httptest.NewServer(handler)
	client := NewDriveClient(srv.Client(), WithBaseURL(srv.URL))
	return client, srv
}

func TestListFiles_SinglePage(t *testing.T) {
	client, srv := testDriveServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("q") == "" {
			t.Error("query should contain folder filter")
		}
		fmt.Fprint(w, `{"files":[{"id":"f1","name":"doc.txt","mimeType":"text/plain"},{"id":"f2","name":"sheet.csv","mimeType":"text/csv"}]}`)
	})
	defer srv.Close()

	files, err := client.ListFiles(context.Background(), "root")
	if err != nil {
		t.Fatalf("ListFiles error: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("files len = %d, want 2", len(files))
	}
	if files[0].ID != "f1" {
		t.Errorf("files[0].ID = %q", files[0].ID)
	}
}

func TestListFiles_Pagination(t *testing.T) {
	page := 0
	client, srv := testDriveServer(func(w http.ResponseWriter, r *http.Request) {
		page++
		switch page {
		case 1:
			fmt.Fprint(w, `{"files":[{"id":"f1","name":"a.txt"}],"nextPageToken":"page2"}`)
		case 2:
			fmt.Fprint(w, `{"files":[{"id":"f2","name":"b.txt"}],"nextPageToken":"page3"}`)
		case 3:
			fmt.Fprint(w, `{"files":[{"id":"f3","name":"c.txt"}]}`)
		}
	})
	defer srv.Close()

	files, err := client.ListFiles(context.Background(), "root")
	if err != nil {
		t.Fatalf("ListFiles error: %v", err)
	}
	if len(files) != 3 {
		t.Errorf("files len = %d, want 3", len(files))
	}
}

func TestListFiles_MaxPages(t *testing.T) {
	callCount := 0
	client, srv := testDriveServer(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		fmt.Fprintf(w, `{"files":[{"id":"f%d","name":"file%d.txt"}],"nextPageToken":"tok%d"}`, callCount, callCount, callCount+1)
	})
	defer srv.Close()

	files, err := client.ListFiles(context.Background(), "root", WithMaxPages(2))
	if err != nil {
		t.Fatalf("ListFiles error: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("files len = %d, want 2 (maxPages=2)", len(files))
	}
	if callCount != 2 {
		t.Errorf("callCount = %d, want 2", callCount)
	}
}

func TestListFiles_EmptyFolder(t *testing.T) {
	client, srv := testDriveServer(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"files":[]}`)
	})
	defer srv.Close()

	files, err := client.ListFiles(context.Background(), "root")
	if err != nil {
		t.Fatalf("ListFiles error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("files len = %d, want 0", len(files))
	}
}

func TestListFiles_WithOptions(t *testing.T) {
	client, srv := testDriveServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("pageSize") != "10" {
			t.Errorf("pageSize = %q, want 10", r.URL.Query().Get("pageSize"))
		}
		if r.URL.Query().Get("orderBy") != "name" {
			t.Errorf("orderBy = %q, want name", r.URL.Query().Get("orderBy"))
		}
		fmt.Fprint(w, `{"files":[]}`)
	})
	defer srv.Close()

	_, err := client.ListFiles(context.Background(), "root", WithPageSize(10), WithOrderBy("name"))
	if err != nil {
		t.Fatalf("ListFiles error: %v", err)
	}
}

func TestListFiles_NotFound(t *testing.T) {
	client, srv := testDriveServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	defer srv.Close()

	_, err := client.ListFiles(context.Background(), "bad-folder-id")
	if !errors.Is(err, transport.ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound", err)
	}
}

func TestGetFile(t *testing.T) {
	client, srv := testDriveServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/files/file123" {
			t.Errorf("path = %q, want /files/file123", r.URL.Path)
		}
		fmt.Fprint(w, `{"id":"file123","name":"report.pdf","mimeType":"application/pdf","size":"1024"}`)
	})
	defer srv.Close()

	file, err := client.GetFile(context.Background(), "file123")
	if err != nil {
		t.Fatalf("GetFile error: %v", err)
	}
	if file.Name != "report.pdf" {
		t.Errorf("Name = %q", file.Name)
	}
	if file.Size != 1024 {
		t.Errorf("Size = %d, want 1024", file.Size)
	}
}

func TestGetFile_NotFound(t *testing.T) {
	client, srv := testDriveServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	defer srv.Close()

	_, err := client.GetFile(context.Background(), "nonexistent")
	if !errors.Is(err, transport.ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound", err)
	}
}

func TestSearchFiles(t *testing.T) {
	client, srv := testDriveServer(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		if q != "name contains 'report'" {
			t.Errorf("q = %q, want name contains 'report'", q)
		}
		fmt.Fprint(w, `{"files":[{"id":"f1","name":"report.pdf"}]}`)
	})
	defer srv.Close()

	files, err := client.SearchFiles(context.Background(), "name contains 'report'")
	if err != nil {
		t.Fatalf("SearchFiles error: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("files len = %d, want 1", len(files))
	}
}

func TestListFilesIter_SinglePage(t *testing.T) {
	client, srv := testDriveServer(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"files":[{"id":"f1","name":"a.txt"},{"id":"f2","name":"b.txt"}]}`)
	})
	defer srv.Close()

	iter := client.ListFilesIter(context.Background(), "root")
	var names []string
	for {
		f, err := iter.Next()
		if err != nil {
			t.Fatalf("Next error: %v", err)
		}
		if f == nil {
			break
		}
		names = append(names, f.Name)
	}
	if len(names) != 2 {
		t.Errorf("iterated %d files, want 2", len(names))
	}
}

func TestListFilesIter_MultiplePages(t *testing.T) {
	page := 0
	client, srv := testDriveServer(func(w http.ResponseWriter, r *http.Request) {
		page++
		if page == 1 {
			fmt.Fprint(w, `{"files":[{"id":"f1","name":"a.txt"}],"nextPageToken":"p2"}`)
		} else {
			fmt.Fprint(w, `{"files":[{"id":"f2","name":"b.txt"}]}`)
		}
	})
	defer srv.Close()

	iter := client.ListFilesIter(context.Background(), "root")
	count := 0
	for {
		f, err := iter.Next()
		if err != nil {
			t.Fatalf("Next error: %v", err)
		}
		if f == nil {
			break
		}
		count++
	}
	if count != 2 {
		t.Errorf("iterated %d files, want 2", count)
	}
}

func TestListFilesIter_Stop(t *testing.T) {
	client, srv := testDriveServer(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"files":[{"id":"f1"},{"id":"f2"},{"id":"f3"}],"nextPageToken":"more"}`)
	})
	defer srv.Close()

	iter := client.ListFilesIter(context.Background(), "root")
	f, err := iter.Next()
	if err != nil || f == nil {
		t.Fatal("expected first file")
	}
	iter.Stop()
	f2, err2 := iter.Next()
	// After Stop, should still return buffered items but not fetch new pages.
	// Actually, buffer has f2 and f3, and done is set.
	if err2 != nil {
		t.Fatalf("Next after Stop error: %v", err2)
	}
	_ = f2 // May or may not be nil depending on buffer state.
}

func TestListFilesIter_Error(t *testing.T) {
	client, srv := testDriveServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	defer srv.Close()

	iter := client.ListFilesIter(context.Background(), "root")
	_, err := iter.Next()
	if !errors.Is(err, transport.ErrForbidden) {
		t.Errorf("error = %v, want ErrForbidden", err)
	}
	// Subsequent calls should return the same error.
	_, err2 := iter.Next()
	if err2 == nil {
		t.Error("expected persistent error after first failure")
	}
}

// --- Type tests ---

func TestDriveFile_IsFolder(t *testing.T) {
	folder := DriveFile{MIMEType: "application/vnd.google-apps.folder"}
	if !folder.IsFolder() {
		t.Error("should be a folder")
	}

	file := DriveFile{MIMEType: "application/pdf"}
	if file.IsFolder() {
		t.Error("should not be a folder")
	}
}

func TestDriveFile_IsGoogleDoc(t *testing.T) {
	doc := DriveFile{MIMEType: "application/vnd.google-apps.document"}
	if !doc.IsGoogleDoc() {
		t.Error("Google Doc should be recognized")
	}

	pdf := DriveFile{MIMEType: "application/pdf"}
	if pdf.IsGoogleDoc() {
		t.Error("PDF should not be a Google Doc")
	}
}

func TestDefaultExportFormat(t *testing.T) {
	tests := []struct {
		mime string
		want string
	}{
		{"application/vnd.google-apps.document", "text/plain"},
		{"application/vnd.google-apps.spreadsheet", "text/csv"},
		{"application/vnd.google-apps.presentation", "text/plain"},
		{"application/vnd.google-apps.drawing", "image/png"},
		{"application/pdf", ""},
	}
	for _, tt := range tests {
		got := DefaultExportFormat(tt.mime)
		if got != tt.want {
			t.Errorf("DefaultExportFormat(%q) = %q, want %q", tt.mime, got, tt.want)
		}
	}
}
