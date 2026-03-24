package gemini

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// sseResponse builds an SSE-formatted response string from JSON chunks.
func sseResponse(chunks ...string) string {
	var sb strings.Builder
	for _, chunk := range chunks {
		sb.WriteString("data: ")
		sb.WriteString(chunk)
		sb.WriteString("\n\n")
	}
	return sb.String()
}

func TestStreamResponse_TextContent(t *testing.T) {
	chunk1 := `{"candidates":[{"content":{"role":"model","parts":[{"text":"Hello"}]}}]}`
	chunk2 := `{"candidates":[{"content":{"role":"model","parts":[{"text":" world"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":5,"totalTokenCount":15}}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, sseResponse(chunk1, chunk2))
	}))
	defer server.Close()

	client := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL), WithAPIVersion("v1beta"))
	events := client.StreamResponse(context.Background(), "test-model", &GenerateContentRequest{
		Contents: []Message{{Role: "user", Parts: []Part{{Text: "Hi"}}}},
	})

	var collected []ProviderEvent
	for ev := range events {
		collected = append(collected, ev)
	}

	if len(collected) < 3 {
		t.Fatalf("expected at least 3 events, got %d", len(collected))
	}

	// First event: text "Hello"
	if collected[0].Type != EventText || collected[0].Text != "Hello" {
		t.Errorf("event 0: want Text='Hello', got type=%s text=%q", collected[0].Type, collected[0].Text)
	}

	// Second event: text " world"
	if collected[1].Type != EventText || collected[1].Text != " world" {
		t.Errorf("event 1: want Text=' world', got type=%s text=%q", collected[1].Type, collected[1].Text)
	}

	// Final event: Done with usage
	last := collected[len(collected)-1]
	if last.Type != EventDone {
		t.Errorf("last event: want Done, got %s", last.Type)
	}
	if last.Usage == nil {
		t.Fatal("last event: usage is nil")
	}
	if last.Usage.TotalTokenCount != 15 {
		t.Errorf("usage TotalTokenCount: want 15, got %d", last.Usage.TotalTokenCount)
	}
}

func TestStreamResponse_ThoughtParts(t *testing.T) {
	chunk := `{"candidates":[{"content":{"role":"model","parts":[{"text":"Let me think...","thought":true},{"text":"The answer is 42"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":5,"candidatesTokenCount":10,"totalTokenCount":15}}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, sseResponse(chunk))
	}))
	defer server.Close()

	client := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL), WithAPIVersion("v1beta"))
	events := client.StreamResponse(context.Background(), "test-model", &GenerateContentRequest{
		Contents: []Message{{Role: "user", Parts: []Part{{Text: "What is 6*7?"}}}},
	})

	var thoughts, texts int
	for ev := range events {
		switch ev.Type {
		case EventThoughtSignature:
			thoughts++
			if ev.Thought == nil || !ev.Thought.Thought {
				t.Error("thought event missing Thought data")
			}
			if ev.Thought.Text != "Let me think..." {
				t.Errorf("thought text: want 'Let me think...', got %q", ev.Thought.Text)
			}
		case EventText:
			texts++
			if ev.Text != "The answer is 42" {
				t.Errorf("text: want 'The answer is 42', got %q", ev.Text)
			}
		}
	}

	if thoughts != 1 {
		t.Errorf("want 1 thought event, got %d", thoughts)
	}
	if texts != 1 {
		t.Errorf("want 1 text event, got %d", texts)
	}
}

func TestStreamResponse_FunctionCall(t *testing.T) {
	chunk := `{"candidates":[{"content":{"role":"model","parts":[{"functionCall":{"name":"get_weather","args":{"city":"London"},"id":"call_1"}}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":5,"candidatesTokenCount":3,"totalTokenCount":8}}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, sseResponse(chunk))
	}))
	defer server.Close()

	client := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL), WithAPIVersion("v1beta"))
	events := client.StreamResponse(context.Background(), "test-model", &GenerateContentRequest{
		Contents: []Message{{Role: "user", Parts: []Part{{Text: "Weather?"}}}},
	})

	var toolCalls int
	for ev := range events {
		if ev.Type == EventToolCall {
			toolCalls++
			if ev.ToolCall == nil {
				t.Fatal("tool call event has nil ToolCall")
			}
			if ev.ToolCall.FunctionName != "get_weather" {
				t.Errorf("function name: want 'get_weather', got %q", ev.ToolCall.FunctionName)
			}
			if ev.ToolCall.ID != "call_1" {
				t.Errorf("call ID: want 'call_1', got %q", ev.ToolCall.ID)
			}
			city, ok := ev.ToolCall.Arguments["city"]
			if !ok || city != "London" {
				t.Errorf("args city: want 'London', got %v", city)
			}
		}
	}

	if toolCalls != 1 {
		t.Errorf("want 1 tool call event, got %d", toolCalls)
	}
}

func TestStreamResponse_ContextCancellation(t *testing.T) {
	// Server sends data slowly -- context should cancel before stream ends.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("server does not support flushing")
		}
		for i := 0; i < 100; i++ {
			chunk := fmt.Sprintf(`{"candidates":[{"content":{"role":"model","parts":[{"text":"chunk %d "}]}}]}`, i)
			fmt.Fprintf(w, "data: %s\n\n", chunk)
			flusher.Flush()
			time.Sleep(50 * time.Millisecond)
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	client := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL), WithAPIVersion("v1beta"))
	events := client.StreamResponse(ctx, "test-model", &GenerateContentRequest{
		Contents: []Message{{Role: "user", Parts: []Part{{Text: "slow"}}}},
	})

	var count int
	var gotError bool
	for ev := range events {
		count++
		if ev.Type == EventError {
			gotError = true
		}
	}

	if !gotError {
		t.Error("expected an error event from context cancellation")
	}
	if count >= 100 {
		t.Errorf("expected early termination, got %d events", count)
	}
}

func TestStreamResponse_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprint(w, `{"error":{"message":"quota exceeded"}}`)
	}))
	defer server.Close()

	client := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL), WithAPIVersion("v1beta"))
	events := client.StreamResponse(context.Background(), "test-model", &GenerateContentRequest{
		Contents: []Message{{Role: "user", Parts: []Part{{Text: "Hi"}}}},
	})

	var errorEvents []ProviderEvent
	for ev := range events {
		if ev.Type == EventError {
			errorEvents = append(errorEvents, ev)
		}
	}

	if len(errorEvents) == 0 {
		t.Fatal("expected error event for 429 response")
	}
	apiErr, ok := errorEvents[0].Error.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", errorEvents[0].Error)
	}
	if apiErr.StatusCode != 429 {
		t.Errorf("status code: want 429, got %d", apiErr.StatusCode)
	}
}

func TestStreamResponse_SafetyBlocked(t *testing.T) {
	chunk := `{"candidates":[{"finishReason":"SAFETY","safetyRatings":[{"category":"HARM_CATEGORY_DANGEROUS_CONTENT","probability":"HIGH","blocked":true}]}]}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, sseResponse(chunk))
	}))
	defer server.Close()

	client := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL), WithAPIVersion("v1beta"))
	events := client.StreamResponse(context.Background(), "test-model", &GenerateContentRequest{
		Contents: []Message{{Role: "user", Parts: []Part{{Text: "bad"}}}},
	})

	var gotSafetyError bool
	for ev := range events {
		if ev.Type == EventError && strings.Contains(ev.Error.Error(), "safety filter") {
			gotSafetyError = true
		}
	}

	if !gotSafetyError {
		t.Error("expected safety filter error event")
	}
}

func TestStreamResponse_PromptBlocked(t *testing.T) {
	chunk := `{"promptFeedback":{"blockReason":"SAFETY","safetyRatings":[{"category":"HARM_CATEGORY_HARASSMENT","probability":"HIGH","blocked":true}]}}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, sseResponse(chunk))
	}))
	defer server.Close()

	client := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL), WithAPIVersion("v1beta"))
	events := client.StreamResponse(context.Background(), "test-model", &GenerateContentRequest{
		Contents: []Message{{Role: "user", Parts: []Part{{Text: "bad prompt"}}}},
	})

	var gotBlockError bool
	for ev := range events {
		if ev.Type == EventError && strings.Contains(ev.Error.Error(), "prompt blocked") {
			gotBlockError = true
		}
	}

	if !gotBlockError {
		t.Error("expected prompt blocked error event")
	}
}

func TestStreamResponse_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {invalid json}\n\n")
	}))
	defer server.Close()

	client := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL), WithAPIVersion("v1beta"))
	events := client.StreamResponse(context.Background(), "test-model", &GenerateContentRequest{
		Contents: []Message{{Role: "user", Parts: []Part{{Text: "Hi"}}}},
	})

	var gotParseError bool
	for ev := range events {
		if ev.Type == EventError && strings.Contains(ev.Error.Error(), "parse SSE chunk") {
			gotParseError = true
		}
	}

	if !gotParseError {
		t.Error("expected JSON parse error event")
	}
}

func TestStreamResponse_GroundingMetadata(t *testing.T) {
	chunk := `{"candidates":[{"content":{"role":"model","parts":[{"text":"Grounded answer"}]},"finishReason":"STOP","groundingMetadata":{"groundingChunks":[{"web":{"uri":"https://example.com","title":"Example"}}],"webSearchQueries":["test query"]}}],"usageMetadata":{"promptTokenCount":5,"candidatesTokenCount":3,"totalTokenCount":8}}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, sseResponse(chunk))
	}))
	defer server.Close()

	client := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL), WithAPIVersion("v1beta"))
	events := client.StreamResponse(context.Background(), "test-model", &GenerateContentRequest{
		Contents: []Message{{Role: "user", Parts: []Part{{Text: "search"}}}},
	})

	var gotGrounding bool
	for ev := range events {
		if ev.Type == EventGrounding {
			gotGrounding = true
			if len(ev.Grounding) == 0 {
				t.Error("grounding event has no citations")
			} else if ev.Grounding[0].URL != "https://example.com" {
				t.Errorf("citation URL: want 'https://example.com', got %q", ev.Grounding[0].URL)
			}
		}
	}

	if !gotGrounding {
		t.Error("expected grounding event")
	}
}

func TestStreamResponse_EmptyStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		// Empty response body.
	}))
	defer server.Close()

	client := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL), WithAPIVersion("v1beta"))
	events := client.StreamResponse(context.Background(), "test-model", &GenerateContentRequest{
		Contents: []Message{{Role: "user", Parts: []Part{{Text: "Hi"}}}},
	})

	var count int
	var lastType EventType
	for ev := range events {
		count++
		lastType = ev.Type
	}

	// Should get at least a Done event.
	if count == 0 {
		t.Fatal("expected at least one event")
	}
	if lastType != EventDone {
		t.Errorf("last event type: want Done, got %s", lastType)
	}
}

func TestPartToEvent_AllTypes(t *testing.T) {
	tests := []struct {
		name     string
		part     Part
		wantType EventType
		wantNil  bool
	}{
		{"text", Part{Text: "hello"}, EventText, false},
		{"thought", Part{Text: "thinking...", Thought: true}, EventThoughtSignature, false},
		{"signature_only", Part{ThoughtSignature: "abc123"}, EventThoughtSignature, false},
		{"thought_with_sig", Part{Text: "t", Thought: true, ThoughtSignature: "sig"}, EventThoughtSignature, false},
		{"function_call", Part{FunctionCall: &FunctionCall{Name: "fn", Args: map[string]any{"a": "b"}}}, EventToolCall, false},
		{"empty", Part{}, EventText, true},
		{"inline_data_only", Part{InlineData: &InlineData{MIMEType: "image/png", Data: "abc"}}, EventText, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev := partToEvent(tt.part)
			if tt.wantNil {
				if ev != nil {
					t.Errorf("expected nil event, got type=%s", ev.Type)
				}
				return
			}
			if ev == nil {
				t.Fatal("expected non-nil event")
			}
			if ev.Type != tt.wantType {
				t.Errorf("type: want %s, got %s", tt.wantType, ev.Type)
			}
		})
	}
}

func TestExtractCitations(t *testing.T) {
	gm := &GroundingMetadata{
		GroundingChunks: []GroundingChunk{
			{Web: &WebChunk{URI: "https://a.com", Title: "A"}},
			{Web: &WebChunk{URI: "https://b.com", Title: "B"}},
			{}, // chunk without web
		},
	}

	citations := extractCitations(gm)
	if len(citations) != 2 {
		t.Fatalf("want 2 citations, got %d", len(citations))
	}
	if citations[0].URL != "https://a.com" {
		t.Errorf("citation 0 URL: want 'https://a.com', got %q", citations[0].URL)
	}
	if citations[1].Title != "B" {
		t.Errorf("citation 1 Title: want 'B', got %q", citations[1].Title)
	}
}

func TestExtractCitations_Nil(t *testing.T) {
	if c := extractCitations(nil); c != nil {
		t.Errorf("expected nil for nil input, got %v", c)
	}
	if c := extractCitations(&GroundingMetadata{}); c != nil {
		t.Errorf("expected nil for empty metadata, got %v", c)
	}
}
