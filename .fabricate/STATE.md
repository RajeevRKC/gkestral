---
project: Gkestral
current_milestone: v0.1.0
current_phase: 01
status: planning
last_updated: 2026-03-22
---

# Gkestral State

## Current Position

**Milestone:** v0.1.0 -- The Foundation
**Phase:** 01 -- Gemini Intelligence Core (PLANNING)
**Blocker:** None

## Decisions

| Date | Decision | Rationale |
|------|----------|-----------|
| 2026-03-22 | Language: Go | Commander preference, proven track record, Google alignment |
| 2026-03-22 | UI: Wails v2 (Path 3) | Custom renderer > xterm.js for zero-flicker. Commander approved demo. |
| 2026-03-22 | Name: Gkestral | "G" for Gemini + kestrel. Unique, pronounceable. |
| 2026-03-22 | Tier: T1 | Ship-grade. Targeting Google showcase. |
| 2026-03-22 | Paradigm: Architect & Surgeon | 1M input (read everything) + iterative 8K output (edit surgically) |
| 2026-03-22 | Model default: gemini-2.5-flash | Best cost/performance. Pro for heavy reasoning. Flash Lite for scanning. |
| 2026-03-22 | Phase 1 focus: Gemini intelligence, not UI | Commander: "understanding the nuances of Gemini and maximising productivity is crucial" |

## Research Completed

- [x] 34-tool CLI landscape survey (Claude)
- [x] 10-module architecture analysis (Codex GPT-5.4)
- [x] Strategic gap analysis with Aletheia (Gemini Deep Think)
- [x] Gemini API deep-dive (12 areas, all endpoints documented)
- [x] Autoresearch/self-improvement patterns (7 vectors)
- [x] Gemini CLI source code study (architecture, tools, weaknesses)
- [x] POC built: Go binary (9.5MB), embedded frontend, real API streaming

## POC Location

`${WORKSPACE_ROOT}/output/kestrel/` -- working prototype with:
- Go binary, embedded HTML, WebSocket, Gemini API streaming
- 5 tools: read_file, list_dir, search_files, write_file, run_command
- To be migrated into Wails project structure in Phase 03

## Key Intelligence (Reference)

### Gemini API Critical Findings
1. **Context caching**: 90% discount on 2.5+ models. Minimum 1024 tokens (Flash), 4096 (Pro). Prefix-only.
2. **Thought signatures**: MANDATORY for Gemini 3.x. Must circulate back in all subsequent turns.
3. **Parallel function calling**: Supported. Match results by `id` field.
4. **Search + function calling**: Only combinable on Gemini 3.x with `VALIDATED` mode + `includeServerSideToolInvocations`.
5. **Temperature**: Keep at 1.0 for Gemini 3. Lower values cause looping.
6. **Observation masking**: JetBrains research shows 50%+ cost reduction with equivalent solve rates.
7. **Compression**: Factory.ai's anchored iterative summarisation beats LLM summarisation.
8. **Self-improvement**: SICA (ICLR 2025) achieved 17% -> 53% on SWE-bench via self-editing scaffolding.
