package gemini

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCacheManager_Create(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/cachedContents") {
			t.Errorf("path = %q, want to end with /cachedContents", r.URL.Path)
		}

		var req CachedContentRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Model == "" {
			t.Error("model should not be empty")
		}
		if req.DisplayName != "test-cache" {
			t.Errorf("displayName = %q, want %q", req.DisplayName, "test-cache")
		}

		resp := CacheEntry{
			Name:        "cachedContents/abc123",
			Model:       req.Model,
			DisplayName: req.DisplayName,
			CreateTime:  "2026-03-24T00:00:00Z",
			ExpireTime:  "2026-03-24T01:00:00Z",
			UsageMetadata: &UsageMetadata{
				TotalTokenCount: 5000,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithAPIKey("key"), WithBaseURL(server.URL))
	cm := NewCacheManager(client)

	// Build content that exceeds minimum token threshold.
	longText := strings.Repeat("This is a test sentence for caching. ", 200)
	req := &CachedContentRequest{
		Model: "models/gemini-2.5-flash",
		Contents: []Message{
			{Role: "user", Parts: []Part{{Text: longText}}},
		},
		DisplayName: "test-cache",
		TTL:         "3600s",
	}

	entry, err := cm.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if entry.Name != "cachedContents/abc123" {
		t.Errorf("Name = %q, want %q", entry.Name, "cachedContents/abc123")
	}
	if entry.UsageMetadata.TotalTokenCount != 5000 {
		t.Errorf("TotalTokenCount = %d, want 5000", entry.UsageMetadata.TotalTokenCount)
	}
}

func TestCacheManager_Create_MinTokenValidation(t *testing.T) {
	client := NewClient(WithAPIKey("key"))
	cm := NewCacheManager(client)

	// Content too short for the model's minimum.
	req := &CachedContentRequest{
		Model: "models/gemini-2.5-flash",
		Contents: []Message{
			{Role: "user", Parts: []Part{{Text: "short"}}},
		},
		DisplayName: "too-small",
	}

	_, err := cm.Create(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for content below minimum token threshold")
	}
	if !strings.Contains(err.Error(), "minimum") {
		t.Errorf("error = %q, want to mention minimum tokens", err.Error())
	}
}

func TestCacheManager_Create_NilRequest(t *testing.T) {
	client := NewClient(WithAPIKey("key"))
	cm := NewCacheManager(client)

	_, err := cm.Create(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil request")
	}
}

func TestCacheManager_Create_NoModel(t *testing.T) {
	client := NewClient(WithAPIKey("key"))
	cm := NewCacheManager(client)

	req := &CachedContentRequest{
		Contents: []Message{
			{Role: "user", Parts: []Part{{Text: "test"}}},
		},
	}

	_, err := cm.Create(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for missing model")
	}
}

func TestCacheManager_Get(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if !strings.Contains(r.URL.Path, "cachedContents/abc123") {
			t.Errorf("path = %q, want cachedContents/abc123", r.URL.Path)
		}

		resp := CacheEntry{
			Name:       "cachedContents/abc123",
			Model:      "models/gemini-2.5-flash",
			CreateTime: "2026-03-24T00:00:00Z",
			ExpireTime: "2026-03-24T01:00:00Z",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithAPIKey("key"), WithBaseURL(server.URL))
	cm := NewCacheManager(client)

	entry, err := cm.Get(context.Background(), "cachedContents/abc123")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if entry.Name != "cachedContents/abc123" {
		t.Errorf("Name = %q, want %q", entry.Name, "cachedContents/abc123")
	}
}

func TestCacheManager_Get_EmptyName(t *testing.T) {
	client := NewClient(WithAPIKey("key"))
	cm := NewCacheManager(client)

	_, err := cm.Get(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestCacheManager_List(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}

		resp := cacheListResponse{
			CachedContents: []CacheEntry{
				{Name: "cachedContents/aaa", Model: "models/gemini-2.5-flash"},
				{Name: "cachedContents/bbb", Model: "models/gemini-3.1-pro-preview"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithAPIKey("key"), WithBaseURL(server.URL))
	cm := NewCacheManager(client)

	entries, err := cm.List(context.Background())
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if entries[0].Name != "cachedContents/aaa" {
		t.Errorf("entries[0].Name = %q, want %q", entries[0].Name, "cachedContents/aaa")
	}
}

func TestCacheManager_Delete(t *testing.T) {
	deleteCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %q, want DELETE", r.Method)
		}
		deleteCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(WithAPIKey("key"), WithBaseURL(server.URL))
	cm := NewCacheManager(client)

	err := cm.Delete(context.Background(), "cachedContents/abc123")
	if err != nil {
		t.Fatalf("Delete error: %v", err)
	}
	if !deleteCalled {
		t.Error("DELETE was not called")
	}
}

func TestCacheManager_Delete_EmptyName(t *testing.T) {
	client := NewClient(WithAPIKey("key"))
	cm := NewCacheManager(client)

	err := cm.Delete(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestCacheManager_Update(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("method = %q, want PATCH", r.Method)
		}

		var body cacheUpdateRequest
		json.NewDecoder(r.Body).Decode(&body)

		if body.TTL != "7200s" {
			t.Errorf("TTL = %q, want %q", body.TTL, "7200s")
		}

		resp := CacheEntry{
			Name:       "cachedContents/abc123",
			ExpireTime: "2026-03-24T03:00:00Z",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithAPIKey("key"), WithBaseURL(server.URL))
	cm := NewCacheManager(client)

	entry, err := cm.Update(context.Background(), "cachedContents/abc123", 2*time.Hour)
	if err != nil {
		t.Fatalf("Update error: %v", err)
	}
	if entry.ExpireTime != "2026-03-24T03:00:00Z" {
		t.Errorf("ExpireTime = %q, want %q", entry.ExpireTime, "2026-03-24T03:00:00Z")
	}
}

func TestCacheManager_Update_EmptyName(t *testing.T) {
	client := NewClient(WithAPIKey("key"))
	cm := NewCacheManager(client)

	_, err := cm.Update(context.Background(), "", time.Hour)
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestUseCachedContent(t *testing.T) {
	req := &GenerateContentRequest{
		Contents: []Message{
			{Role: "user", Parts: []Part{{Text: "test"}}},
		},
	}

	UseCachedContent(req, "cachedContents/xyz789")

	if req.CachedContent != "cachedContents/xyz789" {
		t.Errorf("CachedContent = %q, want %q", req.CachedContent, "cachedContents/xyz789")
	}
}

func TestCacheEconomics(t *testing.T) {
	tests := []struct {
		name         string
		modelID      string
		inputTokens  int
		cachedTokens int
		outputTokens int
		wantSavings  bool
	}{
		{
			name:         "flash with caching saves money",
			modelID:      "gemini-2.5-flash",
			inputTokens:  100000,
			cachedTokens: 80000,
			outputTokens: 1000,
			wantSavings:  true,
		},
		{
			name:         "pro with caching saves more",
			modelID:      "gemini-3.1-pro-preview",
			inputTokens:  100000,
			cachedTokens: 80000,
			outputTokens: 1000,
			wantSavings:  true,
		},
		{
			name:         "no cached tokens means no savings",
			modelID:      "gemini-2.5-flash",
			inputTokens:  100000,
			cachedTokens: 0,
			outputTokens: 1000,
			wantSavings:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CacheEconomics(tt.modelID, tt.inputTokens, tt.cachedTokens, tt.outputTokens)
			if err != nil {
				t.Fatalf("CacheEconomics error: %v", err)
			}

			if tt.wantSavings && result.SavingsPerReq <= 0 {
				t.Errorf("expected savings > 0, got %f", result.SavingsPerReq)
			}
			if !tt.wantSavings && result.SavingsPerReq != 0 {
				t.Errorf("expected no savings, got %f", result.SavingsPerReq)
			}
			if result.CostWithoutCache < result.CostWithCache && tt.wantSavings {
				t.Errorf("cost without cache (%f) should be >= cost with cache (%f)", result.CostWithoutCache, result.CostWithCache)
			}
		})
	}
}

func TestCacheEconomics_UnknownModel(t *testing.T) {
	_, err := CacheEconomics("nonexistent-model", 100000, 80000, 1000)
	if err == nil {
		t.Fatal("expected error for unknown model")
	}
}

func TestSplitContext(t *testing.T) {
	tests := []struct {
		name            string
		messageCount    int
		threshold       int
		wantStableCount int
		wantActiveCount int
	}{
		{
			name:            "below threshold returns all active",
			messageCount:    3,
			threshold:       4,
			wantStableCount: 0,
			wantActiveCount: 3,
		},
		{
			name:            "equal to threshold returns all active",
			messageCount:    4,
			threshold:       4,
			wantStableCount: 0,
			wantActiveCount: 4,
		},
		{
			name:            "above threshold splits correctly",
			messageCount:    10,
			threshold:       4,
			wantStableCount: 6,
			wantActiveCount: 4,
		},
		{
			name:            "large history with default threshold",
			messageCount:    20,
			threshold:       0,
			wantStableCount: 16,
			wantActiveCount: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messages := make([]Message, tt.messageCount)
			for i := range messages {
				role := "user"
				if i%2 == 1 {
					role = "model"
				}
				messages[i] = Message{Role: role, Parts: []Part{{Text: "msg"}}}
			}

			stable, active := SplitContext(messages, tt.threshold)
			if len(stable) != tt.wantStableCount {
				t.Errorf("stable count = %d, want %d", len(stable), tt.wantStableCount)
			}
			if len(active) != tt.wantActiveCount {
				t.Errorf("active count = %d, want %d", len(active), tt.wantActiveCount)
			}
		})
	}
}

func TestSplitContext_Independence(t *testing.T) {
	// Verify split returns independent copies (no shared backing arrays).
	messages := []Message{
		{Role: "user", Parts: []Part{{Text: "a"}}},
		{Role: "model", Parts: []Part{{Text: "b"}}},
		{Role: "user", Parts: []Part{{Text: "c"}}},
		{Role: "model", Parts: []Part{{Text: "d"}}},
		{Role: "user", Parts: []Part{{Text: "e"}}},
		{Role: "model", Parts: []Part{{Text: "f"}}},
	}

	stable, active := SplitContext(messages, 2)
	if len(stable) != 4 {
		t.Fatalf("stable = %d, want 4", len(stable))
	}

	// Mutate stable -- should not affect original.
	stable[0].Role = "mutated"
	if messages[0].Role == "mutated" {
		t.Error("SplitContext did not copy stable slice independently")
	}

	// Mutate active -- should not affect original.
	active[0].Role = "mutated"
	if messages[4].Role == "mutated" {
		t.Error("SplitContext did not copy active slice independently")
	}
}

func TestFormatTTL(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{"one hour", time.Hour, "3600s"},
		{"30 minutes", 30 * time.Minute, "1800s"},
		{"zero", 0, ""},
		{"negative", -1 * time.Second, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTTL(tt.d)
			if got != tt.want {
				t.Errorf("formatTTL(%v) = %q, want %q", tt.d, got, tt.want)
			}
		})
	}
}

func TestStripModelPrefix(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"models/gemini-2.5-flash", "gemini-2.5-flash"},
		{"gemini-2.5-flash", "gemini-2.5-flash"},
		{"models/", "models/"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := stripModelPrefix(tt.input)
			if got != tt.want {
				t.Errorf("stripModelPrefix(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
