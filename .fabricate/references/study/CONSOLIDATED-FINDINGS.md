# Consolidated Source Code Study -- gKestral Intelligence Gathering

Date: 2026-03-23
Sources: Gemini CLI (98k stars), OpenCode (120k stars), Entropic (just open-sourced)

---

## 1. GEMINI API PATTERNS (from Gemini CLI)

### Authentication
- Three strategies: API key, OAuth 2.0 PKCE, Application Default Credentials
- API key supports env vars (`$ENV_VAR`) and shell commands (`!command`)
- OAuth uses PKCE with browser launch + localhost callback
- Auto-retry on 401/403 (max 3 attempts)

### Streaming
- Custom fork of `@google/genai` chats.ts (SDK bug workaround for function responses)
- Stream events: CHUNK, RETRY (discard partial), AGENT_EXECUTION_STOPPED, BLOCKED
- Mid-stream retry: 4 attempts, 1s initial, exponential backoff
- Live output callback pattern for long-running tools

### Retry & Resilience
- 10 max attempts, 5s initial, 30s max delay
- Retryable: ECONNRESET, ETIMEDOUT, EPIPE, ENOTFOUND, SSL errors
- 429: full backoff with `onPersistent429()` callback for user intervention
- Model fallback: preview -> default tier automatic downgrade

### Thought Signatures (Gemini 3.x)
- `SYNTHETIC_THOUGHT_SIGNATURE = 'skip_thought_signature_validator'`
- Custom validation: text must be non-empty, thought must be false, no function data
- Handled in forked chat handler, not via SDK

### Context Caching
- `cached_content_token_count` tracked in telemetry
- Cache headers handled implicitly by SDK
- Chat compression service for history management
- Session recording with cache awareness

### Model Routing
- Preview: 3.1-pro-preview, 3-flash-preview, 3.1-flash-lite-preview
- Default: 2.5-pro, 2.5-flash, 2.5-flash-lite
- Aliases: auto->3.1Pro, pro->3.1Pro, flash->3Flash, flash-lite->2.5FlashLite
- Automatic downgrade if preview access unavailable

### Tools (15 registered)
glob, grep_search, list_directory, read_file, read_many_files, run_shell_command,
write_file, replace, google_web_search, web_fetch, write_todos, save_memory,
get_internal_docs, activate_skill, ask_user

### MCP Support
- 4 transports: SSE, Stdio, Streamable HTTP, Xcode bridge
- Tool naming: `mcp_{server}_{tool}` with wildcard support
- 10-minute default timeout
- Per-server lifecycle with diagnostic deduplication

### What They DON'T Have
- No Google Drive/Gmail/Calendar
- No DOCX/PPTX generation
- No canvas/artifact pane
- No structured knowledge base or RAG

---

## 2. GO ARCHITECTURE PATTERNS (from OpenCode)

### Package Structure (adopt this)
```
cmd/           -- CLI entry (Cobra)
internal/
  app/         -- Main orchestrator
  db/          -- SQLc-generated database layer
  llm/
    provider/  -- Provider abstraction (Gemini, Claude, OpenAI...)
    agent/     -- Stateful request processor
    models/    -- Model definitions
    prompt/    -- Prompt management
    tools/     -- Tool interface + implementations
  tui/         -- Terminal UI (Bubble Tea)
  config/      -- Configuration (Viper)
  session/     -- Session service
  message/     -- Chat history
  permission/  -- Tool execution gating
  pubsub/      -- Generic event broker
```

### Key Go Patterns (adopt all)

**1. Provider Interface:**
```go
type Provider interface {
  SendMessages(ctx, messages, tools) (*ProviderResponse, error)
  StreamResponse(ctx, messages, tools) <-chan ProviderEvent
  Model() models.Model
}
```

**2. Functional Options:**
```go
type ProviderClientOption func(*providerClientOptions)
// Usage: NewProvider(Gemini, WithAPIKey("..."), WithMaxTokens(8192))
```

**3. Generic Pub-Sub Broker:**
```go
type Broker[T any] struct { subs map[chan Event[T]]struct{}; mu sync.RWMutex }
```

**4. Channel-Based Streaming:**
```go
func stream(ctx context.Context, ...) <-chan ProviderEvent {
  ch := make(chan ProviderEvent)
  go func() { defer close(ch); /* stream loop */ }()
  return ch
}
```

**5. sync.Map for concurrent sessions** (no global lock)
**6. SQLc for type-safe database access** (compile-time verified)
**7. Context-based cancellation** throughout
**8. Error wrapping with %w** at every level

### Gemini Integration (OpenCode's approach)
- Uses `google.golang.org/genai` SDK
- Message roles: User->"user", Assistant->"model", Tool->"function"
- Tool declaration via `genai.FunctionDeclaration`
- Streaming via `chat.SendMessageStream(ctx, ...parts)`
- Retry: exponential backoff 2s * 2^(n-1) + 20% jitter, max 8 retries
- Token counting from `resp.UsageMetadata`

### Session Persistence (SQLite)
```sql
sessions: id, parent_session_id, title, message_count, prompt_tokens,
          completion_tokens, cost, summary_message_id, timestamps
messages: id, session_id, role, parts(JSON), model, timestamps
files:    id, session_id, path, content, version, timestamps
```

### Agent Execution Loop
1. Check `IsSessionBusy()` via sync.Map
2. Create cancellable context
3. Load message history (with summary truncation)
4. Stream to provider, accumulate response
5. If tool calls: execute tools, append results, loop back to step 4
6. If done: publish AgentEvent, close channel

### MCP Client
- Stdio + SSE transports
- Tool wrapping: MCP tools appear as native tools via `mcpTool` adapter
- Per-server client lifecycle
- Permission gating on MCP tool execution

---

## 3. GOOGLE SERVICE INTEGRATION (from Entropic)

### OAuth Pattern (PKCE, localhost callback)
```
1. Generate code_verifier + code_challenge (S256)
2. Bind TCP server to 127.0.0.1:0 (random port)
3. Build auth URL with scopes, redirect_uri, PKCE params
4. Open system browser
5. Wait for callback with auth code
6. Exchange code for tokens (access + refresh)
7. Store in encrypted vault
```

### Google Scopes
- Email: gmail.modify, gmail.labels, userinfo.email, openid
- Calendar: calendar, calendar.readonly, userinfo.email, openid
- Progressive: request minimal first, expand on demand

### Token Management
- Refresh tokens never expire; access tokens expire in 3600s
- Early refresh at 90s before expiry
- Throttle refreshes to 30-second minimum
- Failed refresh -> session expiry -> re-authenticate

### Two-Tier Token Storage
1. **Encrypted vault** (Stronghold/SQLite): full token bundle
2. **Plaintext index** (JSON): provider, email, scopes only (for UI)

### Gmail API Patterns
```
GET  /users/me/messages          -- List with query filter
GET  /users/me/messages/{id}     -- Full message + metadata
POST /users/me/messages/send     -- Raw RFC 2822, base64url encoded
```

### Calendar API Patterns
```
GET  /calendars/primary/events   -- List with timeMin/timeMax
POST /calendars/primary/events   -- Create event
```

### Desktop Architecture (Tauri, translate to Wails)
- Frontend (React) communicates with backend via `invoke()` + events
- File system access via native APIs
- Plugin architecture for shell, store, encryption, dialogs, deep-link, auto-update

### Quick Actions Pattern
```json
{ "id": "inbox_cleanup", "kind": "agent", "label": "Clean up my inbox",
  "requirement": { "kind": "integration", "provider": "google_email" } }
```

---

## 4. STRATEGIC SYNTHESIS -- What gKestral Must Do

### ADOPT from all three:

| Pattern | Source | gKestral Implementation |
|---------|--------|------------------------|
| Provider interface | OpenCode | Go interface, Gemini-first |
| Channel streaming | OpenCode | `<-chan ProviderEvent` |
| Generic pub-sub | OpenCode | `Broker[T]` for service decoupling |
| SQLite + sqlc | OpenCode | Type-safe session persistence |
| Functional options | OpenCode | Clean configuration |
| PKCE OAuth localhost | Entropic | Go net/http listener on :0 |
| Two-tier token storage | Entropic | Encrypted SQLite + plaintext cache |
| Token refresh throttle | Entropic | 30s minimum, 90s early refresh |
| Progressive scopes | Entropic | Minimal first, expand on demand |
| Retry with user callback | Gemini CLI | `onPersistent429()` pattern |
| Hook system | Gemini CLI | Before/after tool execution |
| Model fallback chain | Gemini CLI | Preview -> default auto-downgrade |
| Live output streaming | Gemini CLI | `outputUpdateHandler` callback |
| Session recording | Gemini CLI | History + resume capability |

### AVOID:

| Anti-Pattern | Source | Why |
|-------------|--------|-----|
| SDK forking | Gemini CLI | Maintenance burden, contribute upstream instead |
| Parallel tool registries | Gemini CLI | Unified registry from day one |
| Docker dependency | Entropic | Pure Go, no container runtime |
| Bubble Tea TUI | OpenCode | Wails webview instead (richer UI) |
| Node.js runtime | Gemini CLI | Single Go binary, zero deps |

### BUILD (nobody has these):

| Capability | Why It's Our Moat |
|-----------|-------------------|
| DOCX/PPTX generation pipeline | Zero tools do research-to-document |
| Google Drive/Gmail as context sources | Only Entropic has email; nobody has Drive |
| Context caching as visible feature | Nobody surfaces cache economics to users |
| Right-pane artifact canvas | No desktop CLI has a two-pane workbench |
| Research cards with citations | No tool structures grounded research visually |
| Gemini 3.1 native optimization | Everyone wraps SDK; we exploit the API |

---

## 5. IMMEDIATE NEXT STEPS

Phase 01 (Gemini Mastery) should focus on:
1. Build Go Gemini client using raw REST + SSE (not SDK -- avoid forking problems)
2. Implement thought signature handling for 3.x
3. Build context caching with stable/active split and TTL management
4. Implement model routing: 3.1-pro -> 3.1-flash -> 2.5-flash
5. Test Search Grounding via native API tools array
6. Build retry engine with exponential backoff + 429 callback
7. Benchmark token economics across models

Phase 02 (Google APIs) should focus on:
1. Implement PKCE OAuth with localhost callback (Go net/http)
2. Build token vault (encrypted SQLite)
3. Gmail read/import API client
4. Drive browse/open/save API client
5. Progressive scope management
6. Token refresh goroutine with throttling
