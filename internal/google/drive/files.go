package drive

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"

	"gkestral/internal/google/transport"
)

// DownloadOption configures a file download.
type DownloadOption func(*downloadConfig)

type downloadConfig struct {
	exportFormat string
}

// WithExportFormat sets the MIME type to export Google Docs to (e.g., "text/plain").
func WithExportFormat(mimeType string) DownloadOption {
	return func(c *downloadConfig) { c.exportFormat = mimeType }
}

// Download downloads file content by ID. For Google Workspace formats (Docs, Sheets,
// Slides), it exports to the specified format. For binary files, it returns raw bytes.
// Caller must close the returned ReadCloser.
func (d *DriveClient) Download(ctx context.Context, fileID string, opts ...DownloadOption) (io.ReadCloser, error) {
	cfg := downloadConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	// If no explicit export format, try to detect from file metadata.
	if cfg.exportFormat == "" {
		file, err := d.GetFile(ctx, fileID)
		if err != nil {
			return nil, err
		}
		if format := DefaultExportFormat(file.MIMEType); format != "" {
			cfg.exportFormat = format
		}
	}

	if cfg.exportFormat != "" {
		// Export Google Workspace format.
		return d.exportFile(ctx, fileID, cfg.exportFormat)
	}

	// Direct download for binary files.
	return d.downloadBinary(ctx, fileID)
}

func (d *DriveClient) downloadBinary(ctx context.Context, fileID string) (io.ReadCloser, error) {
	params := url.Values{"alt": {"media"}}
	resp, err := d.client.Do(ctx, http.MethodGet, "files/"+fileID, nil,
		transport.WithQuery(params))
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

func (d *DriveClient) exportFile(ctx context.Context, fileID, mimeType string) (io.ReadCloser, error) {
	params := url.Values{"mimeType": {mimeType}}
	resp, err := d.client.Do(ctx, http.MethodGet, "files/"+fileID+"/export", nil,
		transport.WithQuery(params))
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

// UploadOption configures a file upload.
type UploadOption func(*uploadConfig)

type uploadConfig struct {
	parentFolder string
	mimeType     string
	description  string
}

// WithParentFolder sets the parent folder ID for the uploaded file.
func WithParentFolder(folderID string) UploadOption {
	return func(c *uploadConfig) { c.parentFolder = folderID }
}

// WithMIMEType sets the MIME type of the uploaded content.
func WithMIMEType(mimeType string) UploadOption {
	return func(c *uploadConfig) { c.mimeType = mimeType }
}

// WithDescription sets the file description.
func WithDescription(desc string) UploadOption {
	return func(c *uploadConfig) { c.description = desc }
}

// Upload creates a new file in Google Drive using multipart/related upload.
// Returns the created file metadata.
func (d *DriveClient) Upload(ctx context.Context, name string, content io.Reader, opts ...UploadOption) (*DriveFile, error) {
	cfg := uploadConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.mimeType == "" {
		cfg.mimeType = detectMIME(name)
	}

	// Build metadata JSON.
	metadata := map[string]any{"name": name}
	if cfg.parentFolder != "" {
		metadata["parents"] = []string{cfg.parentFolder}
	}
	if cfg.description != "" {
		metadata["description"] = cfg.description
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("drive: marshal metadata: %w", err)
	}

	// Read content into memory for multipart construction.
	contentBytes, err := io.ReadAll(content)
	if err != nil {
		return nil, fmt.Errorf("drive: read content: %w", err)
	}

	// Build multipart/related body manually (Go's mime/multipart produces form-data).
	body, contentType := buildMultipartRelated(metadataJSON, contentBytes, cfg.mimeType)

	// Upload endpoint (different base URL for uploads).
	uploadURL := strings.Replace(d.client.BaseURL, "/drive/v3", "/upload/drive/v3", 1)
	uploadURL += "/files?uploadType=multipart"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, body)
	if err != nil {
		return nil, fmt.Errorf("drive: create upload request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("User-Agent", "Gkestral/0.1")

	resp, err := d.client.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("drive: upload: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("drive: upload failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var file DriveFile
	if err := json.NewDecoder(resp.Body).Decode(&file); err != nil {
		return nil, fmt.Errorf("drive: decode upload response: %w", err)
	}
	return &file, nil
}

// buildMultipartRelated constructs a multipart/related body with proper boundaries.
// Returns the body reader and the full Content-Type header value.
func buildMultipartRelated(metadataJSON, content []byte, contentMIME string) (io.Reader, string) {
	boundary := "gkestral_boundary_" + randomBoundary()
	var buf bytes.Buffer

	buf.WriteString("--" + boundary + "\r\n")
	buf.WriteString("Content-Type: application/json; charset=UTF-8\r\n\r\n")
	buf.Write(metadataJSON)
	buf.WriteString("\r\n")

	buf.WriteString("--" + boundary + "\r\n")
	buf.WriteString("Content-Type: " + contentMIME + "\r\n\r\n")
	buf.Write(content)
	buf.WriteString("\r\n")

	buf.WriteString("--" + boundary + "--\r\n")

	return &buf, "multipart/related; boundary=" + boundary
}

// randomBoundary generates a unique boundary string (thread-safe).
func randomBoundary() string {
	return fmt.Sprintf("%d", boundaryCounter.Add(1))
}

var boundaryCounter atomic.Int64

// detectMIME guesses MIME type from file extension.
func detectMIME(name string) string {
	ext := ""
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		ext = name[idx:]
	}
	if ext == "" {
		return "application/octet-stream"
	}
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		return "application/octet-stream"
	}
	return mimeType
}
