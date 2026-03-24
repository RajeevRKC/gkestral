# Gemini API Mastery Reference

> Gkestral Phase 01 Deliverable -- Go Client Library Deep Reference
> Covers all 13 Gemini API areas with implementation details, code examples,
> pricing tables, and gotchas discovered during development.
>
> Last updated: 2026-03-24 | Library: `internal/gemini/`

---

## 1. Model Landscape

Gkestral targets five Gemini models spanning two generations. The model registry
in `models.go` holds the full specification for each.

### Target Models

| Model ID | Family | Context | Max Output | Thinking | Grounding | Caching | Min Cache |
|----------|--------|---------|------------|----------|-----------|---------|-----------|
| `gemini-3.1-pro-preview` | Pro 3.1 | 1M | 65K | Yes (MEDIUM default) | Yes | Yes | 4096 |
| `gemini-3.1-flash` | Flash 3.1 | 1M | 8K | Yes (LOW default) | Yes | Yes | 1024 |
| `gemini-3.1-flash-image-preview` | Flash 3.1 Image | 1M | 8K | No | No | No | -- |
| `gemini-2.5-pro` | Pro 2.5 | 1M | 65K | Yes (OFF default) | Yes | Yes | 4096 |
| `gemini-2.5-flash` | Flash 2.5 | 1M | 8K | Yes (OFF default) | Yes | Yes | 1024 |

### Pricing (per 1M tokens, March 2026)

| Model | Input | Output | Caching Discount |
|-------|-------|--------|-----------------|
| 3.1 Pro | $1.25 | $10.00 | 75% |
| 3.1 Flash | $0.15 | $0.60 | 75% |
| 2.5 Pro | $1.25 | $10.00 | 75% |
| 2.5 Flash | $0.15 | $0.60 | 75% |

### Model Selection

```go
model, err := gemini.GetModel("gemini-2.5-flash")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Context: %d, Output: %d, Caching: %v\n",
    model.ContextWindow, model.MaxOutput, model.SupportsCaching)
```

### Gotcha: Model ID is the Registry Key

The `GetModel` function requires the exact model ID string. When using cache
resources, the API prepends `models/` -- use `stripModelPrefix` internally
to normalise.

---

## 2. Streaming

Gemini streams via Server-Sent Events (SSE) on the `streamGenerateContent`
endpoint with `?alt=sse`. Each SSE `data:` line contains a complete
`GenerateContentResponse` JSON chunk.

### Wire Format

```
data: {"candidates":[{"content":{"role":"model","parts":[{"text":"Hello"}]}}]}

data: {"candidates":[{"content":{"role":"model","parts":[{"text":" world"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":5,"totalTokenCount":15}}
```

### Channel-Based Consumer

```go
events := client.StreamResponse(ctx, "gemini-2.5-flash", request)
for ev := range events {
    switch ev.Type {
    case gemini.EventText:
        fmt.Print(ev.Text)
    case gemini.EventThoughtSignature:
        // Store for circulation
    case gemini.EventToolCall:
        // Dispatch to executor
    case gemini.EventGrounding:
        // Display citations
    case gemini.EventDone:
        fmt.Printf("\nTokens: %+v\n", ev.Usage)
    case gemini.EventError:
        log.Printf("Error: %v", ev.Error)
    }
}
```

### Implementation Details

- **Buffer size:** 100 events (backpressure if consumer is slow)
- **Scanner buffer:** 64KB initial, 1MB max (Gemini can send large JSON chunks)
- **Goroutine:** Stream runs in a dedicated goroutine; channel is closed on completion
- **Context cancellation:** Checked between SSE lines; cancellation error sent via channel
- **MUST drain:** Consumer MUST read until channel close to avoid goroutine leaks

### Gotcha: UsageMetadata on Final Chunk Only

Usage data appears only on the last SSE chunk (the one with a `finishReason`).
The parser tracks `lastUsage` and attaches it to the `EventDone` event.

### Gotcha: Partial Chunks

Gemini sends complete JSON per SSE line (no partial chunks across lines).
The `bufio.Scanner` handles line splitting. However, large code blocks in
responses can produce very long SSE lines -- hence the 1MB scanner limit.

---

## 3. Thinking (Thought Signatures)

Gemini 3.x models support native reasoning ("thinking") where the model
shows its reasoning process before generating the final answer.

### Enabling Thinking

```go
config := gemini.NewThinkingConfig(gemini.ThinkingLevelMedium, 1024, true)
gemini.ApplyThinkingConfig(request, "gemini-3.1-pro-preview", config)
```

### Thought Parts in Responses

Thought parts appear as regular `Part` objects with `thought: true` and an
opaque `thoughtSignature` string. These signatures MUST be circulated back
in subsequent turns exactly as received.

```go
// After receiving a model response with thought parts:
thoughts := gemini.ExtractThoughtParts(response)
// When building the next turn:
messages = gemini.CirculateThoughts(messages, originalModelParts)
```

### ThinkingConfig Parameters

| Parameter | 2.5 Models | 3.x Models |
|-----------|-----------|-----------|
| `thinkingBudget` | Token count (int), -1=dynamic | Not used |
| `thinkingLevel` | Not used | LOW, MEDIUM, HIGH, MINIMAL |
| `includeThoughts` | Must be true for thought text | Must be true for thought text |

### Gotcha: Temperature Must Be 1.0 for 3.x

Gemini 3.x models loop when temperature is set below 1.0.
`EnforceTemperature` in `system.go` forces 1.0 for all 3.x family models.

### Gotcha: Signature-Only Parts Are Valid

Some thought parts have empty text but carry a `thoughtSignature`. These
are valid and must be preserved -- they represent checkpoint signatures
in the model's reasoning chain.

### Gotcha: Never Modify Thought Parts

The `CirculateThoughts` function preserves the original model parts exactly.
Never concatenate, merge, or reconstruct thought parts -- the API validates
the opaque signature and will reject tampered parts.

---

## 4. Context Caching

Context caching is Gkestral's economic moat. It provides a 75% discount on
input tokens for stable content that doesn't change between requests.

### How It Works

1. **Create** a cache: upload stable content (system prompt, reference docs, tools)
2. **Receive** a cache name URI (`cachedContents/abc123`)
3. **Reference** the cache name in subsequent `generateContent` requests
4. Cached tokens are billed at 25% of normal input price

### CacheManager API

```go
cm := gemini.NewCacheManager(client)

// Create a cache
entry, err := cm.Create(ctx, &gemini.CachedContentRequest{
    Model:   "models/gemini-2.5-flash",
    Contents: stableMessages,
    SystemInstruction: systemPrompt,
    TTL:     "3600s",
    DisplayName: "project-context",
})

// Use in a request
gemini.UseCachedContent(request, entry.Name)

// Manage lifecycle
cm.Update(ctx, entry.Name, 2*time.Hour) // Extend TTL
cm.Delete(ctx, entry.Name)               // Clean up
```

### Stable/Active Split Strategy

```go
stable, active := gemini.SplitContext(allMessages, 4)
// stable -> cache with long TTL
// active -> send fresh each turn (last 4 messages)
```

### Minimum Token Thresholds

| Model | Minimum Tokens |
|-------|---------------|
| Flash (2.5, 3.1) | 1,024 |
| Pro (2.5, 3.1) | 4,096 |

Content below the threshold will be rejected. The library estimates token
count heuristically (4 chars per token) and returns an error before making
the API call.

### Break-Even Analysis

```go
comparison, err := gemini.CacheEconomics("gemini-2.5-flash", 50000, 45000, 1000)
// comparison.SavingsPerReq = cost saved per request
// comparison.BreakEvenReqs = requests needed to recover cache creation cost
```

### Gotcha: Cache Is a Stateful REST Resource

Context caching is NOT a request parameter. It's a separate REST resource at
`/v1beta/cachedContents`. You must POST to create it, and it has a lifecycle
(TTL, update, delete). This is different from how most LLM APIs handle caching.

### Gotcha: Model Prefix

The cache creation API requires `models/gemini-2.5-flash` (with prefix),
but `GetModel` uses just `gemini-2.5-flash`. The `stripModelPrefix` utility
handles this normalisation.

---

## 5. Function Calling

Gemini supports declaring tools that the model can invoke during generation.
The model returns `FunctionCall` parts that the application executes and
returns results for.

### Declaring Tools

```go
weatherTool := gemini.DeclareTool("get_weather",
    "Get current weather for a city",
    gemini.SchemaObject("parameters",
        map[string]any{
            "city":    gemini.SchemaString("City name"),
            "units":   gemini.SchemaEnum("Temperature units", []string{"celsius", "fahrenheit"}),
        },
        []string{"city"},
    ),
)
gemini.ApplyTools(request, []gemini.FunctionDeclaration{weatherTool}, nil)
```

### Parallel Function Calls

Gemini can issue multiple function calls in a single response. Each call has
a unique `id` field for matching results back.

```go
calls := gemini.ParseToolCalls(response)
results := gemini.DispatchParallel(ctx, calls, myExecutor)
responseParts := gemini.BuildToolResponse(calls, results)
```

### Tool Registry Pattern

```go
registry := gemini.NewToolRegistry()
registry.RegisterTool(weatherDecl, weatherExecutor)
registry.RegisterTool(searchDecl, searchExecutor)

// Dispatch using registered executors
results := registry.DispatchFromRegistry(ctx, calls)
```

### Schema Helpers

| Helper | JSON Schema Type |
|--------|-----------------|
| `SchemaString(desc)` | `{"type":"string","description":"..."}` |
| `SchemaInt(desc)` | `{"type":"integer","description":"..."}` |
| `SchemaNumber(desc)` | `{"type":"number","description":"..."}` |
| `SchemaBool(desc)` | `{"type":"boolean","description":"..."}` |
| `SchemaArray(desc, items)` | `{"type":"array","items":{...}}` |
| `SchemaObject(desc, props, required)` | `{"type":"object","properties":{...}}` |
| `SchemaEnum(desc, values)` | `{"type":"string","enum":[...]}` |

### Gotcha: ID Matching Is Position-Independent

Results must be matched to calls by the `id` field, not by position. However,
`DispatchParallel` returns results in the same order as input calls for
convenience -- the `BuildToolResponse` function pairs them by index.

### Gotcha: ToolConfig Modes

- `AUTO`: model decides whether to call tools (default)
- `ANY`: model MUST call at least one tool
- `NONE`: tools are declared but disabled

---

## 6. Search Grounding

Google Search grounding allows the model to access real-time web information.
Grounding metadata in the response includes citations with source URLs,
confidence scores, and the specific text segments they support.

### Enabling Search Grounding

```go
// Basic grounding (2.x and 3.x)
gemini.EnableSearchGrounding(request)

// 3.x with function calling (VALIDATED mode required)
threshold := 0.7
gemini.EnableSearchGroundingValidated(request, &threshold)
```

### Extracting Citations

```go
// From a synchronous response
gm := gemini.ExtractGroundingMetadata(response)
citations := gemini.ExtractCitationsFromMetadata(gm)
queries := gemini.SearchQueriesFromMetadata(gm)

// Or directly
citations := gemini.ExtractCitationsFromResponse(response)
```

### Citation Structure

```go
type Citation struct {
    URL           string  // Source URL
    Title         string  // Source page title
    Confidence    float64 // 0.0-1.0 confidence score
    SupportedText string  // Text segment this citation supports
}
```

### Grounding Metadata Hierarchy

```
GroundingMetadata
  SearchEntryPoint      -- rendered search entry (HTML/SDK blob)
  GroundingChunks[]     -- web sources (URI, title)
  GroundingSupports[]   -- text segments mapped to chunk indices with confidence
  WebSearchQueries[]    -- actual queries the model used
```

### Gotcha: Search + Function Calling on 3.x

Combining search grounding with function calling requires:
1. `DynamicRetrievalConfig` with `mode: "VALIDATED"`
2. Function declarations in a separate `Tool` entry (not the same one as `googleSearch`)
3. Only works on Gemini 3.x models

### Gotcha: Streaming Grounding

In streaming mode, grounding metadata appears on the final chunk alongside the
`finishReason`. The SSE parser emits an `EventGrounding` event with extracted
citations. The streaming `extractCitations` helper produces basic citations;
for richer citations with confidence scores, use `ExtractCitationsFromMetadata`
on a synchronous response.

---

## 7. System Instructions

System instructions configure the model's behaviour for the entire conversation.
Gemini expects the `systemInstruction` field as a `Content` object (Message),
not a bare string.

### Structured System Prompt

```go
prompt := gemini.SystemPrompt{
    Role:         "You are a precise technical assistant.",
    Constraints:  []string{"Be concise", "Never fabricate citations"},
    OutputFormat: "Use markdown with code blocks.",
    ToolGuidance: "Use tools proactively for file operations.",
}
instruction := gemini.BuildSystemInstruction(prompt)
gemini.ApplySystemInstruction(request, instruction)
```

### From Plain Text

```go
instruction := gemini.BuildSystemInstructionFromText("You are a helpful assistant.")
```

### Temperature Enforcement

```go
// Enforces 1.0 for 3.x models, clamps to [0, 2] for 2.x
gemini.ApplyTemperature(request, "gemini-3.1-pro-preview", 0.5)
// Result: temperature set to 1.0 (3.x override)
```

### Prompt Validation

```go
warnings := gemini.ValidateSystemPrompt(promptText, modelID)
for _, w := range warnings {
    log.Printf("Warning: %s", w)
}
```

Detection patterns:
- Excessive negative constraints (>5 "never"/"don't" chains)
- Very long prompts (>10K chars) -- suggest caching instead
- Temperature instructions in text (should use GenerationConfig)
- "JSON only" instructions without structured output mode
- "Think step by step" on models with native thinking (redundant)

### Gotcha: SystemInstruction Is *Message Not *Part

The Gemini API documents `systemInstruction` as a `Content` object. In Go,
that maps to `*Message` with role `"user"` and a single text Part. Earlier
implementations incorrectly used `*Part` -- this caused silent failures
where the system instruction was ignored.

---

## 8. Safety Settings

Safety settings control which content categories the model filters.
For technical/coding content, the default is `BlockOnlyHigh` to avoid
false positives on code snippets and technical documentation.

### Preset Configurations

```go
gemini.DefaultSafetySettings()    // BlockOnlyHigh -- technical content
gemini.PermissiveSafetySettings() // BlockNone -- controlled environments
gemini.StrictSafetySettings()     // BlockLowAndAbove -- most restrictive
```

### Custom Overrides

```go
settings := gemini.CustomSafety(
    map[string]string{
        gemini.HarmCategoryDangerousContent: gemini.BlockNone,
    },
    gemini.BlockOnlyHigh, // Fallback for unspecified categories
)
```

### Harm Categories

| Category | Constant |
|----------|----------|
| Harassment | `HARM_CATEGORY_HARASSMENT` |
| Hate Speech | `HARM_CATEGORY_HATE_SPEECH` |
| Dangerous Content | `HARM_CATEGORY_DANGEROUS_CONTENT` |
| Sexually Explicit | `HARM_CATEGORY_SEXUALLY_EXPLICIT` |
| Civic Integrity | `HARM_CATEGORY_CIVIC_INTEGRITY` |

### Response Validation

```go
blocked, reasons := gemini.ValidateResponse(response)
if blocked {
    log.Printf("Response blocked: %v", reasons)
}
```

### Gotcha: False Positives on Code

`BlockMediumAndAbove` frequently triggers on code containing security tools,
exploit discussions, or system administration commands. Use `BlockOnlyHigh`
for any coding-related use case.

---

## 9. Token Economics

Token economics drive model selection and caching strategy. The library
provides first-class cost calculation.

### Cost Calculation

```go
cost, err := gemini.TokenEconomics("gemini-2.5-flash", 50000, 1000, 0)
// cost.InputCost = $0.0075
// cost.OutputCost = $0.0006
// cost.TotalCost = $0.0081
```

### With Caching

```go
cost, err := gemini.TokenEconomics("gemini-2.5-flash", 50000, 1000, 45000)
// 45000 cached tokens at 75% discount
// cost.TotalCost = ~$0.0019
// cost.Savings = ~$0.0062
```

### Cost Comparison Table (per request, 50K input / 1K output)

| Model | No Cache | With Cache (90%) | Savings |
|-------|----------|------------------|---------|
| 2.5 Flash | $0.0081 | $0.0019 | 76% |
| 2.5 Pro | $0.0725 | $0.0194 | 73% |
| 3.1 Flash | $0.0081 | $0.0019 | 76% |
| 3.1 Pro | $0.0725 | $0.0194 | 73% |

### Break-Even Analysis

```go
comp, _ := gemini.CacheEconomics("gemini-2.5-flash", 50000, 45000, 1000)
// comp.BreakEvenReqs = 1 (caching wins on the first request)
```

### Key Economic Insights

1. **Cache everything repeatable.** At 75% discount, caching pays for itself
   on the first request for any content above the minimum threshold.
2. **Flash for volume, Pro for depth.** Flash is 8x cheaper on input and 17x
   cheaper on output. Use Pro only when you need 65K output or deep reasoning.
3. **Thinking tokens are free.** Thinking token usage does not count toward
   billing (only prompt and candidate tokens are billed).

---

## 10. Model Routing

The router selects the optimal model based on task classification without
requiring an LLM call. It uses keyword heuristics and structural signals.

### Default Routing Table

| Task Class | Primary Model | Fallback | Thinking | Grounding |
|-----------|---------------|----------|----------|-----------|
| Deep Reasoning | 3.1 Pro | 2.5 Pro | HIGH | No |
| Fast Extraction | 3.1 Flash | 2.5 Flash | No | No |
| Image Generation | 3.1 Flash Image | -- | No | No |
| Code Generation | 3.1 Flash | 2.5 Flash | MEDIUM | No |
| Research | 3.1 Pro | 2.5 Pro | No | Yes |
| Conversation | 2.5 Flash | -- | No | No |
| Large Context | 2.5 Pro | 2.5 Flash | No | No |

### Usage

```go
router := gemini.NewRouter()
taskClass := gemini.ClassifyTask(prompt, hasTools, hasImages)
model, rule, err := router.RouteModel(taskClass)
```

### Classification Heuristics

| Signal | Classification |
|--------|---------------|
| Has images | Image Generation |
| >2000 chars + analysis keywords | Deep Reasoning |
| Code keywords (function, debug, etc.) | Code Generation |
| Question words + factual intent | Research |
| <500 chars + extraction keywords | Fast Extraction |
| >50K chars | Large Context |
| Default | Conversation |

### Custom Routing

```go
router := gemini.NewCustomRouter([]gemini.RoutingRule{
    {TaskClass: gemini.TaskCodeGeneration, PreferredModel: "gemini-2.5-pro"},
})
router.SetRule(gemini.RoutingRule{
    TaskClass: gemini.TaskResearch,
    PreferredModel: "gemini-3.1-pro-preview",
    RequiresGrounding: true,
})
```

### Gotcha: Classification Is Heuristic

The classifier uses keyword matching, not semantic understanding. Short
prompts like "debug" will route to code generation even if the intent is
philosophical. Future versions may add a lightweight LLM-based classifier.

---

## 11. Error Handling

The retry engine handles transient API errors with exponential backoff,
jitter, and user callbacks for persistent rate limiting.

### Retry Configuration

```go
client := gemini.NewClient(
    gemini.WithRetryConfig(gemini.RetryConfig{
        MaxAttempts:  10,
        InitialDelay: 2 * time.Second,
        MaxDelay:     30 * time.Second,
        JitterFraction: 0.2,
        OnPersistent429: func(attempt int) bool {
            log.Printf("Rate limited (attempt %d), continuing...", attempt)
            return true // return false to abort
        },
    }),
)
```

### Error Classification

| HTTP Status | Retryable | Action |
|-------------|-----------|--------|
| 429 | Yes | Exponential backoff + jitter |
| 500, 502, 503 | Yes | Exponential backoff |
| 400, 401, 403, 404 | No | Return error immediately |
| Connection reset / timeout | Yes | Backoff + retry |

### Backoff Formula

```
delay = min(initialDelay * 2^attempt, maxDelay) +/- jitter
jitter = delay * jitterFraction * random[-1, 1]
```

### Persistent 429 Callback

After 5 consecutive 429 errors, the `OnPersistent429` callback is invoked.
If it returns `false`, the retry loop aborts. This allows the application
to prompt the user or switch to a different model.

### Gotcha: Streaming Does Not Retry

The `StreamResponse` method uses the raw HTTP client (not the retry wrapper)
because reconnecting mid-stream is not supported. If the stream fails, the
consumer receives an `EventError` and must decide whether to retry the full
request.

---

## 12. Observation Masking

JetBrains research demonstrates that masking tool observation outputs (replacing
detailed tool results with brief summaries) reduces token costs by 50%+ while
maintaining equivalent solve rates for coding tasks.

### Strategy

For multi-turn coding sessions:

1. **First tool call:** Return full observation (file contents, command output)
2. **Subsequent references:** Replace with summary: "File X: 250 lines, Go, functions: A, B, C"
3. **On edit:** Show only the diff, not the full file

### Implementation Note

Observation masking is not implemented in the Phase 01 client library. It will
be implemented in Phase 02 (context engine) where conversation history
management lives. The key design decision is: masking happens at the context
layer, not at the API layer.

### Key Research Findings

- 50%+ cost reduction with equivalent solve rates (JetBrains, 2025)
- Works because LLMs retain working memory of recent observations
- Most effective for file reads and command outputs (high token cost, low information density)
- Factory.ai's anchored iterative summarisation beats LLM summarisation for compression

---

## 13. Multimodal Input

Gemini models accept images, audio, video, and PDFs as input alongside text.
These are sent either inline (base64) or via the File API for large files.

### Inline Data

```go
parts := []gemini.Part{
    {Text: "Describe this image:"},
    {InlineData: &gemini.InlineData{
        MIMEType: "image/png",
        Data:     base64EncodedImage,
    }},
}
```

### File API (Large Files)

```go
parts := []gemini.Part{
    {Text: "Summarise this PDF:"},
    {FileData: &gemini.FileData{
        MIMEType: "application/pdf",
        FileURI:  "https://generativelanguage.googleapis.com/v1beta/files/abc123",
    }},
}
```

### Token Costs by Media Type

| Type | Approximate Tokens |
|------|--------------------|
| Image (standard) | ~258 tokens per image |
| Image (high-res) | ~514 tokens per image |
| Audio | ~32 tokens per second |
| Video | ~263 tokens per second (video frames + audio) |
| PDF | ~258 tokens per page |

### Supported MIME Types

- Images: `image/png`, `image/jpeg`, `image/webp`, `image/gif`
- Audio: `audio/mp3`, `audio/wav`, `audio/ogg`, `audio/flac`
- Video: `video/mp4`, `video/mpeg`, `video/webm`
- Documents: `application/pdf`

### Implementation Note

Multimodal types (`InlineData`, `FileData`) are defined in Phase 01's
`types.go`. File upload and media processing will be implemented in Phase 02.
The current library supports constructing multimodal requests but does not
yet have convenience helpers for encoding or file upload.

---

## Appendix A: API Endpoints

All endpoints use the `v1beta` prefix (required for caching, grounding, 3.x models).

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/v1beta/models/{model}:generateContent` | POST | Synchronous generation |
| `/v1beta/models/{model}:streamGenerateContent?alt=sse` | POST | Streaming (SSE) |
| `/v1beta/models/{model}:countTokens` | POST | Token counting |
| `/v1beta/cachedContents` | POST | Create cache |
| `/v1beta/cachedContents` | GET | List caches |
| `/v1beta/{name}` | GET | Get cache |
| `/v1beta/{name}` | PATCH | Update cache TTL |
| `/v1beta/{name}` | DELETE | Delete cache |

## Appendix B: Library File Map

| File | Lines | Purpose |
|------|-------|---------|
| `types.go` | 354 | All API types and structures |
| `models.go` | 188 | Model registry with pricing |
| `client.go` | 248 | HTTP client with retry integration |
| `retry.go` | ~190 | Retry engine with exponential backoff |
| `streaming.go` | 257 | SSE streaming parser |
| `thought.go` | 131 | Thought signature handling |
| `safety.go` | 120 | Safety settings management |
| `structured.go` | 161 | Structured output (JSON Schema) |
| `cache.go` | 316 | Context caching (CachedContents REST) |
| `tools.go` | 274 | Function calling with parallel dispatch |
| `router.go` | 249 | Model router with task classification |
| `system.go` | 202 | System prompt engineering |
| `grounding.go` | ~150 | Search grounding API |

## Appendix C: Quick Start

```go
package main

import (
    "context"
    "fmt"
    "gkestral/internal/gemini"
    "os"
)

func main() {
    client := gemini.NewClient(
        gemini.WithAPIKey(os.Getenv("GEMINI_API_KEY")),
    )

    req := &gemini.GenerateContentRequest{
        Contents: []gemini.Message{{
            Role:  "user",
            Parts: []gemini.Part{{Text: "Hello, Gemini!"}},
        }},
    }

    events := client.StreamResponse(context.Background(), "gemini-2.5-flash", req)
    for ev := range events {
        if ev.Type == gemini.EventText {
            fmt.Print(ev.Text)
        }
    }
    fmt.Println()
}
```
