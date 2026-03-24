// Package gemini provides a Go client library for the Google Gemini REST API.
//
// It implements raw REST + SSE streaming without external SDK dependencies,
// targeting Gemini 2.5 and 3.x model families. Key capabilities:
//
//   - Streaming via Server-Sent Events (SSE) with backpressure-aware channels
//   - Thought signature extraction and circulation for Gemini 3.x models
//   - Context caching via the CachedContents REST resource (90% cost discount)
//   - Parallel function calling with ID-based result matching
//   - Google Search grounding with structured citation extraction
//   - Model routing with task classification heuristics
//   - Structured output via JSON Schema generation from Go structs
//   - Exponential backoff retry with persistent 429 callbacks
//   - System prompt engineering with model-aware temperature enforcement
//
// Quick start:
//
//	client := gemini.NewClient(
//	    gemini.WithAPIKey(os.Getenv("GEMINI_API_KEY")),
//	    gemini.WithModel("gemini-2.5-flash"),
//	)
//
//	request := &gemini.GenerateContentRequest{
//	    Contents: []gemini.Message{
//	        {Role: "user", Parts: []gemini.Part{{Text: "Hello"}}},
//	    },
//	}
//
//	// Streaming
//	for event := range client.StreamResponse(ctx, "", request) {
//	    switch event.Type {
//	    case gemini.EventText:
//	        fmt.Print(event.Text)
//	    case gemini.EventDone:
//	        fmt.Println()
//	    }
//	}
package gemini
