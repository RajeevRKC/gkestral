package gemini

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

const (
	// sseDataPrefix is the SSE line prefix for data events.
	sseDataPrefix = "data: "
	// streamEventBufferSize is the channel buffer for streaming events.
	// Provides backpressure: if consumer is slow, producer blocks after 100 events.
	streamEventBufferSize = 100
)

// StreamResponse sends a streaming request to the Gemini API and returns a
// channel of ProviderEvents. The channel is closed when the stream ends
// (either successfully with an EventDone or on error with an EventError).
//
// Consumers MUST read from the channel until it is closed to avoid goroutine leaks.
// Cancel the context to abort the stream early.
func (c *Client) StreamResponse(ctx context.Context, model string, request *GenerateContentRequest) <-chan ProviderEvent {
	events := make(chan ProviderEvent, streamEventBufferSize)

	go func() {
		defer close(events)
		c.runStream(ctx, model, request, events)
	}()

	return events
}

// runStream performs the HTTP request and parses the SSE response, emitting
// events to the channel. This runs in a dedicated goroutine.
func (c *Client) runStream(ctx context.Context, model string, request *GenerateContentRequest, events chan<- ProviderEvent) {
	req, err := c.buildStreamRequest(ctx, model, request)
	if err != nil {
		sendEvent(ctx, events, ProviderEvent{Type: EventError, Error: fmt.Errorf("build request: %w", err)})
		return
	}

	// Use the raw HTTP client (no retry wrapper) for streaming.
	resp, err := c.httpClient.Do(req)
	if err != nil {
		sendEvent(ctx, events, ProviderEvent{Type: EventError, Error: fmt.Errorf("stream request: %w", err)})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		sendEvent(ctx, events, ProviderEvent{Type: EventError, Error: &APIError{StatusCode: resp.StatusCode, Message: string(body)}})
		return
	}

	parseSSEStream(ctx, resp.Body, events)
}

// parseSSEStream reads SSE lines from the response body and emits ProviderEvents.
// Each SSE "data: " line contains a JSON GenerateContentResponse chunk.
func parseSSEStream(ctx context.Context, body io.Reader, events chan<- ProviderEvent) {
	scanner := bufio.NewScanner(body)
	// Increase buffer for large chunks (Gemini can send large JSON in a single SSE line).
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var lastUsage *UsageMetadata

	for scanner.Scan() {
		// Check context cancellation between lines.
		select {
		case <-ctx.Done():
			events <- ProviderEvent{Type: EventError, Error: ctx.Err()}
			return
		default:
		}

		line := scanner.Text()

		// Skip empty lines and comment lines (SSE spec).
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		// Only process data lines.
		if !strings.HasPrefix(line, sseDataPrefix) {
			continue
		}

		data := strings.TrimPrefix(line, sseDataPrefix)
		if data == "" {
			continue
		}

		var chunk GenerateContentResponse
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			sendEvent(ctx, events, ProviderEvent{Type: EventError, Error: fmt.Errorf("parse SSE chunk: %w", err)})
			return
		}

		// Track usage metadata (only reliably present on the final chunk).
		if chunk.UsageMetadata != nil {
			lastUsage = chunk.UsageMetadata
		}

		// Check for prompt-level blocking.
		if chunk.PromptFeedback != nil && chunk.PromptFeedback.BlockReason != "" {
			sendEvent(ctx, events, ProviderEvent{Type: EventError, Error: fmt.Errorf("prompt blocked: %s", chunk.PromptFeedback.BlockReason)})
			return
		}

		// Process candidates.
		if len(chunk.Candidates) == 0 {
			continue
		}

		candidate := chunk.Candidates[0]

		// Check for safety-blocked candidate.
		if candidate.FinishReason == "SAFETY" {
			reasons := extractBlockReasons(candidate.SafetyRatings)
			sendEvent(ctx, events, ProviderEvent{Type: EventError, Error: fmt.Errorf("response blocked by safety filter: %s", strings.Join(reasons, ", "))})
			return
		}

		// Emit events for each part in the candidate content.
		if candidate.Content != nil {
			for _, part := range candidate.Content.Parts {
				event := partToEvent(part)
				if event != nil {
					if !sendEvent(ctx, events, *event) {
						return
					}
				}
			}
		}

		// Emit grounding metadata if present.
		if candidate.GroundingMetadata != nil {
			citations := extractCitations(candidate.GroundingMetadata)
			if len(citations) > 0 {
				if !sendEvent(ctx, events, ProviderEvent{Type: EventGrounding, Grounding: citations}) {
					return
				}
			}
		}

		// Check for terminal finish reasons.
		switch candidate.FinishReason {
		case "STOP":
			sendEvent(ctx, events, ProviderEvent{Type: EventDone, Usage: lastUsage})
			return
		case "MAX_TOKENS", "RECITATION", "OTHER", "BLOCKLIST", "PROHIBITED_CONTENT", "SPII":
			sendEvent(ctx, events, ProviderEvent{Type: EventDone, Usage: lastUsage, Text: candidate.FinishReason})
			return
		}
	}

	if err := scanner.Err(); err != nil {
		sendEvent(ctx, events, ProviderEvent{Type: EventError, Error: fmt.Errorf("read SSE stream: %w", err)})
		return
	}

	// Stream ended without a STOP finish reason -- emit Done with whatever
	// usage we accumulated.
	sendEvent(ctx, events, ProviderEvent{Type: EventDone, Usage: lastUsage})
}

// partToEvent converts a Gemini response Part to a ProviderEvent.
// Returns nil if the part doesn't map to a meaningful event.
func partToEvent(part Part) *ProviderEvent {
	// Thought parts (Gemini 3.x): has thought=true flag OR thoughtSignature.
	// Empty-text parts carrying only a signature are valid and must be preserved.
	if part.Thought || part.ThoughtSignature != "" {
		return &ProviderEvent{
			Type: EventThoughtSignature,
			Thought: &ThoughtPart{
				Text:             part.Text,
				Thought:          part.Thought,
				ThoughtSignature: part.ThoughtSignature,
			},
		}
	}

	// Function call (may also carry a thoughtSignature on Gemini 3.x).
	if part.FunctionCall != nil {
		return &ProviderEvent{
			Type: EventToolCall,
			ToolCall: &ToolCallData{
				ID:           part.FunctionCall.ID,
				FunctionName: part.FunctionCall.Name,
				Arguments:    part.FunctionCall.Args,
			},
		}
	}

	// Text content.
	if part.Text != "" {
		return &ProviderEvent{
			Type: EventText,
			Text: part.Text,
		}
	}

	return nil
}

// extractBlockReasons returns the categories that triggered safety blocking.
func extractBlockReasons(ratings []SafetyRating) []string {
	var reasons []string
	for _, r := range ratings {
		if r.Blocked {
			reasons = append(reasons, r.Category)
		}
	}
	return reasons
}

// extractCitations converts GroundingMetadata into structured Citation objects.
func extractCitations(gm *GroundingMetadata) []Citation {
	if gm == nil || len(gm.GroundingChunks) == 0 {
		return nil
	}

	var citations []Citation
	for _, chunk := range gm.GroundingChunks {
		if chunk.Web != nil {
			citations = append(citations, Citation{
				URL:   chunk.Web.URI,
				Title: chunk.Web.Title,
			})
		}
	}
	return citations
}

// sendEvent sends an event to the channel with context cancellation guard.
// Returns false if the context was cancelled (caller should return).
func sendEvent(ctx context.Context, events chan<- ProviderEvent, event ProviderEvent) bool {
	select {
	case events <- event:
		return true
	case <-ctx.Done():
		// Best-effort send of cancellation error.
		select {
		case events <- ProviderEvent{Type: EventError, Error: ctx.Err()}:
		default:
		}
		return false
	}
}

// APIError is defined in retry.go.
