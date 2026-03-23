package gemini

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient_Defaults(t *testing.T) {
	c := NewClient()
	if c.baseURL != DefaultBaseURL {
		t.Errorf("baseURL = %q, want %q", c.baseURL, DefaultBaseURL)
	}
	if c.apiVersion != DefaultAPIVersion {
		t.Errorf("apiVersion = %q, want %q", c.apiVersion, DefaultAPIVersion)
	}
	if c.defaultModel != "gemini-2.5-flash" {
		t.Errorf("defaultModel = %q, want %q", c.defaultModel, "gemini-2.5-flash")
	}
	if c.apiKey != "" {
		t.Errorf("apiKey = %q, want empty", c.apiKey)
	}
}

func TestNewClient_WithOptions(t *testing.T) {
	c := NewClient(
		WithAPIKey("test-key"),
		WithModel("gemini-3.1-pro-preview"),
		WithBaseURL("https://custom.api.com"),
		WithAPIVersion("v1"),
	)
	if c.apiKey != "test-key" {
		t.Errorf("apiKey = %q, want %q", c.apiKey, "test-key")
	}
	if c.defaultModel != "gemini-3.1-pro-preview" {
		t.Errorf("defaultModel = %q, want %q", c.defaultModel, "gemini-3.1-pro-preview")
	}
	if c.baseURL != "https://custom.api.com" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://custom.api.com")
	}
	if c.apiVersion != "v1" {
		t.Errorf("apiVersion = %q, want %q", c.apiVersion, "v1")
	}
}

func TestClient_ModelEndpoint(t *testing.T) {
	c := NewClient(WithAPIKey("key"))

	tests := []struct {
		name   string
		model  string
		action string
		want   string
	}{
		{
			name:   "explicit model",
			model:  "gemini-3.1-pro-preview",
			action: "generateContent",
			want:   DefaultBaseURL + "/v1beta/models/gemini-3.1-pro-preview:generateContent",
		},
		{
			name:   "default model",
			model:  "",
			action: "generateContent",
			want:   DefaultBaseURL + "/v1beta/models/gemini-2.5-flash:generateContent",
		},
		{
			name:   "countTokens",
			model:  "gemini-2.5-flash",
			action: "countTokens",
			want:   DefaultBaseURL + "/v1beta/models/gemini-2.5-flash:countTokens",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := c.ModelEndpoint(tt.model, tt.action)
			if got != tt.want {
				t.Errorf("ModelEndpoint(%q, %q) = %q, want %q", tt.model, tt.action, got, tt.want)
			}
		})
	}
}

func TestClient_StreamEndpoint(t *testing.T) {
	c := NewClient()
	got := c.StreamEndpoint("gemini-2.5-flash")
	want := DefaultBaseURL + "/v1beta/models/gemini-2.5-flash:streamGenerateContent?alt=sse"
	if got != want {
		t.Errorf("StreamEndpoint = %q, want %q", got, want)
	}
}

func TestClient_BuildRequest_AuthHeader(t *testing.T) {
	c := NewClient(WithAPIKey("test-api-key"))
	req, err := c.buildRequest(context.Background(), http.MethodPost, "https://example.com/api", map[string]string{"key": "value"})
	if err != nil {
		t.Fatalf("buildRequest error: %v", err)
	}

	if got := req.Header.Get("x-goog-api-key"); got != "test-api-key" {
		t.Errorf("x-goog-api-key header = %q, want %q", got, "test-api-key")
	}
	if got := req.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type header = %q, want %q", got, "application/json")
	}
}

func TestClient_BuildRequest_NoAPIKey(t *testing.T) {
	c := NewClient() // No API key
	req, err := c.buildRequest(context.Background(), http.MethodPost, "https://example.com/api", nil)
	if err != nil {
		t.Fatalf("buildRequest error: %v", err)
	}

	if got := req.Header.Get("x-goog-api-key"); got != "" {
		t.Errorf("x-goog-api-key header = %q, want empty", got)
	}
}

func TestClient_GenerateContent_MockServer(t *testing.T) {
	// Create a mock server that returns a canned response.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request.
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if got := r.Header.Get("x-goog-api-key"); got != "test-key" {
			t.Errorf("API key header = %q, want %q", got, "test-key")
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want %q", got, "application/json")
		}

		resp := GenerateContentResponse{
			Candidates: []Candidate{
				{
					Content: &CandidateContent{
						Role: "model",
						Parts: []Part{
							{Text: "Hello, world!"},
						},
					},
					FinishReason: "STOP",
				},
			},
			UsageMetadata: &UsageMetadata{
				PromptTokenCount:     10,
				CandidatesTokenCount: 5,
				TotalTokenCount:      15,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewClient(
		WithAPIKey("test-key"),
		WithBaseURL(server.URL),
		WithAPIVersion("v1beta"),
	)

	request := &GenerateContentRequest{
		Contents: []Message{
			{Role: "user", Parts: []Part{{Text: "Say hello"}}},
		},
	}

	result, err := c.GenerateContent(context.Background(), "gemini-2.5-flash", request)
	if err != nil {
		t.Fatalf("GenerateContent error: %v", err)
	}

	if len(result.Candidates) != 1 {
		t.Fatalf("got %d candidates, want 1", len(result.Candidates))
	}
	if result.Candidates[0].Content.Parts[0].Text != "Hello, world!" {
		t.Errorf("response text = %q, want %q", result.Candidates[0].Content.Parts[0].Text, "Hello, world!")
	}
	if result.UsageMetadata.TotalTokenCount != 15 {
		t.Errorf("total tokens = %d, want 15", result.UsageMetadata.TotalTokenCount)
	}
}

func TestClient_GenerateContent_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "invalid API key"}`))
	}))
	defer server.Close()

	c := NewClient(
		WithAPIKey("bad-key"),
		WithBaseURL(server.URL),
	)

	request := &GenerateContentRequest{
		Contents: []Message{
			{Role: "user", Parts: []Part{{Text: "test"}}},
		},
	}

	_, err := c.GenerateContent(context.Background(), "", request)
	if err == nil {
		t.Fatal("expected error for 401, got nil")
	}

	var apiErr *APIError
	if !matchAPIError(err, &apiErr) {
		t.Errorf("expected APIError, got %T: %v", err, err)
	}
}

func TestClient_CountTokens_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := CountTokensResponse{TotalTokens: 42}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewClient(
		WithAPIKey("key"),
		WithBaseURL(server.URL),
	)

	request := &CountTokensRequest{
		Contents: []Message{
			{Role: "user", Parts: []Part{{Text: "Count these tokens"}}},
		},
	}

	result, err := c.CountTokens(context.Background(), "", request)
	if err != nil {
		t.Fatalf("CountTokens error: %v", err)
	}
	if result.TotalTokens != 42 {
		t.Errorf("TotalTokens = %d, want 42", result.TotalTokens)
	}
}

func TestClient_WithRetryConfig(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("temporarily unavailable"))
			return
		}
		resp := GenerateContentResponse{
			Candidates: []Candidate{
				{Content: &CandidateContent{Role: "model", Parts: []Part{{Text: "ok"}}}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := DefaultRetryConfig()
	cfg.InitialDelay = 1 * 1000 * 1000 // 1ms in nanoseconds
	cfg.MaxAttempts = 5

	c := NewClient(
		WithAPIKey("key"),
		WithBaseURL(server.URL),
		WithRetryConfig(cfg),
	)

	request := &GenerateContentRequest{
		Contents: []Message{{Role: "user", Parts: []Part{{Text: "test"}}}},
	}

	result, err := c.GenerateContent(context.Background(), "", request)
	if err != nil {
		t.Fatalf("GenerateContent with retry error: %v", err)
	}
	if result.Candidates[0].Content.Parts[0].Text != "ok" {
		t.Errorf("response = %q, want %q", result.Candidates[0].Content.Parts[0].Text, "ok")
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestClient_DefaultsApplied(t *testing.T) {
	temp := float64(1.0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req GenerateContentRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.GenerationConfig == nil || req.GenerationConfig.Temperature == nil {
			t.Error("generation config not applied")
		}
		if len(req.SafetySettings) == 0 {
			t.Error("safety settings not applied")
		}

		json.NewEncoder(w).Encode(GenerateContentResponse{
			Candidates: []Candidate{{Content: &CandidateContent{Role: "model", Parts: []Part{{Text: "ok"}}}}},
		})
	}))
	defer server.Close()

	c := NewClient(
		WithAPIKey("key"),
		WithBaseURL(server.URL),
		WithGenerationConfig(GenerationConfig{Temperature: &temp}),
		WithSafetySettings([]SafetySetting{{Category: "HARM_CATEGORY_HARASSMENT", Threshold: "BLOCK_NONE"}}),
	)

	request := &GenerateContentRequest{
		Contents: []Message{{Role: "user", Parts: []Part{{Text: "test"}}}},
	}

	_, err := c.GenerateContent(context.Background(), "", request)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
}

// matchAPIError checks if err wraps an APIError (handles fmt.Errorf wrapping).
func matchAPIError(err error, target **APIError) bool {
	for e := err; e != nil; {
		if apiErr, ok := e.(*APIError); ok {
			*target = apiErr
			return true
		}
		if unwrapper, ok := e.(interface{ Unwrap() error }); ok {
			e = unwrapper.Unwrap()
		} else {
			break
		}
	}
	return false
}
