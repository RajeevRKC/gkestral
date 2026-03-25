---
phase: 2
plan: 1
title: "Google API & SDK Landscape -- OAuth, Drive, Gmail, Strategy"
status: complete
started: 2026-03-25
completed: 2026-03-25
duration_days: 1
tier: T1

provides:
  - "internal/google/transport/ -- shared Google API HTTP transport with bounded pagination"
  - "internal/google/auth/ -- OAuth 2.0 desktop PKCE, OS keyring token store, progressive scopes"
  - "internal/google/drive/ -- Drive API v3 client (list, search, download, upload)"
  - "internal/google/gmail/ -- Gmail API v1 client (search, extract, batch, attachments)"
  - "docs/google-api-strategy.md -- rates, pricing, scope economics, SDK analysis"
  - "docs/prompt-engineering-playbook.md -- Gemini-specific patterns + agent architecture"

requires:
  - "Go 1.23+ with golang.org/x/oauth2, golang.org/x/crypto, golang.org/x/time"
  - "Google Cloud project with OAuth consent screen (for integration tests)"
  - "GEMINI_API_KEY for Gemini operations (API key auth, not OAuth)"

affects:
  - "Phase 03 will build Wails architecture on top of these clients"
  - "Phase 04 hero demo will use Drive import, Gmail import, and Gemini streaming"
---

# Summary: Phase 02, Plan 01 -- Google API & SDK Landscape

## Outcome

**COMPLETE.** All 14 tasks across 8 waves delivered. Built a complete Google
services layer (OAuth, Drive, Gmail) alongside the Phase 01 Gemini client.
Three-way AR-3 handshake completed with 25 findings fixed across all reviewers.

## Quality Gates

| Gate | Status | Detail |
|------|--------|--------|
| `go vet ./...` | PASS | Zero warnings |
| Tests (google) | PASS | 161/161 passing |
| Tests (gemini) | PASS | 151/151 passing (maintained) |
| Coverage: auth | 73.5% | Below 80% -- gap is external system calls (gcloud, browser) |
| Coverage: transport | 91.9% | Above 80% |
| Coverage: drive | 89.3% | Above 80% |
| Coverage: gmail | 88.4% | Above 80% |
| Coverage: gemini | 88.3% | Above 85% (maintained) |
| AR-1 Handshake | PASS | Gemini DT (7) + Codex (8) -- all fixed in plan |
| AR-3 Three-Way | PASS | Codex (7) + M2.7 (23) + Gemini DT (8) -- 25 fixed |

## AR-3 Three-Way Handshake Results

### Codex GPT-5.4 (Architecture + Code Quality)
- **Verdict:** REJECTED -> fixed -> APPROVED
- **Findings:** 2 HIGH, 5 MEDIUM
- HIGH: Gmail attachment base64url padding mismatch -- FIXED
- HIGH: TokenRefresher callbacks under mutex -- FIXED
- MEDIUM: Upload bypasses transport -- DEFERRED (Phase 03)
- MEDIUM: Boundary race condition -- FIXED (then upgraded to crypto/rand)
- MEDIUM: Credential file errors silently discarded -- FIXED
- MEDIUM: OAuth exchangeCode uses http.DefaultClient -- FIXED
- MEDIUM: Integration tests are placeholders -- ACKNOWLEDGED

### Minimax M2.7 (Go Idioms + Runtime Safety)
- **Verdict:** REJECTED -> fixed -> APPROVED
- **Findings:** 8 CRITICAL, 15 IMPORTANT, 9 NIT
- CRITICAL: Nil HTTP client panic -- FIXED
- CRITICAL: TCP listener leak in OAuth -- FIXED
- CRITICAL: Command injection in GCloudCreateProject -- FIXED (regex validation)
- CRITICAL: Command injection in OpenBrowser -- FIXED (URL scheme validation)
- CRITICAL: Buggy contains/searchString -- FIXED (strings.Contains)
- CRITICAL: Fragile upload URL construction -- DEFERRED
- CRITICAL: Semaphore deadlock in BatchExtract -- FIXED (select pattern)
- CRITICAL: Boundary race -- FIXED (crypto/rand)
- IMPORTANT: DoJSON body leak -- FIXED
- IMPORTANT: PaginatedList body leak -- FIXED
- Others: mix of fixed and deferred

### Gemini Deep Think (API Protocol + Security)
- **Verdict:** APPROVED WITH WARNINGS
- **Findings:** 1 CRITICAL, 3 HIGH, 3 MEDIUM, 1 LOW
- CRITICAL: Upload OOM (io.ReadAll) -- DEFERRED (needs resumable uploads)
- HIGH: Iterator empty page truncation -- FIXED
- HIGH: Predictable boundary -- FIXED (crypto/rand)
- HIGH: TokenRefresher deadlock -- ALREADY FIXED (M2.7)
- MEDIUM: OAuth callback sync.Once DoS -- FIXED (state validation)
- MEDIUM: API key in URL query -- FIXED (x-goog-api-key header)
- MEDIUM: Email date parsing -- FIXED (net/mail.ParseDate)
- LOW: BatchExtract semaphore drain -- ALREADY FIXED (M2.7)

## Metrics

| Metric | Value |
|--------|-------|
| Tasks | 14/14 complete |
| Waves | 8/8 complete |
| Duration | 1 day (2026-03-25) |
| Commits | 7 |
| New source files | 13 (.go) |
| New test files | 15 (.go, incl. integration) |
| New Go lines | ~6,500 |
| New doc lines | ~660 |
| AR-3 findings fixed | 25 |

## Deferred Items

| Item | Source | Target |
|------|--------|--------|
| Upload bypasses transport layer | Codex AR-3 | Phase 03 (add transport.DoRaw) |
| Upload OOM (io.ReadAll) | Gemini DT AR-3 | Phase 03 (resumable upload protocol) |
| Fragile upload URL construction | M2.7 AR-3 | Phase 03 (store upload base URL separately) |
| Integration tests need real assertions | Codex AR-3 | When GCP credentials available |
| PBKDF2 iterations below OWASP 2023 rec | M2.7 NIT | Make configurable |

## Files Delivered

```
internal/google/transport/
  transport.go          205 lines -- Shared HTTP transport, pagination, errors
  transport_test.go     442 lines -- 43 tests

internal/google/auth/
  scopes.go             162 lines -- Scope management, progressive permissioning
  token.go              328 lines -- OS keyring + PBKDF2 token store, auto-refresh
  oauth.go              330 lines -- Desktop PKCE flow, state validation
  config.go             103 lines -- Credential resolution chain
  setup.go              168 lines -- Setup wizard, gcloud integration
  *_test.go           1,502 lines -- 63 tests + integration stubs

internal/google/drive/
  types.go               57 lines -- DriveFile, MIME format detection
  client.go             256 lines -- List, search, iterator, metadata
  files.go              214 lines -- Download (binary + export), upload (multipart/related)
  *_test.go             595 lines -- 28 tests + integration stubs

internal/google/gmail/
  types.go              102 lines -- Message types, header extraction
  client.go             187 lines -- Search, metadata, labels
  messages.go           258 lines -- Content extraction, MIME walking, batch extract
  *_test.go             699 lines -- 27 tests + integration stubs

docs/
  google-api-strategy.md          283 lines
  prompt-engineering-playbook.md  374 lines
```

## Next Phase

Phase 03: Architecture from Intelligence -- System architecture informed by
Phase 01 + 02 findings. Wails v2 scaffold, context management, model routing
integration, session persistence, tool architecture.
