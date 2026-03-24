package transport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func testServer(handler http.HandlerFunc) (*GoogleClient, *httptest.Server) {
	srv := httptest.NewServer(handler)
	client := NewGoogleClient(srv.Client(), srv.URL)
	return client, srv
}

func TestNewGoogleClient(t *testing.T) {
	c := NewGoogleClient(http.DefaultClient, "https://example.com/api/v1/")
	if c.BaseURL != "https://example.com/api/v1" {
		t.Errorf("BaseURL = %q, want trailing slash trimmed", c.BaseURL)
	}
	if c.UserAgent != "Gkestral/0.1" {
		t.Errorf("UserAgent = %q", c.UserAgent)
	}
}

func TestDo_Success(t *testing.T) {
	c, srv := testServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "Gkestral/0.1" {
			t.Errorf("User-Agent = %q", r.Header.Get("User-Agent"))
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"ok":true}`)
	})
	defer srv.Close()

	resp, err := c.Do(context.Background(), http.MethodGet, "/test", nil)
	if err != nil {
		t.Fatalf("Do() error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d", resp.StatusCode)
	}
}

func TestDo_WithBody(t *testing.T) {
	c, srv := testServer(func(w http.ResponseWriter, r *http.Request) {
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
		body, _ := io.ReadAll(r.Body)
		fmt.Fprint(w, string(body))
	})
	defer srv.Close()

	resp, err := c.Do(context.Background(), http.MethodPost, "/create", strings.NewReader(`{"name":"test"}`))
	if err != nil {
		t.Fatalf("Do() error: %v", err)
	}
	resp.Body.Close()
}

func TestDo_WithQuery(t *testing.T) {
	c, srv := testServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("q") != "hello" {
			t.Errorf("query q = %q, want hello", r.URL.Query().Get("q"))
		}
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	params := url.Values{"q": {"hello"}}
	resp, err := c.Do(context.Background(), http.MethodGet, "/search", nil, WithQuery(params))
	if err != nil {
		t.Fatalf("Do() error: %v", err)
	}
	resp.Body.Close()
}

func TestDo_WithHeader(t *testing.T) {
	c, srv := testServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Custom") != "value" {
			t.Errorf("X-Custom = %q", r.Header.Get("X-Custom"))
		}
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	resp, err := c.Do(context.Background(), http.MethodGet, "/test", nil, WithHeader("X-Custom", "value"))
	if err != nil {
		t.Fatalf("Do() error: %v", err)
	}
	resp.Body.Close()
}

func TestDo_NotFound(t *testing.T) {
	c, srv := testServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	defer srv.Close()

	_, err := c.Do(context.Background(), http.MethodGet, "/missing", nil)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound", err)
	}
}

func TestDo_Forbidden(t *testing.T) {
	c, srv := testServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	defer srv.Close()

	_, err := c.Do(context.Background(), http.MethodGet, "/secret", nil)
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("error = %v, want ErrForbidden", err)
	}
}

func TestDo_Unauthorized(t *testing.T) {
	c, srv := testServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	defer srv.Close()

	_, err := c.Do(context.Background(), http.MethodGet, "/private", nil)
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("error = %v, want ErrUnauthorized", err)
	}
}

func TestDo_RateLimited(t *testing.T) {
	c, srv := testServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	})
	defer srv.Close()

	_, err := c.Do(context.Background(), http.MethodGet, "/api", nil)
	var rl *ErrRateLimited
	if !errors.As(err, &rl) {
		t.Fatalf("error = %v, want ErrRateLimited", err)
	}
	if rl.RetryAfter != 30*time.Second {
		t.Errorf("RetryAfter = %v, want 30s", rl.RetryAfter)
	}
}

func TestDo_RateLimitedNoRetryAfter(t *testing.T) {
	c, srv := testServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	})
	defer srv.Close()

	_, err := c.Do(context.Background(), http.MethodGet, "/api", nil)
	var rl *ErrRateLimited
	if !errors.As(err, &rl) {
		t.Fatalf("error = %v, want ErrRateLimited", err)
	}
	if rl.RetryAfter != 0 {
		t.Errorf("RetryAfter = %v, want 0", rl.RetryAfter)
	}
}

func TestDo_APIError(t *testing.T) {
	c, srv := testServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":"bad input"}`)
	})
	defer srv.Close()

	_, err := c.Do(context.Background(), http.MethodPost, "/api", nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v, want APIError", err)
	}
	if apiErr.StatusCode != 400 {
		t.Errorf("StatusCode = %d, want 400", apiErr.StatusCode)
	}
	if !strings.Contains(apiErr.Body, "bad input") {
		t.Errorf("Body = %q, want to contain 'bad input'", apiErr.Body)
	}
}

func TestDo_ContextCancelled(t *testing.T) {
	c, srv := testServer(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := c.Do(ctx, http.MethodGet, "/slow", nil)
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

func TestDoJSON_Success(t *testing.T) {
	c, srv := testServer(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"name":"test","value":42}`)
	})
	defer srv.Close()

	var result struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}
	err := c.DoJSON(context.Background(), http.MethodGet, "/data", nil, &result)
	if err != nil {
		t.Fatalf("DoJSON() error: %v", err)
	}
	if result.Name != "test" || result.Value != 42 {
		t.Errorf("result = %+v", result)
	}
}

func TestDoJSON_NilResult(t *testing.T) {
	c, srv := testServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	defer srv.Close()

	err := c.DoJSON(context.Background(), http.MethodDelete, "/item", nil, nil)
	if err != nil {
		t.Fatalf("DoJSON(nil result) error: %v", err)
	}
}

func TestPaginatedList_SinglePage(t *testing.T) {
	c, srv := testServer(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"items":["a","b","c"]}`)
	})
	defer srv.Close()

	extract := func(data json.RawMessage) ([]string, string, error) {
		var page struct {
			Items         []string `json:"items"`
			NextPageToken string   `json:"nextPageToken"`
		}
		if err := json.Unmarshal(data, &page); err != nil {
			return nil, "", err
		}
		return page.Items, page.NextPageToken, nil
	}

	items, err := PaginatedList(context.Background(), c, "/list", extract, 10, nil)
	if err != nil {
		t.Fatalf("PaginatedList error: %v", err)
	}
	if len(items) != 3 {
		t.Errorf("items len = %d, want 3", len(items))
	}
}

func TestPaginatedList_MultiplePages(t *testing.T) {
	page := 0
	c, srv := testServer(func(w http.ResponseWriter, r *http.Request) {
		page++
		switch page {
		case 1:
			fmt.Fprint(w, `{"items":["a","b"],"nextPageToken":"tok2"}`)
		case 2:
			fmt.Fprint(w, `{"items":["c","d"],"nextPageToken":"tok3"}`)
		case 3:
			fmt.Fprint(w, `{"items":["e"]}`)
		}
	})
	defer srv.Close()

	extract := func(data json.RawMessage) ([]string, string, error) {
		var p struct {
			Items         []string `json:"items"`
			NextPageToken string   `json:"nextPageToken"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, "", err
		}
		return p.Items, p.NextPageToken, nil
	}

	items, err := PaginatedList(context.Background(), c, "/list", extract, 10, nil)
	if err != nil {
		t.Fatalf("PaginatedList error: %v", err)
	}
	if len(items) != 5 {
		t.Errorf("items len = %d, want 5", len(items))
	}
}

func TestPaginatedList_MaxPagesLimit(t *testing.T) {
	callCount := 0
	c, srv := testServer(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		// Always return a next page token to test the limit.
		fmt.Fprintf(w, `{"items":["item%d"],"nextPageToken":"tok%d"}`, callCount, callCount+1)
	})
	defer srv.Close()

	extract := func(data json.RawMessage) ([]string, string, error) {
		var p struct {
			Items         []string `json:"items"`
			NextPageToken string   `json:"nextPageToken"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, "", err
		}
		return p.Items, p.NextPageToken, nil
	}

	items, err := PaginatedList(context.Background(), c, "/list", extract, 3, nil)
	if err != nil {
		t.Fatalf("PaginatedList error: %v", err)
	}
	if len(items) != 3 {
		t.Errorf("items len = %d, want 3 (maxPages=3)", len(items))
	}
	if callCount != 3 {
		t.Errorf("callCount = %d, want 3", callCount)
	}
}

func TestPaginatedList_DefaultMaxPages(t *testing.T) {
	callCount := 0
	c, srv := testServer(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount <= 10 {
			fmt.Fprintf(w, `{"items":["x"],"nextPageToken":"tok%d"}`, callCount+1)
		} else {
			fmt.Fprint(w, `{"items":["x"]}`)
		}
	})
	defer srv.Close()

	extract := func(data json.RawMessage) ([]string, string, error) {
		var p struct {
			Items         []string `json:"items"`
			NextPageToken string   `json:"nextPageToken"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, "", err
		}
		return p.Items, p.NextPageToken, nil
	}

	items, err := PaginatedList(context.Background(), c, "/list", extract, 0, nil)
	if err != nil {
		t.Fatalf("PaginatedList error: %v", err)
	}
	// Default maxPages is 10
	if len(items) != 10 {
		t.Errorf("items len = %d, want 10 (default maxPages)", len(items))
	}
}

func TestPaginatedList_EmptyPage(t *testing.T) {
	c, srv := testServer(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"items":[]}`)
	})
	defer srv.Close()

	extract := func(data json.RawMessage) ([]string, string, error) {
		var p struct {
			Items         []string `json:"items"`
			NextPageToken string   `json:"nextPageToken"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, "", err
		}
		return p.Items, p.NextPageToken, nil
	}

	items, err := PaginatedList(context.Background(), c, "/list", extract, 10, nil)
	if err != nil {
		t.Fatalf("PaginatedList error: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("items len = %d, want 0", len(items))
	}
}

func TestPaginatedList_WithBaseParams(t *testing.T) {
	c, srv := testServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("pageSize") != "50" {
			t.Errorf("pageSize = %q, want 50", r.URL.Query().Get("pageSize"))
		}
		fmt.Fprint(w, `{"items":["a"]}`)
	})
	defer srv.Close()

	extract := func(data json.RawMessage) ([]string, string, error) {
		var p struct {
			Items         []string `json:"items"`
			NextPageToken string   `json:"nextPageToken"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, "", err
		}
		return p.Items, p.NextPageToken, nil
	}

	params := url.Values{"pageSize": {"50"}}
	items, err := PaginatedList(context.Background(), c, "/list", extract, 10, params)
	if err != nil {
		t.Fatalf("PaginatedList error: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("items len = %d, want 1", len(items))
	}
}

func TestErrRateLimited_Error(t *testing.T) {
	e1 := &ErrRateLimited{RetryAfter: 30 * time.Second}
	if !strings.Contains(e1.Error(), "30s") {
		t.Errorf("Error() = %q, want to contain 30s", e1.Error())
	}

	e2 := &ErrRateLimited{}
	if strings.Contains(e2.Error(), "retry after") {
		t.Errorf("Error() = %q, should not contain retry after when zero", e2.Error())
	}
}

func TestAPIError_Error(t *testing.T) {
	e := &APIError{StatusCode: 500, Status: "500 Internal Server Error", Body: "oops"}
	s := e.Error()
	if !strings.Contains(s, "500") {
		t.Errorf("Error() = %q, want to contain 500", s)
	}
	if !strings.Contains(s, "oops") {
		t.Errorf("Error() = %q, want to contain body", s)
	}
}
