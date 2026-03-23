---
phase: 1
title: Gemini Mastery
created: 2026-03-23
---

# Phase 01: Gemini Mastery -- Context

## Phase Goal

Know the Gemini engine deeper than anyone. Build a Go client library that
exploits every Gemini API capability: streaming, thought signatures, context
caching, function calling, search grounding, model routing, and error recovery.

This is the competitive moat. Everything else in gKestral builds on top of
this foundation.

## Why This Phase First

Commander's directive: "Understanding Gemini models, Google APIs -- in depth --
and strategy to go beyond existing tools in harnessing Gemini to perfection."

The old roadmap jumped to tool architecture and UI. Commander redirected:
understand the engine BEFORE designing around it.

## Key Intelligence Sources

- `.fabricate/references/study/CONSOLIDATED-FINDINGS.md` -- adopt/avoid/build matrix
- `.fabricate/references/study/gemini-cli/` -- Gemini CLI source (98k stars)
- `.fabricate/references/study/opencode/` -- OpenCode source (120k stars, Go)
- `.fabricate/references/gemini-api-deepdive.md` -- 12-area API capability study
- `.fabricate/STATE.md` -- 8 critical findings from research phase

## Technical Decisions (from research)

1. **Raw REST + SSE**, not Go SDK -- avoid Gemini CLI's forking problem
2. **OpenCode's Provider interface** -- clean Go abstraction
3. **Channel-based streaming** -- `<-chan ProviderEvent`
4. **Exponential backoff + user callback** -- `onPersistent429()` pattern
5. **Thought signatures** -- SYNTHETIC_THOUGHT_SIGNATURE validation
6. **Temperature 1.0** -- mandatory for 3.x models (lower causes looping)

## Deliverables

1. Deep reference document covering all 13 API areas
2. Go client library (`internal/gemini/`) with full test suite
3. Benchmark: token economics across all target models
4. Model routing strategy document

## Constraints

- Go 1.23+, no CGO dependencies
- T1 quality: tests alongside code, lint passes
- Single `internal/gemini/` package, clean interface boundaries
- No UI work in this phase (Phase 03)
- No Google OAuth in this phase (Phase 02)

## Dependencies

- Gemini API key (env: `GEMINI_API_KEY`)
- Go 1.23+ installed
- No prior phases -- this is the foundation
