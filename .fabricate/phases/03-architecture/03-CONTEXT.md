---
phase: 3
title: "Architecture from Intelligence"
status: active
created: 2026-03-25
---

# Phase 03 Context: Architecture from Intelligence

## Goal

Design the architecture AFTER we understand the engine. Every design decision
grounded in Phase 01-02 findings. Produce a working Wails v2 application that
integrates all existing libraries.

## What We Have (Phase 01 + 02)

### Gemini Client Library (internal/gemini/ -- 14 modules, 151 tests, 88.3% cov)
- SSE streaming with channel-based consumer
- Context caching (90% discount on 2.5, 75% on 3.x)
- Function calling with parallel dispatch
- Search grounding with citation extraction
- Model routing by task classification
- Thought signature handling for 3.x
- System prompt engineering with temperature enforcement
- Retry with exponential backoff and model fallback

### Google Services Layer (internal/google/ -- 4 packages, 161 tests)
- OAuth 2.0 desktop PKCE with OS keyring token store
- Progressive scope management (start narrow, expand on demand)
- Setup wizard infrastructure (gcloud + manual paths)
- Drive API v3: list, search, download, upload (bounded pagination + iterator)
- Gmail API v1: search, content extraction, batch extract (rate-limited)
- Shared transport: error classification, bounded pagination

### Reference Documents
- docs/gemini-mastery-reference.md (796 lines -- all 13 API areas)
- docs/google-api-strategy.md (283 lines -- rates, pricing, scope economics)
- docs/prompt-engineering-playbook.md (374 lines -- patterns + agent architecture)

### Deferred Items from AR-3
- Upload OOM (needs resumable upload protocol)
- Upload bypasses transport (needs transport.DoRaw)
- Fragile upload URL construction
- Integration tests need real GCP credentials
- CGO/-race needs CI pipeline

## What Phase 03 Must Build

From ROADMAP:
1. System architecture informed by Gemini capability map
2. Wails v2 scaffold with two-pane UI
3. Go backend structure: engine, tools, services, workspace
4. Context management strategy (observation masking, cache-aware partitioning)
5. Model routing implementation (integrated with existing router.go)
6. Google service integration layer (wire OAuth + Drive + Gmail into engine)
7. Session persistence and workspace model
8. Tool architecture (file ops, search, command execution)

From PROJECT.md Architecture:
- Execution Core: tool calling, file ops, terminal, planning, diffs
- Context Engine: repo ingestion, codebase packing, cache-aware partitioning, memory tiers
- Research Engine: grounding, source ranking, citation retention
- Gemini Integration: already built (Phase 01), wire into engine
- Integration Layer: wire Google services (Phase 02) into engine

## Key Constraints

- Wails v2 must be installed on Windows (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)
- CGO is disabled on Windows -- SQLite must use modernc.org/sqlite (pure Go)
- Single binary distribution -- no runtime dependencies
- Two-pane UI: chat/direction left, artifacts/multimodal right
- Custom DOM rendering (not terminal emulation)
- Windows-first testing

## Architecture Decision: Package Layout

```
cmd/
  gkestral/
    main.go                   -- Wails app entry point
internal/
  gemini/                     -- Phase 01 (complete)
  google/                     -- Phase 02 (complete)
    auth/
    drive/
    gmail/
    transport/
  engine/                     -- Phase 03 (NEW)
    engine.go                 -- Core conversation engine
    conversation.go           -- Turn management, history
    context.go                -- Context budget, observation masking
    tools.go                  -- Tool registry, dispatch
    session.go                -- Session persistence (SQLite)
  tools/                      -- Phase 03 (NEW)
    filesystem.go             -- Read, write, search files
    terminal.go               -- Command execution with approval
    research.go               -- Search grounding orchestration
    drive.go                  -- Drive import/export bridge
    gmail.go                  -- Gmail import bridge
  workspace/                  -- Phase 03 (NEW)
    workspace.go              -- Workspace model, project detection
    memory.go                 -- Session/project/global memory tiers
frontend/
  src/
    main.js                   -- Wails frontend entry
    App.svelte (or .html)     -- Two-pane layout
    components/               -- Chat, artifacts, controls
  wails.json                  -- Wails configuration
```
