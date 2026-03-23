# Gkestral -- Gemini-Native Agentic CLI

> Lean Go binary. Two-pane Wails app. Maximise Gemini's unique capabilities.
> Tier: T1 (ship-grade, targeting Google showcase)

## Quick Context

- **Language:** Go 1.23+
- **UI:** Wails v2 (native webview, not Electron)
- **API:** Raw REST + SSE (no Gemini SDK dependency)
- **Default model:** gemini-2.5-flash (routing to Pro/Flash Lite by task)
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
Current: Milestone 1 (v0.1.0), Phase 01 (Gemini Intelligence Core)

## References

Research documents in `.fabricate/references/`:
- `gemini-api-deepdive.md` -- 12-area API capability study
- `autoresearch-patterns.md` -- Self-improvement architecture
- `gemini-cli-source-study.md` -- Official CLI codebase analysis

## Commands

```bash
# Development
wails dev                    # Hot reload development
wails build                  # Production build
go test ./...                # Run tests

# From POC (output/kestrel/)
go run .                     # Run POC prototype
```
