---
project: Gkestral
current_milestone: v0.1.0
current_phase: 3
status: planning
last_updated: "2026-03-25T18:00:00.000Z"
current_plan: 1
plan_status: ready
state_version: 11
current_git_hash: 0762913
decisions: 7
last_decision: "2026-03-24T12:00:00.000Z"
---

## Position

- **Phase:** 3
- **Plan:** 1 (03-01-PLAN.md)
- **Status:** ready for execution
- **Stage:** EXECUTE

## Current Position

**Milestone:** v0.1.0 -- The Sharp Core
**Phase:** 03 -- Architecture from Intelligence (PLAN READY)

## Phase 02 Closure

- **Closed:** 2026-03-25
- **Summary:** `.fabricate/phases/02-google-api-landscape/02-01-SUMMARY.md`
- **Result:** 14/14 tasks, 161 tests, 3-way AR-3 handshake (25 findings fixed)
- **Deferred:** Upload OOM/transport bypass, integration test assertions, PBKDF2 iterations
**Blocker:** None

## Phase 01 Closure

- **Closed:** 2026-03-24
- **Summary:** `.fabricate/phases/01-gemini-mastery/01-01-SUMMARY.md`
- **Result:** 16/16 tasks, 143 tests, 86.1% coverage, 3-way handshake complete
- **Deferred:** List pagination, DynamicRetrievalConfig, -race CI, Codex final round

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

### 2026-03-24T12:00:00.000Z

**Decision:** Phase 01 closed. All 16 tasks complete, 14/14 ACs PASS, 1 N/A (race detection -- CGO), 86.1% coverage. Three-way handshake findings resolved. Moving to Phase 02.

**Rationale:** All quality gates met for T1 closure. Deferred items logged in SUMMARY.md and ISSUES.md.

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
