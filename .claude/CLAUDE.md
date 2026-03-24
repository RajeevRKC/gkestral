# Gkestral -- Gemini-Native Desktop Workbench

> Gemini-native desktop workbench for turning scattered information into
> finished documents, presentations, and research-backed outputs.
> Tier: T1 (ship-grade, targeting Google showcase)

## Quick Context

- **Language:** Go 1.23+ (currently 1.25.6)
- **UI:** Wails v2 (native webview, not Electron) -- Phase 03
- **API:** Raw REST + SSE (no Gemini SDK dependency)
- **Default model:** gemini-2.5-flash (routing to Pro/Flash by task)
- **Architecture:** Architect & Surgeon -- 1M input, iterative 8K output

## Key Design Principles

1. **Gemini intelligence first, UI second.** The API integration quality IS the product.
2. **Context caching is the economic moat.** 90% discount on stable context.
3. **Thought signatures are mandatory.** Gemini 3.x requires circulating them.
4. **Temperature stays at 1.0.** Lower values cause Gemini 3 to loop.
5. **Observation masking over summarisation.** JetBrains proved it's superior.
6. **Single binary, zero runtime deps.** No npm, no Python, no Bun.
7. **Windows-first testing.** Commander's primary platform.

## FABRICATE

Project files: `.fabricate/PROJECT.md`, `ROADMAP.md`, `STATE.md`
Current: Milestone 1 (v0.1.0), Phase 01 -- Gemini Mastery (COMPLETE)
Plan: `.fabricate/phases/01-gemini-mastery/01-01-PLAN.md` (16/16 tasks, 8/8 waves)
Progress: ALL DONE. 88.3% coverage. Three-way handshake complete.
Handoff: `.fabricate/phases/01-gemini-mastery/HANDOFF-2026-03-24.md`
Next: Close Phase 01 (SUMMARY.md), then plan Phase 02

## Project Structure

```
go.mod                          -- Module: gkestral
internal/gemini/                -- Gemini client library (Phase 01 deliverable)
  types.go                      -- All API types and structures
  models.go                     -- Model registry with pricing
  retry.go                      -- Retry engine with exponential backoff
  client.go                     -- HTTP client with retry integration
  streaming.go                  -- SSE streaming parser
  thought.go                    -- Thought signature handling (3.x)
  safety.go                     -- Safety settings management
  structured.go                 -- Structured output (JSON Schema)
  cache.go                      -- Context caching (CachedContents REST)
  tools.go                      -- Function calling with parallel dispatch
  router.go                     -- Model router with task classification
  system.go                     -- System prompt engineering
  grounding.go                  -- Search grounding API
  integration_test.go           -- Integration tests (build tag: integration)
  *_test.go                     -- 88.3% coverage
docs/gemini-mastery-reference.md -- Phase 01 reference document (13 API areas)
output/kestrel/                 -- POC prototype (reference, migrates to Wails in Phase 03)
website/                        -- Marketing site (gkestral.com)
.fabricate/                     -- FABRICATE project management
  references/study/             -- Source code studies (Gemini CLI, OpenCode, Entropic)
docs/                           -- (Phase 01 will create gemini-mastery-reference.md)
```

## References

Research documents in `.fabricate/references/`:
- `gemini-api-deepdive.md` -- 12-area API capability study
- `autoresearch-patterns.md` -- Self-improvement architecture
- `gemini-cli-source-study.md` -- Official CLI codebase analysis
- `study/CONSOLIDATED-FINDINGS.md` -- Adopt/avoid/build matrix from 3 source studies

## Commands

```bash
# Development
go test ./internal/gemini/           # Run unit tests
go test -cover ./internal/gemini/    # Coverage report
go test -v ./internal/gemini/        # Verbose output
go vet ./...                         # Static analysis

# POC (output/kestrel/)
cd output/kestrel && go run .        # Run POC prototype

# Note: -race flag requires CGO (disabled on this Windows install)
```

## Windows Note

CGO is disabled. The `-race` flag will not work. Race detection needs CI with CGO enabled.
