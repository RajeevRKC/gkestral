//go:build integration

package gemini

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// skipIfNoKey skips the test if GEMINI_API_KEY is not set.
func skipIfNoKey(t *testing.T) string {
	t.Helper()
	key := os.Getenv("GEMINI_API_KEY")
	if key == "" {
		t.Skip("GEMINI_API_KEY not set -- skipping integration test")
	}
	return key
}

// integrationClient creates a Client configured for integration testing.
func integrationClient(t *testing.T) *Client {
	t.Helper()
	key := skipIfNoKey(t)
	return NewClient(
		WithAPIKey(key),
		WithRetryConfig(RetryConfig{
			MaxAttempts:  5,
			InitialDelay: 2 * time.Second,
			MaxDelay:     15 * time.Second,
		}),
	)
}

func TestIntegration_SimplePrompt(t *testing.T) {
	client := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	events := client.StreamResponse(ctx, "gemini-2.5-flash", &GenerateContentRequest{
		Contents: []Message{{
			Role:  "user",
			Parts: []Part{{Text: "Reply with exactly the word: PONG"}},
		}},
	})

	var text strings.Builder
	var gotDone bool
	var usage *UsageMetadata

	for ev := range events {
		switch ev.Type {
		case EventText:
			text.WriteString(ev.Text)
		case EventDone:
			gotDone = true
			usage = ev.Usage
		case EventError:
			t.Fatalf("stream error: %v", ev.Error)
		}
	}

	if !gotDone {
		t.Error("did not receive Done event")
	}
	if !strings.Contains(text.String(), "PONG") {
		t.Errorf("expected response containing PONG, got %q", text.String())
	}
	if usage != nil {
		t.Logf("Token usage: prompt=%d, candidates=%d, total=%d",
			usage.PromptTokenCount, usage.CandidatesTokenCount, usage.TotalTokenCount)
	}
}

func TestIntegration_ThinkingMode(t *testing.T) {
	client := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	req := &GenerateContentRequest{
		Contents: []Message{{
			Role:  "user",
			Parts: []Part{{Text: "What is 2^10? Think through it step by step."}},
		}},
		GenerationConfig: &GenerationConfig{
			ThinkingConfig: &ThinkingConfig{
				IncludeThoughts: true,
				ThinkingBudget:  1024,
			},
		},
	}

	events := client.StreamResponse(ctx, "gemini-2.5-flash", req)

	var thoughtCount, textCount int
	var text strings.Builder

	for ev := range events {
		switch ev.Type {
		case EventThoughtSignature:
			thoughtCount++
			if ev.Thought != nil {
				t.Logf("Thought: %s", truncate(ev.Thought.Text, 100))
			}
		case EventText:
			textCount++
			text.WriteString(ev.Text)
		case EventDone:
			if ev.Usage != nil {
				t.Logf("Tokens: prompt=%d, candidates=%d, thinking=%d, total=%d",
					ev.Usage.PromptTokenCount, ev.Usage.CandidatesTokenCount,
					ev.Usage.ThinkingTokenCount, ev.Usage.TotalTokenCount)
			}
		case EventError:
			t.Fatalf("stream error: %v", ev.Error)
		}
	}

	if textCount == 0 {
		t.Error("no text events received")
	}
	if !strings.Contains(text.String(), "1024") {
		t.Errorf("expected response containing 1024, got %q", text.String())
	}
	t.Logf("Thought events: %d, Text events: %d", thoughtCount, textCount)
}

func TestIntegration_FunctionCalling(t *testing.T) {
	client := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	weatherTool := DeclareTool("get_weather", "Get current weather for a city", SchemaObject(
		"Weather request",
		map[string]any{
			"city": SchemaString("City name"),
		},
		[]string{"city"},
	))

	req := &GenerateContentRequest{
		Contents: []Message{{
			Role:  "user",
			Parts: []Part{{Text: "What is the weather in Riyadh?"}},
		}},
	}
	ApplyTools(req, []FunctionDeclaration{weatherTool}, BuildToolConfig("AUTO"))

	events := client.StreamResponse(ctx, "gemini-2.5-flash", req)

	var gotToolCall bool
	var toolCallData *ToolCallData

	for ev := range events {
		switch ev.Type {
		case EventToolCall:
			gotToolCall = true
			toolCallData = ev.ToolCall
			t.Logf("Tool call: %s(%v)", ev.ToolCall.FunctionName, ev.ToolCall.Arguments)
		case EventError:
			t.Fatalf("stream error: %v", ev.Error)
		}
	}

	if !gotToolCall {
		t.Error("expected model to call get_weather tool")
	}
	if toolCallData != nil {
		if toolCallData.FunctionName != "get_weather" {
			t.Errorf("function name = %q, want get_weather", toolCallData.FunctionName)
		}
		city, ok := toolCallData.Arguments["city"]
		if !ok {
			t.Error("expected city argument")
		}
		cityStr, _ := city.(string)
		if !strings.Contains(strings.ToLower(cityStr), "riyadh") {
			t.Errorf("city = %q, expected to contain riyadh", cityStr)
		}
	}
}

func TestIntegration_ParallelCalls(t *testing.T) {
	client := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tools := []FunctionDeclaration{
		DeclareTool("get_weather", "Get current weather", SchemaObject(
			"Weather request",
			map[string]any{"city": SchemaString("City name")},
			[]string{"city"},
		)),
		DeclareTool("get_time", "Get current time in a timezone", SchemaObject(
			"Time request",
			map[string]any{"timezone": SchemaString("Timezone like Asia/Riyadh")},
			[]string{"timezone"},
		)),
	}

	req := &GenerateContentRequest{
		Contents: []Message{{
			Role:  "user",
			Parts: []Part{{Text: "What is the weather and current time in Riyadh?"}},
		}},
	}
	ApplyTools(req, tools, BuildToolConfig("AUTO"))

	events := client.StreamResponse(ctx, "gemini-2.5-flash", req)

	var toolCalls []ToolCallData
	for ev := range events {
		if ev.Type == EventToolCall {
			toolCalls = append(toolCalls, *ev.ToolCall)
			t.Logf("Tool call: %s(%v) id=%s", ev.ToolCall.FunctionName, ev.ToolCall.Arguments, ev.ToolCall.ID)
		}
		if ev.Type == EventError {
			t.Fatalf("stream error: %v", ev.Error)
		}
	}

	// Model may or may not issue parallel calls -- log what we get.
	t.Logf("Total tool calls: %d", len(toolCalls))
	if len(toolCalls) == 0 {
		t.Error("expected at least one tool call")
	}
}

func TestIntegration_SearchGrounding(t *testing.T) {
	client := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req := &GenerateContentRequest{
		Contents: []Message{{
			Role:  "user",
			Parts: []Part{{Text: "What is the current population of Riyadh, Saudi Arabia?"}},
		}},
	}
	EnableSearchGrounding(req)

	events := client.StreamResponse(ctx, "gemini-2.5-flash", req)

	var text strings.Builder
	var gotGrounding bool
	var citations []Citation

	for ev := range events {
		switch ev.Type {
		case EventText:
			text.WriteString(ev.Text)
		case EventGrounding:
			gotGrounding = true
			citations = ev.Grounding
			t.Logf("Grounding citations: %d", len(citations))
			for i, c := range citations {
				t.Logf("  [%d] %s -- %s", i, c.Title, c.URL)
			}
		case EventError:
			t.Fatalf("stream error: %v", ev.Error)
		}
	}

	if text.Len() == 0 {
		t.Error("no text response received")
	}
	t.Logf("Response: %s", truncate(text.String(), 200))

	if gotGrounding {
		t.Logf("Search grounding returned %d citations", len(citations))
	} else {
		t.Log("WARNING: No grounding metadata in response (model may have answered without search)")
	}
}

func TestIntegration_ContextCaching(t *testing.T) {
	client := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cm := NewCacheManager(client)

	// Build stable content exceeding minimum token threshold.
	// 4096 min for Pro, 1024 min for Flash. We target Flash.
	var sb strings.Builder
	for i := 0; i < 500; i++ {
		sb.WriteString(fmt.Sprintf("Technical document section %d: This section covers the implementation details of context caching in the Gemini API. ", i))
	}

	cacheReq := &CachedContentRequest{
		Model: "models/gemini-2.5-flash",
		Contents: []Message{{
			Role:  "user",
			Parts: []Part{{Text: sb.String()}},
		}},
		TTL:         "300s",
		DisplayName: "integration-test-cache",
	}

	entry, err := cm.Create(ctx, cacheReq)
	if err != nil {
		t.Fatalf("cache create failed: %v", err)
	}
	t.Logf("Cache created: %s (expires: %s)", entry.Name, entry.ExpireTime)

	// Clean up the cache after the test.
	defer func() {
		_ = cm.Delete(context.Background(), entry.Name)
	}()

	// Use the cache in a generate request.
	req := &GenerateContentRequest{
		Contents: []Message{{
			Role:  "user",
			Parts: []Part{{Text: "Summarize the key points of the technical document."}},
		}},
	}
	UseCachedContent(req, entry.Name)

	resp, err := client.GenerateContent(ctx, "gemini-2.5-flash", req)
	if err != nil {
		t.Fatalf("generate with cache failed: %v", err)
	}

	if resp.UsageMetadata != nil {
		t.Logf("Tokens: prompt=%d, cached=%d, candidates=%d, total=%d",
			resp.UsageMetadata.PromptTokenCount,
			resp.UsageMetadata.CachedContentTokenCount,
			resp.UsageMetadata.CandidatesTokenCount,
			resp.UsageMetadata.TotalTokenCount)

		if resp.UsageMetadata.CachedContentTokenCount > 0 {
			t.Logf("Cache HIT: %d tokens served from cache", resp.UsageMetadata.CachedContentTokenCount)
		}
	}

	// Verify response has content.
	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		t.Error("empty response from cached request")
	}
}

func TestIntegration_StructuredOutput(t *testing.T) {
	client := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	type CityInfo struct {
		Name       string `json:"name"`
		Country    string `json:"country"`
		Population int    `json:"population"`
	}

	schema, err := SchemaFromStruct(CityInfo{})
	if err != nil {
		t.Fatalf("schema generation failed: %v", err)
	}

	req := &GenerateContentRequest{
		Contents: []Message{{
			Role:  "user",
			Parts: []Part{{Text: "Give me basic info about Tokyo in the requested JSON format."}},
		}},
	}
	EnableStructuredOutput(req, schema)

	resp, err := client.GenerateContent(ctx, "gemini-2.5-flash", req)
	if err != nil {
		t.Fatalf("structured output request failed: %v", err)
	}

	var city CityInfo
	if err := ParseStructuredResponse(resp, &city); err != nil {
		t.Fatalf("parse structured response failed: %v", err)
	}

	t.Logf("Structured response: %+v", city)
	if city.Name == "" {
		t.Error("expected non-empty city name")
	}
	if city.Country == "" {
		t.Error("expected non-empty country")
	}
}

// ---------- Token Economics Benchmark ----------

func TestIntegration_TokenEconomicsBenchmark(t *testing.T) {
	key := skipIfNoKey(t)
	client := NewClient(
		WithAPIKey(key),
		WithRetryConfig(RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 2 * time.Second,
			MaxDelay:     10 * time.Second,
		}),
	)

	// Standard prompt for comparing models.
	prompt := "Explain the concept of context caching in large language model APIs. " +
		"Cover the economic benefits, minimum token thresholds, and TTL management. " +
		"Be concise but thorough."

	models := []string{
		"gemini-2.5-flash",
		// Only test flash by default to avoid cost. Uncomment for full benchmark:
		// "gemini-2.5-pro",
		// "gemini-3.1-flash",
		// "gemini-3.1-pro-preview",
	}

	t.Log("=== Token Economics Benchmark ===")
	t.Logf("%-30s | %8s | %8s | %10s | %10s | %8s",
		"Model", "Input", "Output", "Cost", "Cached$", "Savings")
	t.Log(strings.Repeat("-", 90))

	for _, modelID := range models {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)

		resp, err := client.GenerateContent(ctx, modelID, &GenerateContentRequest{
			Contents: []Message{{
				Role:  "user",
				Parts: []Part{{Text: prompt}},
			}},
		})
		cancel()

		if err != nil {
			t.Logf("%-30s | ERROR: %v", modelID, err)
			continue
		}

		if resp.UsageMetadata == nil {
			t.Logf("%-30s | no usage metadata", modelID)
			continue
		}

		u := resp.UsageMetadata
		noCacheCost, _ := TokenEconomics(modelID, u.PromptTokenCount, u.CandidatesTokenCount, 0)
		cachedCost, _ := TokenEconomics(modelID, u.PromptTokenCount, u.CandidatesTokenCount, u.PromptTokenCount)

		t.Logf("%-30s | %8d | %8d | $%9.6f | $%9.6f | %7.1f%%",
			modelID,
			u.PromptTokenCount,
			u.CandidatesTokenCount,
			noCacheCost.TotalCost,
			cachedCost.TotalCost,
			(noCacheCost.TotalCost-cachedCost.TotalCost)/noCacheCost.TotalCost*100)
	}
}

// truncate shortens a string to maxLen characters.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
