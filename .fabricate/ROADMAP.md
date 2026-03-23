---
project: Gkestral
milestone: v0.1.0
created: 2026-03-22
updated: 2026-03-23
structure: Commander-directed (REQUIREMENTS.md Section 4)
---

# Gkestral Roadmap

> "Understanding Gemini models, Google APIs -- in depth -- and strategy to go
> beyond existing tools in harnessing Gemini to perfection." -- Commander

## Milestone 1: v0.1.0 -- The Sharp Core

> Gemini-native desktop workbench. Information in, finished work out.
> Hero demo: Drop PDF, research, produce grounded DOCX output.

### Phase 01: Gemini Mastery

**Goal:** Know the engine deeper than anyone. This is the competitive moat.

**Target Models (current generation only):**
- `gemini-3.1-pro-preview` -- deep reasoning, architecture, research synthesis
- `gemini-3.1-flash` -- fast coding, information extraction, high-volume tasks
- `gemini-3.1-flash-image-preview` (NanoBanana Pro) -- image generation
- `gemini-2.5-pro` / `gemini-2.5-flash` -- production stable, context caching

**Deep-dive areas:**
- [ ] Streaming (SSE) -- partial chunks, reconnection, backpressure
- [ ] Thought signatures -- mandatory circulation in 3.x, budget/level control
- [ ] Context caching -- stable/active split, TTL, economics (90% discount), minimum token thresholds
- [ ] Search Grounding -- native API tool, grounding metadata, citation extraction
- [ ] Code Execution -- built-in Python sandbox, when to use vs local execution
- [ ] Multimodal input -- image/PDF/audio ingestion, token costs per modality
- [ ] Function calling -- parallel calls, id-based matching, tool declaration patterns
- [ ] System instruction optimisation -- what Gemini responds to vs ignores
- [ ] Temperature/safety -- 1.0 mandatory for 3.x (lower causes looping), safety setting tuning
- [ ] Token economics -- input/output pricing per model, caching discount tiers
- [ ] Model routing strategy -- when to use Pro vs Flash vs Flash Lite vs NanoBanana
- [ ] Error patterns -- 429 retry, model fallback chains, stream recovery
- [ ] Observation masking -- JetBrains research: 50%+ cost reduction, same solve rate

**Output:** Deep reference document + Go client library that exploits every capability.

---

### Phase 02: Google API & SDK Landscape

**Goal:** Map the full Google API surface for information access. Understand what
a single OAuth token can unlock, what costs what, and how to build progressive
permissioning.

**API deep-dive areas:**
- [ ] Google OAuth 2.0 -- desktop app flow, progressive scopes, token refresh
- [ ] Drive API v3 -- file browse/open/save, folder structure, permissions
- [ ] Gmail API -- read messages, search, import as context, draft creation
- [ ] Google Search (via Gemini Grounding) -- how it works, metadata, citations
- [ ] Google AI SDK (`google-genai` for Go) -- vs raw REST, trade-offs
- [ ] Vertex AI -- when needed vs direct Gemini API, pricing differences
- [ ] Rate limits and quotas per API, per tier (free vs paid Google account)
- [ ] Scope economics -- minimal scopes for v0.1, expansion path for v0.2+

**Skills, Agents & Prompt Engineering:**
- [ ] MCP protocol -- what it offers, when to use vs native Go brokers
- [ ] Agent patterns -- planner/executor, research/action split
- [ ] Prompt engineering for Gemini specifically -- what works, what doesn't
- [ ] System prompt structure -- role, constraints, output format, tool guidance
- [ ] Few-shot vs zero-shot performance on Gemini 3.x models
- [ ] Structured output (JSON mode) -- reliability, schema enforcement
- [ ] Chain-of-thought vs direct answering -- when each is optimal
- [ ] Grounding + function calling combination (3.x VALIDATED mode)

**Output:** API strategy document + OAuth prototype + prompt engineering playbook.

---

### Phase 03: Architecture from Intelligence

**Goal:** Design the architecture AFTER we understand the engine. Not before.
Every design decision grounded in Phase 01-02 findings.

- [ ] System architecture informed by Gemini capability map
- [ ] Wails v2 scaffold with two-pane UI (leveraging existing POC)
- [ ] Go backend structure: engine, tools, services, workspace
- [ ] Context management strategy (based on caching findings)
- [ ] Model routing implementation (based on model comparison findings)
- [ ] Google service integration layer (based on API findings)
- [ ] Session persistence and workspace model
- [ ] Tool architecture (file ops, search, command execution -- invisible)

**Output:** Working architecture with Gemini streaming, OAuth, and session persistence.

---

### Phase 04: First Working Surface

**Goal:** The v0.1 that proves the thesis. Information in, finished work out.

- [ ] Information ingestion -- PDF, web pages, Drive files, Gmail, local files
- [ ] Research cards with citations in right pane
- [ ] Document generation -- real DOCX to local workspace
- [ ] Presentation generation -- real PPTX to local workspace
- [ ] Artifact preview in right pane
- [ ] Human-in-the-loop approval before file writes
- [ ] The hero demo works end-to-end

**Output:** Shippable v0.1 binary that a non-developer can use to produce real work.

---

## Milestone 2: v0.2.0 -- Workspace Expansion (Future)

> Richer multimodal artifact handling, diagram and image pipelines, better memory
> and project context discovery, stronger task planning, first Google ecosystem bridges.

Phases TBD after v0.1 is proven.

## Milestone 3: v0.5.0 -- Adaptive Intelligence (Future)

> Tool metrics, eval suite, reflection loops, prompt tuning based on measured outcomes.

## Milestone 4: v1.0.0 -- Gemini-Native Category Leader (Future)

> Polished multimodal workspace, stable Google ecosystem integrations,
> benchmarkable gains over default Gemini tooling, open-source community-ready release.

---

## Decisions (Settled)

| # | Decision | Resolution | Date |
|---|----------|------------|------|
| 1 | First wedge user | Technical founder (dogfood first) | 2026-03-23 |
| 2 | UI mode | Single surface, code hidden unless asked | 2026-03-23 |
| 3 | Hero demo | PDF drop + research + DOCX output | 2026-03-23 |
| 4 | Phase structure | Commander-directed: Mastery -> APIs -> Architecture -> Surface | 2026-03-23 |
