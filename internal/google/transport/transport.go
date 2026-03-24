// Package transport provides a shared HTTP transport layer for Google REST APIs.
// Both Drive and Gmail clients build on this to avoid duplicating request
// building, error classification, and pagination logic.
package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Common errors returned by Google API operations.
var (
	ErrNotFound     = fmt.Errorf("google: resource not found (404)")
	ErrForbidden    = fmt.Errorf("google: access denied (403)")
	ErrUnauthorized = fmt.Errorf("google: authentication required (401)")
)

// ErrRateLimited is returned when the API returns HTTP 429.
type ErrRateLimited struct {
	RetryAfter time.Duration // Parsed from Retry-After header; zero if absent.
}

func (e *ErrRateLimited) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("google: rate limited (429), retry after %s", e.RetryAfter)
	}
	return "google: rate limited (429)"
}

// APIError wraps a non-2xx response from a Google API.
type APIError struct {
	StatusCode int
	Status     string
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("google: API error %d %s: %s", e.StatusCode, e.Status, e.Body)
}

// GoogleClient provides shared HTTP transport for Google REST APIs.
type GoogleClient struct {
	HTTPClient *http.Client
	BaseURL    string
	UserAgent  string
}

// NewGoogleClient creates a GoogleClient with the given authenticated HTTP client
// and base URL (e.g., "https://www.googleapis.com/drive/v3").
func NewGoogleClient(httpClient *http.Client, baseURL string) *GoogleClient {
	return &GoogleClient{
		HTTPClient: httpClient,
		BaseURL:    strings.TrimRight(baseURL, "/"),
		UserAgent:  "Gkestral/0.1",
	}
}

// RequestOption configures an individual API request.
type RequestOption func(req *http.Request)

// WithQuery adds query parameters to the request URL.
func WithQuery(params url.Values) RequestOption {
	return func(req *http.Request) {
		q := req.URL.Query()
		for k, vs := range params {
			for _, v := range vs {
				q.Add(k, v)
			}
		}
		req.URL.RawQuery = q.Encode()
	}
}

// WithHeader sets a request header.
func WithHeader(key, value string) RequestOption {
	return func(req *http.Request) {
		req.Header.Set(key, value)
	}
}

// Do executes an HTTP request against the Google API and returns the raw response.
// Non-2xx status codes are converted to typed errors.
func (c *GoogleClient) Do(ctx context.Context, method, path string, body io.Reader, opts ...RequestOption) (*http.Response, error) {
	fullURL := c.BaseURL + "/" + strings.TrimLeft(path, "/")
	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return nil, fmt.Errorf("google: create request: %w", err)
	}

	req.Header.Set("User-Agent", c.UserAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	for _, opt := range opts {
		opt(req)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google: execute request: %w", err)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return resp, nil
	}

	// Read error body for diagnostics.
	defer resp.Body.Close()
	errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))

	switch resp.StatusCode {
	case http.StatusNotFound:
		return nil, ErrNotFound
	case http.StatusForbidden:
		return nil, ErrForbidden
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	case http.StatusTooManyRequests:
		e := &ErrRateLimited{}
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if secs, err := strconv.Atoi(ra); err == nil {
				e.RetryAfter = time.Duration(secs) * time.Second
			}
		}
		return nil, e
	default:
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       string(errBody),
		}
	}
}

// DoJSON executes an HTTP request and JSON-decodes the response into result.
func (c *GoogleClient) DoJSON(ctx context.Context, method, path string, body io.Reader, result any, opts ...RequestOption) error {
	resp, err := c.Do(ctx, method, path, body, opts...)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if result == nil {
		return nil
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

// PageExtractor extracts items and the next page token from a raw JSON response.
type PageExtractor[T any] func(data json.RawMessage) (items []T, nextPageToken string, err error)

// PaginatedList fetches items from a paginated Google API endpoint.
// It follows nextPageToken up to maxPages (default 10 if maxPages <= 0).
func PaginatedList[T any](ctx context.Context, c *GoogleClient, path string, extract PageExtractor[T], maxPages int, baseParams url.Values) ([]T, error) {
	if maxPages <= 0 {
		maxPages = 10
	}

	var all []T
	pageToken := ""

	for page := 0; page < maxPages; page++ {
		params := url.Values{}
		for k, vs := range baseParams {
			params[k] = vs
		}
		if pageToken != "" {
			params.Set("pageToken", pageToken)
		}

		resp, err := c.Do(ctx, http.MethodGet, path, nil, WithQuery(params))
		if err != nil {
			return all, err
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return all, fmt.Errorf("google: read response: %w", err)
		}

		items, nextToken, err := extract(body)
		if err != nil {
			return all, fmt.Errorf("google: parse page: %w", err)
		}

		all = append(all, items...)

		if nextToken == "" {
			break
		}
		pageToken = nextToken
	}

	return all, nil
}
