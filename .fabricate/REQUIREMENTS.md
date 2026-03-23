---
project: Gkestral
version: 1.0
created: 2026-03-23
sources: Mandan (cross-doc analysis), Codex GPT-5.4 (product strategy), Gemini 3.1 Pro Deep Think (7-pass Aletheia)
status: DRAFT -- awaiting Commander approval
---

# gKestral -- Crystallized Requirements

> Three AI models analysed 11 documents, surfaced 6 contradictions and 8 open
> questions, then converged on this requirements baseline. Commander decides.

---

## 1. Product Identity (SETTLED)

**gKestral is a new-era office tool.** Not a coding CLI. Not a chatbot. Not an
autonomous agent. A desktop workbench where AI creates and the user directs.

| Attribute | Definition |
|-----------|-----------|
| **Tagline** | "For those who want control." |
| **Subtitle** | "AI designed to work for you. Your office in a terminal." |
| **Identity** | Information-first desktop workbench |
| **Engine** | Gemini-native (not model-agnostic) |
| **Architecture** | Two-pane: chat/direction left, artifacts right |
| **Outputs** | DOCX, PPTX, research briefs, images, code (when needed) |
| **Philosophy** | People tool. Human directs, AI produces, human refines. |
| **Stack** | Go 1.23+ / Wails v2 / Raw REST + SSE / Single binary |
| **Tier** | T1 ship-grade |
| **Brand** | gKestral (lowercase g, capital K). Domain: gkestral.com |

---

## 2. Resolved Contradictions

All three models agreed on resolutions. No dissent.

| # | Contradiction | Resolution |
|---|--------------|------------|
| 1 | Google integrations deferred to v0.2 | **Move to v0.1.** Drive, Gmail, Search are day-one identity. |
| 2 | DOCX/PPTX generation missing from roadmap | **Add to v0.1.** Primary product output. Not optional. |
| 3 | Execution Core weighted Heavy in PROJECT.md | **Reclassify to Light.** Reliable and invisible. |
| 4 | PROJECT.md description still says "coding" | **Rewrite.** New description below. |
| 5 | Phase 01-04 assume coding-CLI-first | **Restructure roadmap.** New phases below. |
| 6 | Website says "terminal" as identity | **Keep for now.** Commander chose this language deliberately. |

**New PROJECT.md description:**
> "Gemini-native desktop workbench for turning scattered information into
> finished documents, presentations, and research-backed outputs."

---

## 3. Consensus: What v0.1 Must Contain

All three models converged independently on these v0.1 requirements:

### Must Have (v0.1)
- [ ] Single Google OAuth (progressive permissioning)
- [ ] Gemini streaming with context caching and thought signatures
- [ ] Two-pane Wails desktop app (chat left, artifacts right)
- [ ] Google Search Grounding (native Gemini API, same key)
- [ ] Google Drive: open/save/browse files
- [ ] Gmail: read/import emails as context
- [ ] Document generation: real DOCX files to local workspace
- [ ] Document generation: real PPTX files to local workspace
- [ ] PDF/web page ingestion as context
- [ ] Research cards with citations in right pane
- [ ] Session persistence (local disk, folders are the workspace)
- [ ] Artifact preview in right pane (HTML/CSS render)
- [ ] Human-in-the-loop: user approves before file writes
- [ ] Local file watching and workspace attachment

### Should Have (v0.1 if capacity allows)
- [ ] Presentation generation: real PPTX with templates/themes
- [ ] Subscription tier detection and feature gating
- [ ] Token/usage tracking and budget display
- [ ] Model routing (Flash for speed, Pro for depth)

### Defer (v0.2+)
- [ ] Maps integration
- [ ] MCP client
- [ ] Stitch integration
- [ ] Eval suite / benchmarks
- [ ] Autoresearch / self-improvement
- [ ] Calendar integration
- [ ] NotebookLM bridge

### Keep from Old Roadmap (as invisible infrastructure)
- [ ] File read/write tools (behind the scenes)
- [ ] Search/grep tools (behind the scenes)
- [ ] Command execution with approval flow
- [ ] Git operations (available but not primary UI)
- [ ] Context management (observation masking, token budgets)

### Drop
- [ ] Code diff viewer as primary artifact type
- [ ] Terminal emulation as product identity
- [ ] Ripgrep-led UX
- [ ] Fully autonomous agentic loops

---

## 4. Revised v0.1 Phase Structure (Commander-Directed)

> "Understanding Gemini models, Google APIs -- in depth -- and strategy to go
> beyond existing tools in harnessing Gemini to perfection." -- Commander

### Phase 01: Gemini Mastery

**Goal:** Know the engine deeper than anyone. This is the competitive moat.

**Target Models (current generation only, no legacy):**
- `gemini-3.1-pro-preview` -- deep reasoning, architecture, research synthesis
- `gemini-3.1-flash` -- fast coding, information extraction, high-volume tasks
- `gemini-3.1-flash-image-preview` (Nano Banana Pro) -- image generation
- `gemini-2.5-pro` / `gemini-2.5-flash` -- production stable, context caching
- Any new models announced during development

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

## 5. Decisions Needed From Commander

### Decision 1: First Wedge User

| Option | Proposed By | Argument |
|--------|------------|----------|
| **A. Student/Researcher** | Codex | Matches hero use case, needs Google, cares about deliverables |
| **B. Technical Founder/TPM** | Gemini | Lives at intersection of information + code, Commander IS this user, validates business model |
| **C. Mixed power user** | (neither recommended this) | Broader but harder to focus |

### Decision 2: Dual-Mode (Office/Dev) or Single Surface?

| Option | Proposed By | Argument |
|--------|------------|----------|
| **A. Explicit Office/Dev toggle** | Gemini | Clear separation, coding doesn't pollute information UX |
| **B. Single surface, code hidden** | Codex | Simpler, code surfaces only when user asks for it |

### Decision 3: Hero Demo Scenario for v0.1

| Option | Proposed By | Scenario |
|--------|------------|----------|
| **A. School presentation** | Commander (vision) | "I need a presentation for school tomorrow" -> research + PPTX |
| **B. Product launch brief** | Gemini | PRD from Drive + email thread + web research -> DOCX + PPTX -> pivot to code |
| **C. PDF drop + research** | Mandan | Drop PDF, ask question, get grounded answer, save DOCX |

### Decision 4: v0.1 Phase Structure

| Option | Proposed By |
|--------|------------|
| **A. Commander-directed (RECOMMENDED)** | Gemini Mastery -> Google APIs -> Architecture from Intelligence -> First Working Surface |
| **B. Codex's structure** | Identity -> Research -> Documents -> Refinement |
| **C. Gemini's structure** | Foundation -> Engine -> Dual-Mode -> Integration |

---

## 6. Risk Register

| # | Risk | Severity | Mitigation |
|---|------|----------|------------|
| 1 | Google OAuth verification delays | HIGH | Start verification process immediately, progressive scopes |
| 2 | DOCX/PPTX generation quality | HIGH | Use proven Go libraries (unioffice), invest in templates |
| 3 | Context window overflow | HIGH | .gkestralignore, token-aware chunking, observation masking |
| 4 | Vision too broad for v0.1 | MEDIUM | Lock first wedge user, build for that persona first |
| 5 | UI identity crisis (office vs terminal) | MEDIUM | Clear visual partition between modes |
| 6 | Approval fatigue (too many confirmations) | MEDIUM | Tiered: low-risk auto-proceed, high-risk requires confirm |
| 7 | Subscription tier mechanics undefined | MEDIUM | Defer gating to v0.2, build tracking in v0.1 |

---

## 7. What All Three Models Agree On

This is the bedrock. No dissent on any of these:

1. **The roadmap must be rewritten.** Current phases are for a different product.
2. **Google services are day-one.** Not v0.2. Not nice-to-have.
3. **DOCX/PPTX generation is the product demo.** Not markdown. Not code diffs.
4. **Code is kept but invisible.** It runs behind the scenes. User never sees it unless they choose to.
5. **Two-pane UI is correct.** Chat left, artifacts right.
6. **MCP is deferred.** Go-native brokers for Drive/Gmail in v0.1.
7. **Maps deferred to v0.2.**
8. **Local-first.** Sessions persist as folders on the user's machine.
9. **Human-in-the-loop.** User approves file writes, email sends, code execution.
10. **Single binary, no runtime dependencies.**

---

*Generated by MasterMindMandan with input from Codex GPT-5.4 and Gemini 3.1 Pro (Aletheia GVR, 7-pass).*
*2026-03-23 | Status: DRAFT -- awaiting Commander decisions on 4 open items.*
