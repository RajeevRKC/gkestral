# Google API Strategy -- Gkestral v0.1

> Phase 02 deliverable. Maps the Google API surface for information access,
> rates, pricing, scope economics, and SDK trade-offs.

---

## 1. API Inventory for v0.1

| API | Version | Auth | Base URL | Purpose |
|-----|---------|------|----------|---------|
| Gemini API | v1beta | API key (`x-goog-api-key`) | `generativelanguage.googleapis.com` | Core AI engine -- streaming, caching, grounding, tools |
| Google Drive | v3 | OAuth 2.0 | `www.googleapis.com/drive/v3` | File browse, download, upload, workspace integration |
| Gmail | v1 | OAuth 2.0 | `gmail.googleapis.com/gmail/v1` | Email search, content extraction, context import |
| OAuth 2.0 | v2 | -- | `accounts.google.com/o/oauth2/v2` | Authentication layer for Drive + Gmail |

**Not in v0.1:** Vertex AI, Calendar, Maps, NotebookLM, Firebase, Stitch.

---

## 2. Rate Limits and Quotas

### Gemini API

| Model | Requests/min (free) | Requests/min (paid) | Tokens/min | Daily limit |
|-------|--------------------|--------------------|------------|-------------|
| gemini-3.1-pro-preview | 2 | 1,000 | 4M | No hard cap |
| gemini-3.1-flash | 15 | 2,000 | 4M | No hard cap |
| gemini-2.5-pro | 5 | 1,000 | 4M | No hard cap |
| gemini-2.5-flash | 15 | 2,000 | 4M | No hard cap |

**Context caching:** Cached content counts toward storage quota (bytes), not request quota. Minimum 1024 tokens (Flash) or 4096 tokens (Pro). TTL: 1 minute to 48 hours.

### Drive API v3

| Quota | Free Tier | Google Workspace |
|-------|-----------|-----------------|
| Queries/day | 1,000,000,000 | Same |
| Queries/100 seconds/user | 12,000 | 12,000 |
| Upload bandwidth | 750 GB/day | 750 GB/day |
| Download bandwidth | 10 GB/day | 10 GB/day |
| File size limit | 5 TB | 5 TB |

**Practical for Gkestral:** file listing and metadata are cheap. Download bandwidth (10 GB/day) is the real constraint for heavy document import workflows.

### Gmail API v1

| Quota | Value |
|-------|-------|
| Quota units/second/user | 250 |
| messages.list | 5 units/call |
| messages.get | 5 units/call |
| messages.send | 100 units/call |
| Effective messages.get/second | ~50 (with headroom) |
| Batch extract rate | 40 req/s via Token Bucket (implemented) |

**Practical:** batch extraction of 100 emails takes ~3 seconds. Good enough for context import.

### OAuth 2.0

No per-request quota. Token refresh is rate-limited informally -- excessive refresh attempts may trigger temporary blocks. Our implementation refreshes with 60-second buffer before expiry.

---

## 3. Pricing Analysis

### Gemini API (per 1M tokens)

| Model | Input | Output | Cached Input | Thinking |
|-------|-------|--------|-------------|----------|
| gemini-3.1-pro-preview | $2.50 | $10.00 | $0.625 (75% off) | $2.50 |
| gemini-3.1-flash | $0.15 | $0.60 | $0.0375 (75% off) | $0.15 |
| gemini-2.5-pro | $1.25 | $5.00 | $0.125 (90% off) | $1.25 |
| gemini-2.5-flash | $0.075 | $0.30 | $0.0075 (90% off) | $0.075 |

**Caching is the economic moat.** For a typical 100K-token context:
- Without caching: $0.125/request (Flash 2.5)
- With caching: $0.0125/request (90% discount)
- Break-even: 2 requests with the same cached context

### Drive + Gmail APIs

Free with Google account. No per-request cost. Storage counts toward Google One quota (15 GB free, expandable with paid plans).

### OAuth 2.0

Free. No cost.

**Total v0.1 cost model:** Gemini API tokens are the only variable cost. Drive and Gmail are free. OAuth is free. The product's unit economics hinge entirely on context caching efficiency.

---

## 4. Scope Economics

### Minimal v0.1 Scope Set

| Scope | When Requested | Why |
|-------|---------------|-----|
| `userinfo.email` | First sign-in | Identity, account display |
| `drive.readonly` | First Drive action | File browsing and download |
| `gmail.readonly` | First Gmail action | Email search and import |

**Not requested by default:**
- `generative-language` -- Gemini uses API key auth, not OAuth
- `drive.file` -- too broad for v0.1
- `gmail.compose` -- send capability deferred to v0.2

### Progressive Permissioning Flow

```
User opens Gkestral
  -> Sign in (email scope only, minimal consent screen)
  -> Use Gemini features (API key, no additional consent)
  -> First "Import from Drive" action
     -> Prompt: "Gkestral needs access to your Google Drive files"
     -> Add drive.readonly (include_granted_scopes=true preserves email)
  -> First "Import from Gmail" action
     -> Prompt: "Gkestral needs access to your Gmail messages"
     -> Add gmail.readonly (preserves email + drive)
```

**Why this matters for adoption:**
1. Small initial consent screen = higher sign-in conversion
2. Permissions tied to visible user action = trust building
3. `include_granted_scopes=true` prevents scope regression

### Scope Expansion Path (v0.2+)

| Scope | Trigger | Version |
|-------|---------|---------|
| `drive.file` | "Save to Drive" action | v0.2 |
| `gmail.compose` | "Draft reply" action | v0.2 |
| `calendar.readonly` | "Check my schedule" action | v0.3 |
| `generative-language.retriever` | Semantic retrieval features | v0.3 |

---

## 5. SDK vs Raw REST Trade-offs

### Option A: google-api-go-client (Official SDK)

**Pros:**
- Auto-generated from Google's discovery documents
- Built-in pagination, retry, error handling
- Type-safe request/response builders
- Maintained by Google

**Cons:**
- Massive dependency: 60+ generated packages
- Binary size impact: adds ~15-20 MB
- Version coupling: SDK updates may break
- Abstraction leaks: generated code is hard to debug
- Inconsistent API surface across services

### Option B: Raw REST (chosen)

**Pros:**
- Lightweight: only `golang.org/x/oauth2` and `golang.org/x/time` as deps
- Full control over request construction and error handling
- Consistent patterns across all Google APIs (shared transport layer)
- Small binary impact: ~500 KB for all Google clients
- Easy to debug: no generated code
- Matches Phase 01 Gemini client approach

**Cons:**
- Manual pagination (mitigated: generic `PaginatedList[T]` in transport)
- Manual error mapping (mitigated: shared error classification)
- Must track API changes manually

### Decision: Raw REST

Consistency with Phase 01, smaller binary, full control. The shared transport layer (`internal/google/transport`) eliminates the main disadvantages by providing pagination, error classification, and request building as reusable infrastructure.

---

## 6. Vertex AI Analysis

### When to Use Vertex AI

| Feature | Direct Gemini API | Vertex AI |
|---------|------------------|-----------|
| Basic generation | Yes | Yes |
| Context caching | Yes | Yes |
| Grounding | Yes (Google Search) | Yes + enterprise sources |
| VPC-SC | No | Yes |
| Customer-managed encryption | No | Yes |
| SLA | Best effort | Enterprise SLA |
| Auth | API key | Service account / OAuth |
| Pricing | Same per-token | Same + platform fee |

### Decision for v0.1: Skip Vertex

- No enterprise customers yet
- Same per-token pricing
- API key auth is simpler for desktop app
- Adds significant complexity (project/region/endpoint management)
- **Revisit trigger:** first enterprise customer or Google partnership

### Migration Path

If needed, migration is straightforward:
1. Change base URL from `generativelanguage.googleapis.com` to `{region}-aiplatform.googleapis.com`
2. Switch auth from API key to service account
3. Add project ID and region to request paths
4. The client library is already REST-based, so the switch is configuration, not architecture

---

## 7. MCP Protocol Assessment

### What MCP Offers

Model Context Protocol standardises how AI tools expose capabilities:
- Discovery: tools register themselves
- Schema: JSON Schema for tool parameters
- Transport: stdio, HTTP, or WebSocket
- Ecosystem: growing plugin marketplace

### For Gkestral v0.1

Native Go tool implementations are faster and lighter than MCP:
- No JSON Schema serialisation overhead
- No process spawning for tool calls
- Type-safe at compile time
- Lower latency (in-process vs IPC)

### Decision: No MCP in v0.1

But design the tool interface to be **MCP-bridgeable**:
- Tool declarations already use JSON Schema (Phase 01 `tools.go`)
- Function calling dispatch is already name-based
- Adding an MCP server wrapper in v0.2 would be a thin adapter

### MCP Value for v0.2+

- Plugin ecosystem: users could add their own tools
- IDE integration: VS Code MCP support
- Cross-tool interop: share tools between Gkestral and other AI agents
- **Decision:** build MCP server adapter when plugin architecture is designed (v0.2)

---

## Appendix A: API Endpoint Quick Reference

### Gemini API
```
POST /v1beta/models/{model}:generateContent
POST /v1beta/models/{model}:streamGenerateContent?alt=sse
POST /v1beta/models/{model}:countTokens
POST /v1beta/cachedContents
GET  /v1beta/cachedContents
GET  /v1beta/cachedContents/{name}
DELETE /v1beta/cachedContents/{name}
PATCH /v1beta/cachedContents/{name}
```

### Drive API v3
```
GET  /drive/v3/files                    -- List/search files
GET  /drive/v3/files/{id}               -- Get file metadata
GET  /drive/v3/files/{id}?alt=media     -- Download binary
GET  /drive/v3/files/{id}/export        -- Export Google format
POST /upload/drive/v3/files             -- Upload (multipart/related)
```

### Gmail API v1
```
GET  /gmail/v1/users/me/messages        -- List/search messages
GET  /gmail/v1/users/me/messages/{id}   -- Get message (metadata/full)
GET  /gmail/v1/users/me/messages/{id}/attachments/{aid} -- Get attachment
GET  /gmail/v1/users/me/labels          -- List labels
```

## Appendix B: Scope String Reference

| Constant | Scope URL | Requested In |
|----------|-----------|-------------|
| `ScopeUserInfoEmail` | `https://www.googleapis.com/auth/userinfo.email` | Default (sign-in) |
| `ScopeDriveReadOnly` | `https://www.googleapis.com/auth/drive.readonly` | First Drive action |
| `ScopeGmailReadOnly` | `https://www.googleapis.com/auth/gmail.readonly` | First Gmail action |
| `ScopeGemini` | `https://www.googleapis.com/auth/generative-language` | Future (retrieval) |
| `ScopeDriveFile` | `https://www.googleapis.com/auth/drive.file` | v0.2 (save) |
| `ScopeGmailCompose` | `https://www.googleapis.com/auth/gmail.compose` | v0.2 (draft) |
