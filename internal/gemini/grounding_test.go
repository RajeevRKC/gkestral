package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEnableSearchGrounding(t *testing.T) {
	req := &GenerateContentRequest{
		Contents: []Message{{Role: "user", Parts: []Part{{Text: "test"}}}},
	}

	EnableSearchGrounding(req)

	if len(req.Tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(req.Tools))
	}
	if req.Tools[0].GoogleSearch == nil {
		t.Fatal("expected GoogleSearch tool to be set")
	}
	if req.Tools[0].GoogleSearch.DynamicRetrievalConfig != nil {
		t.Error("expected no DynamicRetrievalConfig for basic grounding")
	}
}

func TestEnableSearchGrounding_NoDuplicate(t *testing.T) {
	req := &GenerateContentRequest{
		Contents: []Message{{Role: "user", Parts: []Part{{Text: "test"}}}},
	}

	EnableSearchGrounding(req)
	EnableSearchGrounding(req) // Call again

	if len(req.Tools) != 1 {
		t.Fatalf("expected 1 tool after double enable, got %d", len(req.Tools))
	}
}

func TestEnableSearchGroundingValidated(t *testing.T) {
	req := &GenerateContentRequest{
		Contents: []Message{{Role: "user", Parts: []Part{{Text: "test"}}}},
	}

	threshold := 0.7
	EnableSearchGroundingValidated(req, &threshold)

	if len(req.Tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(req.Tools))
	}
	gs := req.Tools[0].GoogleSearch
	if gs == nil {
		t.Fatal("expected GoogleSearch tool")
	}
	if gs.DynamicRetrievalConfig == nil {
		t.Fatal("expected DynamicRetrievalConfig")
	}
	if gs.DynamicRetrievalConfig.Mode != "VALIDATED" {
		t.Errorf("mode = %q, want VALIDATED", gs.DynamicRetrievalConfig.Mode)
	}
	if gs.DynamicRetrievalConfig.DynamicThreshold == nil || *gs.DynamicRetrievalConfig.DynamicThreshold != 0.7 {
		t.Error("expected dynamic threshold of 0.7")
	}
}

func TestEnableSearchGroundingValidated_NilThreshold(t *testing.T) {
	req := &GenerateContentRequest{
		Contents: []Message{{Role: "user", Parts: []Part{{Text: "test"}}}},
	}

	EnableSearchGroundingValidated(req, nil)

	gs := req.Tools[0].GoogleSearch
	if gs.DynamicRetrievalConfig.DynamicThreshold != nil {
		t.Error("expected nil threshold when not specified")
	}
}

func TestHasSearchGrounding(t *testing.T) {
	tests := []struct {
		name string
		req  *GenerateContentRequest
		want bool
	}{
		{
			name: "no tools",
			req:  &GenerateContentRequest{},
			want: false,
		},
		{
			name: "function tools only",
			req: &GenerateContentRequest{
				Tools: []Tool{{FunctionDeclarations: []FunctionDeclaration{{Name: "test"}}}},
			},
			want: false,
		},
		{
			name: "search grounding enabled",
			req: &GenerateContentRequest{
				Tools: []Tool{{GoogleSearch: &GoogleSearch{}}},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasSearchGrounding(tt.req)
			if got != tt.want {
				t.Errorf("HasSearchGrounding = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractGroundingMetadata(t *testing.T) {
	t.Run("nil response", func(t *testing.T) {
		gm := ExtractGroundingMetadata(nil)
		if gm != nil {
			t.Error("expected nil for nil response")
		}
	})

	t.Run("no candidates", func(t *testing.T) {
		gm := ExtractGroundingMetadata(&GenerateContentResponse{})
		if gm != nil {
			t.Error("expected nil for empty candidates")
		}
	})

	t.Run("no grounding metadata", func(t *testing.T) {
		gm := ExtractGroundingMetadata(&GenerateContentResponse{
			Candidates: []Candidate{{
				Content: &CandidateContent{Role: "model", Parts: []Part{{Text: "test"}}},
			}},
		})
		if gm != nil {
			t.Error("expected nil when no grounding metadata")
		}
	})

	t.Run("with grounding metadata", func(t *testing.T) {
		resp := &GenerateContentResponse{
			Candidates: []Candidate{{
				Content: &CandidateContent{Role: "model", Parts: []Part{{Text: "Riyadh has about 7.7M people"}}},
				GroundingMetadata: &GroundingMetadata{
					WebSearchQueries: []string{"population of Riyadh"},
					GroundingChunks: []GroundingChunk{
						{Web: &WebChunk{URI: "https://example.com/riyadh", Title: "Riyadh Population"}},
					},
					GroundingSupports: []GroundingSupport{
						{
							Segment: &Segment{
								StartIndex: 0,
								EndIndex:   28,
								Text:       "Riyadh has about 7.7M people",
							},
							GroundingChunkIndices: []int{0},
							ConfidenceScores:      []float64{0.95},
						},
					},
				},
			}},
		}

		gm := ExtractGroundingMetadata(resp)
		if gm == nil {
			t.Fatal("expected grounding metadata")
		}
		if len(gm.GroundingChunks) != 1 {
			t.Fatalf("expected 1 chunk, got %d", len(gm.GroundingChunks))
		}
		if gm.GroundingChunks[0].Web.URI != "https://example.com/riyadh" {
			t.Errorf("unexpected URI: %s", gm.GroundingChunks[0].Web.URI)
		}
	})
}

func TestExtractCitationsFromMetadata(t *testing.T) {
	t.Run("nil metadata", func(t *testing.T) {
		citations := ExtractCitationsFromMetadata(nil)
		if citations != nil {
			t.Error("expected nil for nil metadata")
		}
	})

	t.Run("chunks only no supports", func(t *testing.T) {
		gm := &GroundingMetadata{
			GroundingChunks: []GroundingChunk{
				{Web: &WebChunk{URI: "https://a.com", Title: "Site A"}},
				{Web: &WebChunk{URI: "https://b.com", Title: "Site B"}},
			},
		}
		citations := ExtractCitationsFromMetadata(gm)
		if len(citations) != 2 {
			t.Fatalf("expected 2 citations, got %d", len(citations))
		}
		if citations[0].URL != "https://a.com" {
			t.Errorf("citation[0].URL = %q", citations[0].URL)
		}
		if citations[0].Confidence != 0 {
			t.Error("expected zero confidence for chunk-only citations")
		}
	})

	t.Run("with supports and confidence", func(t *testing.T) {
		gm := &GroundingMetadata{
			GroundingChunks: []GroundingChunk{
				{Web: &WebChunk{URI: "https://wiki.com/riyadh", Title: "Riyadh"}},
				{Web: &WebChunk{URI: "https://stats.gov.sa", Title: "Saudi Stats"}},
			},
			GroundingSupports: []GroundingSupport{
				{
					Segment:               &Segment{Text: "7.7 million people"},
					GroundingChunkIndices: []int{0, 1},
					ConfidenceScores:      []float64{0.92, 0.85},
				},
			},
		}
		citations := ExtractCitationsFromMetadata(gm)
		if len(citations) != 2 {
			t.Fatalf("expected 2 citations, got %d", len(citations))
		}
		if citations[0].URL != "https://wiki.com/riyadh" {
			t.Errorf("citation[0].URL = %q", citations[0].URL)
		}
		if citations[0].Confidence != 0.92 {
			t.Errorf("citation[0].Confidence = %f, want 0.92", citations[0].Confidence)
		}
		if citations[0].SupportedText != "7.7 million people" {
			t.Errorf("citation[0].SupportedText = %q", citations[0].SupportedText)
		}
		if citations[1].Confidence != 0.85 {
			t.Errorf("citation[1].Confidence = %f, want 0.85", citations[1].Confidence)
		}
	})

	t.Run("invalid chunk index", func(t *testing.T) {
		gm := &GroundingMetadata{
			GroundingChunks: []GroundingChunk{
				{Web: &WebChunk{URI: "https://a.com", Title: "A"}},
			},
			GroundingSupports: []GroundingSupport{
				{
					Segment:               &Segment{Text: "test"},
					GroundingChunkIndices: []int{0, 5}, // 5 is out of bounds
					ConfidenceScores:      []float64{0.9, 0.8},
				},
			},
		}
		citations := ExtractCitationsFromMetadata(gm)
		if len(citations) != 1 {
			t.Fatalf("expected 1 citation (skipping invalid index), got %d", len(citations))
		}
	})

	t.Run("nil web chunk", func(t *testing.T) {
		gm := &GroundingMetadata{
			GroundingChunks: []GroundingChunk{
				{}, // No Web field
			},
		}
		citations := ExtractCitationsFromMetadata(gm)
		if len(citations) != 0 {
			t.Fatalf("expected 0 citations for nil web chunks, got %d", len(citations))
		}
	})
}

func TestExtractCitationsFromResponse(t *testing.T) {
	resp := &GenerateContentResponse{
		Candidates: []Candidate{{
			Content: &CandidateContent{Role: "model", Parts: []Part{{Text: "answer"}}},
			GroundingMetadata: &GroundingMetadata{
				GroundingChunks: []GroundingChunk{
					{Web: &WebChunk{URI: "https://example.com", Title: "Example"}},
				},
			},
		}},
	}

	citations := ExtractCitationsFromResponse(resp)
	if len(citations) != 1 {
		t.Fatalf("expected 1 citation, got %d", len(citations))
	}
	if citations[0].URL != "https://example.com" {
		t.Errorf("URL = %q", citations[0].URL)
	}
}

func TestSearchQueriesFromMetadata(t *testing.T) {
	t.Run("nil metadata", func(t *testing.T) {
		queries := SearchQueriesFromMetadata(nil)
		if queries != nil {
			t.Error("expected nil for nil metadata")
		}
	})

	t.Run("with queries", func(t *testing.T) {
		gm := &GroundingMetadata{
			WebSearchQueries: []string{"population of Riyadh", "Riyadh demographics 2025"},
		}
		queries := SearchQueriesFromMetadata(gm)
		if len(queries) != 2 {
			t.Fatalf("expected 2 queries, got %d", len(queries))
		}
		if queries[0] != "population of Riyadh" {
			t.Errorf("query[0] = %q", queries[0])
		}
	})
}

func TestSearchGrounding_SearchPlusFunctionCalling(t *testing.T) {
	// Test combining search grounding with function calling on a request.
	req := &GenerateContentRequest{
		Contents: []Message{{Role: "user", Parts: []Part{{Text: "What is the weather in Riyadh and what is its population?"}}}},
	}

	// Add function declarations.
	decls := []FunctionDeclaration{
		{Name: "get_weather", Description: "Get weather for a city", Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"city": SchemaString("City name"),
			},
			"required": []string{"city"},
		}},
	}
	ApplyTools(req, decls, nil)

	// Add search grounding with VALIDATED mode for 3.x.
	threshold := 0.5
	EnableSearchWithFunctionCalling(req, &threshold)

	// Verify request structure.
	if len(req.Tools) != 2 {
		t.Fatalf("expected 2 tool entries, got %d", len(req.Tools))
	}

	// First tool: function declarations.
	if len(req.Tools[0].FunctionDeclarations) != 1 {
		t.Errorf("expected 1 function declaration in tools[0], got %d", len(req.Tools[0].FunctionDeclarations))
	}

	// Second tool: google_search with VALIDATED mode.
	if req.Tools[1].GoogleSearch == nil {
		t.Fatal("expected GoogleSearch in tools[1]")
	}
	if req.Tools[1].GoogleSearch.DynamicRetrievalConfig == nil {
		t.Fatal("expected DynamicRetrievalConfig")
	}
	if req.Tools[1].GoogleSearch.DynamicRetrievalConfig.Mode != "VALIDATED" {
		t.Errorf("mode = %q, want VALIDATED", req.Tools[1].GoogleSearch.DynamicRetrievalConfig.Mode)
	}
}

func TestSearchGrounding_JSONSerialization(t *testing.T) {
	req := &GenerateContentRequest{
		Contents: []Message{{Role: "user", Parts: []Part{{Text: "test"}}}},
	}
	EnableSearchGrounding(req)

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	// Verify the JSON contains googleSearch.
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	tools, ok := parsed["tools"].([]any)
	if !ok || len(tools) == 0 {
		t.Fatal("expected tools array")
	}

	tool0, ok := tools[0].(map[string]any)
	if !ok {
		t.Fatal("expected tool object")
	}

	if _, ok := tool0["googleSearch"]; !ok {
		t.Error("expected googleSearch key in tool JSON")
	}
}

func TestSearchGrounding_StreamingWithGrounding(t *testing.T) {
	// Simulate a streaming response with grounding metadata.
	chunk := `{"candidates":[{"content":{"role":"model","parts":[{"text":"Riyadh has about 7.7 million people."}]},"finishReason":"STOP","groundingMetadata":{"webSearchQueries":["population of Riyadh"],"groundingChunks":[{"web":{"uri":"https://example.com/riyadh","title":"Riyadh Pop"}}],"groundingSupports":[{"segment":{"startIndex":0,"endIndex":35,"text":"Riyadh has about 7.7 million people."},"groundingChunkIndices":[0],"confidenceScores":[0.95]}]}}],"usageMetadata":{"promptTokenCount":8,"candidatesTokenCount":12,"totalTokenCount":20}}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "data: %s\n\n", chunk)
	}))
	defer server.Close()

	client := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL), WithAPIVersion("v1beta"))
	events := client.StreamResponse(context.Background(), "test-model", &GenerateContentRequest{
		Contents: []Message{{Role: "user", Parts: []Part{{Text: "population of Riyadh"}}}},
	})

	var textEvents, groundingEvents, doneEvents int
	var citations []Citation
	for ev := range events {
		switch ev.Type {
		case EventText:
			textEvents++
		case EventGrounding:
			groundingEvents++
			citations = ev.Grounding
		case EventDone:
			doneEvents++
		}
	}

	if textEvents != 1 {
		t.Errorf("expected 1 text event, got %d", textEvents)
	}
	if groundingEvents != 1 {
		t.Errorf("expected 1 grounding event, got %d", groundingEvents)
	}
	if doneEvents != 1 {
		t.Errorf("expected 1 done event, got %d", doneEvents)
	}
	if len(citations) != 1 {
		t.Fatalf("expected 1 citation, got %d", len(citations))
	}
	if citations[0].URL != "https://example.com/riyadh" {
		t.Errorf("citation URL = %q", citations[0].URL)
	}
}
