package drive

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"gkestral/internal/google/transport"
)

const (
	// DriveBaseURL is the Google Drive API v3 base URL.
	DriveBaseURL = "https://www.googleapis.com/drive/v3"

	// DefaultPageSize is the default number of files per page.
	DefaultPageSize = 100

	// DefaultMaxPages is the maximum number of pages to fetch.
	DefaultMaxPages = 10

	// DefaultFields is the set of fields returned for each file.
	DefaultFields = "files(id,name,mimeType,size,modifiedTime,parents,webViewLink,iconLink,starred,trashed),nextPageToken,incompleteSearch"
)

// DriveClient provides access to Google Drive API v3.
type DriveClient struct {
	client *transport.GoogleClient
}

// DriveOption configures a DriveClient.
type DriveOption func(*DriveClient)

// NewDriveClient creates a Drive client using the given authenticated HTTP client.
func NewDriveClient(httpClient *http.Client, opts ...DriveOption) *DriveClient {
	d := &DriveClient{
		client: transport.NewGoogleClient(httpClient, DriveBaseURL),
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

// WithBaseURL overrides the Drive API base URL (for testing).
func WithBaseURL(baseURL string) DriveOption {
	return func(d *DriveClient) {
		d.client = transport.NewGoogleClient(d.client.HTTPClient, baseURL)
	}
}

// ListOption configures a file listing request.
type ListOption func(*listConfig)

type listConfig struct {
	query    string
	pageSize int
	maxPages int
	orderBy  string
	fields   string
}

func defaultListConfig() listConfig {
	return listConfig{
		pageSize: DefaultPageSize,
		maxPages: DefaultMaxPages,
		orderBy:  "modifiedTime desc",
		fields:   DefaultFields,
	}
}

// WithQuery sets a custom Drive search query (q parameter).
func WithQuery(q string) ListOption {
	return func(c *listConfig) { c.query = q }
}

// WithPageSize sets the number of files per API page.
func WithPageSize(n int) ListOption {
	return func(c *listConfig) { c.pageSize = n }
}

// WithMaxPages sets the maximum number of pages to fetch.
func WithMaxPages(n int) ListOption {
	return func(c *listConfig) { c.maxPages = n }
}

// WithOrderBy sets the sort order for results.
func WithOrderBy(order string) ListOption {
	return func(c *listConfig) { c.orderBy = order }
}

// WithFields sets the fields projection for file metadata.
func WithFields(fields string) ListOption {
	return func(c *listConfig) { c.fields = fields }
}

// ListFiles lists files in the specified folder with bounded pagination.
func (d *DriveClient) ListFiles(ctx context.Context, folderID string, opts ...ListOption) ([]DriveFile, error) {
	cfg := defaultListConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.query == "" && folderID != "" {
		cfg.query = fmt.Sprintf("'%s' in parents and trashed = false", folderID)
	}

	params := url.Values{
		"pageSize": {fmt.Sprintf("%d", cfg.pageSize)},
		"orderBy":  {cfg.orderBy},
		"fields":   {cfg.fields},
	}
	if cfg.query != "" {
		params.Set("q", cfg.query)
	}

	extract := func(data json.RawMessage) ([]DriveFile, string, error) {
		var resp driveFileListResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, "", err
		}
		return resp.Files, resp.NextPageToken, nil
	}

	return transport.PaginatedList(ctx, d.client, "files", extract, cfg.maxPages, params)
}

// GetFile retrieves metadata for a single file by ID.
func (d *DriveClient) GetFile(ctx context.Context, fileID string) (*DriveFile, error) {
	params := url.Values{
		"fields": {"id,name,mimeType,size,modifiedTime,parents,webViewLink,iconLink,starred,trashed"},
	}
	var file DriveFile
	err := d.client.DoJSON(ctx, http.MethodGet, "files/"+fileID, nil, &file,
		transport.WithQuery(params))
	if err != nil {
		return nil, err
	}
	return &file, nil
}

// SearchFiles performs a free-form Drive search with bounded pagination.
func (d *DriveClient) SearchFiles(ctx context.Context, query string, opts ...ListOption) ([]DriveFile, error) {
	opts = append([]ListOption{WithQuery(query)}, opts...)
	return d.ListFiles(ctx, "", opts...)
}

// DriveFileIterator provides lazy, page-at-a-time iteration over Drive files.
type DriveFileIterator struct {
	client    *DriveClient
	ctx       context.Context
	cfg       listConfig
	params    url.Values
	pageToken string
	buffer    []DriveFile
	bufIdx    int
	done      bool
	err       error
}

// ListFilesIter returns a lazy iterator that fetches one page at a time.
func (d *DriveClient) ListFilesIter(ctx context.Context, folderID string, opts ...ListOption) *DriveFileIterator {
	cfg := defaultListConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.query == "" && folderID != "" {
		cfg.query = fmt.Sprintf("'%s' in parents and trashed = false", folderID)
	}

	params := url.Values{
		"pageSize": {fmt.Sprintf("%d", cfg.pageSize)},
		"orderBy":  {cfg.orderBy},
		"fields":   {cfg.fields},
	}
	if cfg.query != "" {
		params.Set("q", cfg.query)
	}

	return &DriveFileIterator{
		client: d,
		ctx:    ctx,
		cfg:    cfg,
		params: params,
	}
}

// Next returns the next file. Returns nil, nil when iteration is complete.
// Returns nil, error on API errors.
func (it *DriveFileIterator) Next() (*DriveFile, error) {
	if it.err != nil {
		return nil, it.err
	}

	// Return from buffer if available.
	if it.bufIdx < len(it.buffer) {
		f := &it.buffer[it.bufIdx]
		it.bufIdx++
		return f, nil
	}

	// Buffer exhausted -- fetch next page.
	if it.done {
		return nil, nil
	}

	it.err = it.fetchPage()
	if it.err != nil {
		return nil, it.err
	}

	if len(it.buffer) == 0 {
		it.done = true
		return nil, nil
	}

	f := &it.buffer[0]
	it.bufIdx = 1
	return f, nil
}

// Stop terminates iteration early.
func (it *DriveFileIterator) Stop() {
	it.done = true
}

func (it *DriveFileIterator) fetchPage() error {
	if it.pageToken != "" {
		it.params.Set("pageToken", it.pageToken)
	}

	resp, err := it.client.client.Do(it.ctx, http.MethodGet, "files", nil,
		transport.WithQuery(it.params))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var listResp driveFileListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return fmt.Errorf("drive: decode list response: %w", err)
	}

	it.buffer = listResp.Files
	it.bufIdx = 0

	if listResp.NextPageToken == "" {
		it.done = true
	}
	it.pageToken = listResp.NextPageToken

	return nil
}
