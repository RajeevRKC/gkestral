---
project: Gkestral
current_milestone: v0.1.0
current_phase: 1
status: planning
last_updated: "2026-03-24T08:53:22.082Z"
current_plan: 1
plan_status: complete
state_version: 6
current_git_hash: 406e588
decisions: 6
last_decision: "2026-03-23T17:40:03.539Z"
---

## Position

- **Phase:** 1
- **Plan:** 1
- **Status:** complete
- **Stage:** PLAN

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

## Decision Log

### 2026-03-23T17:40:03.539Z

**Decision:** AR-1 handshake complete: Gemini APPROVED WITH WARNINGS (r1), Codex APPROVED (r3). 3/5 rounds used. Key fixes: structured output added, CacheManager API pattern, retry integrated into client, wave dependencies corrected.

**Rationale:** 


### 2026-03-23T17:15:59.837Z

**Decision:** Roadmap replaced: 13-phase coding-CLI -> 4-phase Commander-directed structure

**Rationale:** 


### 2026-03-23T17:15:59.757Z

**Decision:** Phase structure: Gemini Mastery -> Google APIs -> Architecture -> Working Surface

**Rationale:** 


### 2026-03-23T17:15:59.670Z

**Decision:** Hero demo: PDF drop + research + DOCX output

**Rationale:** 


### 2026-03-23T17:15:59.591Z

**Decision:** UI mode: Single surface, code hidden unless asked

**Rationale:** 


### 2026-03-23T17:15:51.382Z

**Decision:** Wedge user: Technical founder (dogfood first)

**Rationale:** 
