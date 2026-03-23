---
name: Gkestral
tier: T1
status: active
created: 2026-03-22
owner: Commander
language: Go
description: Gemini-native multimodal research and execution workspace for coding, design, and grounded problem solving
tags: [gemini, cli, go, agentic, multimodal, research, grounding]
---

# Gkestral -- Gemini-Native Research and Execution Workspace

> "Perfect understanding. Precise execution."

## Thesis

Gkestral is not a generic coding CLI with Gemini support bolted on.

It is a Gemini-native workspace built around what Gemini does unusually well:
- multimodal understanding
- long-context synthesis
- grounded research
- high-bandwidth context ingestion
- integration potential with Google's wider AI ecosystem

Coding is a major output mode, but not the only one. Gkestral should be equally comfortable turning research into code, diagrams, design artifacts, implementation plans, or multimodal work products.

The goal is to become the sharpest tool for users who want Gemini's strengths without the bloat, instability, and uneven UX of existing Gemini-native tools.

## Why This Should Exist

Claude Code is strong because it is disciplined, lean, and highly focused on execution.
Codex CLI is strong because it is direct, compact, and pragmatic.

Gemini's opportunity is different.

Gemini is strongest when work requires:
- very large context windows
- multimodal inputs and outputs
- live research and grounding
- synthesis across documents, images, code, diagrams, and external sources

Current Gemini tooling does not fully convert those strengths into a best-in-class product experience. Gkestral exists to fix that.

## Product Position

Gkestral is a Gemini-first research and execution workspace.

It should feel like:
- the discipline of Claude Code
- the leanness of Codex CLI
- the openness of Aider
- but architected specifically for Gemini's long-context, grounded, and multimodal workflows

The product is not trying to beat every coding tool at pure terminal coding minimalism.
It is trying to win the category where research, multimodal context, and execution need to happen in one coherent loop.

## Core Product Principle

Do not clone Claude Code's center of gravity.

Instead:
- match the reliability and sharpness of the best coding agents
- exceed them on multimodal and research workflows
- integrate directly with Gemini-native capabilities and Google-adjacent tools

In short:
- Claude-style execution discipline
- Gemini-native comprehension, grounding, and multimodal breadth

## The "Architect & Surgeon" Paradigm

Gemini has massive input capacity but bounded output. Gkestral should exploit that asymmetry deliberately.

- **Architect**: absorb the full working picture -- codebase, docs, screenshots, diagrams, notes, maps, references, and research.
- **Surgeon**: act with precise, iterative edits and tool calls, one bounded change at a time.

This is the core operating model:
- broad understanding
- narrow execution
- continuous grounding

## Primary Differentiators

| Capability | Why It Matters |
|-----------|----------------|
| **Gemini-Native Context Engine** | Built for very large context ingestion, repository-scale understanding, and cache-aware prompt construction |
| **Visible Context Caching** | Stable/active context split, cache hit visibility, and measurable token savings |
| **Grounded Research Loop** | Google Search grounding surfaced as first-class research cards, not hidden behind chat text |
| **Multimodal Workspace** | Images, diagrams, maps, previews, and rich artifacts live beside execution, not outside the tool |
| **Execution Discipline** | Tight file editing, tool invocation, and diff flow inspired by the best coding agents |
| **Google Ecosystem Bridges** | Direct workflows into tools like AI Studio, Stitch, and NotebookLM where they add real leverage |
| **Windows-First Reliability** | Built and tested on Windows without treating it as a second-class environment |
| **Lean Distribution** | Single Go binary with embedded frontend and minimal runtime assumptions |

## What Gkestral Is Not

- not a Claude Code clone with Gemini swapped in
- not a bloated desktop shell around a chat box
- not a research-only notebook
- not a pure TUI for text-only coding

It is a multimodal research-to-execution environment.

## System Architecture

### 1. Execution Core

The execution core is responsible for:
- tool calling
- file reads and writes
- terminal and process orchestration
- planning and task decomposition
- precise edit application
- diff generation and review flow

This layer must be boring, predictable, and extremely reliable.

### 2. Context Engine

The context engine is responsible for:
- repository ingestion
- filtered codebase packing
- large-context construction
- cache-aware context partitioning
- project/session/global memory tiers
- artifact indexing across code, docs, and media

This is the heart of the Gemini-native advantage.

### 3. Research Engine

The research engine is responsible for:
- Google Search grounding
- source collection and ranking
- research card rendering
- citation retention
- NotebookLM-style source bundles
- turning research into execution-ready context

Research should not be a side effect. It should be an explicit part of the workflow.

### 4. Multimodal Workspace

The workspace layer is responsible for:
- image rendering
- diagram previews
- map views
- design and UI previews
- thought/planning artifacts where product-safe
- code diff and rich artifact presentation

The right pane is not decoration. It is a working surface.

### 5. Gemini Integration Layer

The Gemini layer is responsible for:
- raw REST + SSE streaming
- model routing
- context caching
- function/tool calling
- grounding orchestration
- multimodal request packaging
- Gemini-specific protocol details

Avoid unnecessary SDK lock-in.

### 6. Integration Layer

Over time, Gkestral should bridge into high-leverage Gemini-adjacent tools:
- **Google AI Studio** for prompt and model workflow handoff
- **Stitch** for design artifact generation and preview flows
- **NotebookLM** for research pack ingestion and synthesis

These should begin as narrow, useful bridges rather than broad, fragile automations.

## UI Direction

### Left Pane

Terminal-inspired execution stream:
- conversation
- actions
- tool calls
- diffs
- task state
- approvals when needed

Custom DOM rendering is preferred over terminal emulation if it produces smoother streaming and better multimodal composition.

### Right Pane

Artifact and multimodal workspace:
- images
- diagrams
- maps
- research cards
- design previews
- generated files and visual diffs

The UI should make Gemini's multimodal value visible, not abstract.

## Technical Stack

| Component | Technology | Why |
|-----------|-----------|-----|
| Language | Go 1.23+ | Fast startup, single binary distribution, strong concurrency model |
| Desktop | Wails v2 | Native webview approach without Electron overhead |
| Frontend | Custom HTML/CSS/JS | Full control over streaming UX and multimodal rendering |
| API | Raw REST + SSE | Protocol control, reduced dependency surface |
| Storage | SQLite via modernc.org/sqlite | Persistent state without CGO |
| Search | ripgrep | Fast local retrieval, strong fallback for repository search |
| Maps | MapLibre GL | Open-source mapping without mandatory key dependency |
| Diagrams | Mermaid.js | Standard diagram rendering pipeline |

## Competitive Study Areas

Gkestral should study other tools by subsystem, not by brand loyalty.

Questions to answer:
- Who has the best edit loop?
- Who has the best diff UX?
- Who handles context most intelligently?
- Who has the safest tool model?
- Who does model routing well?
- Who makes research usable?
- Who renders artifacts best?

Primary study targets:
- Gemini CLI
- Antigravity
- Claude Code
- Codex CLI
- Aider
- selected strong open-source agentic tools

The goal is to combine the best execution patterns with a Gemini-native architecture.

## Phased Roadmap

### v0.1 -- Sharp Core

Prove the product thesis with a minimal but powerful foundation:
- solid Gemini chat + tool loop
- repository-aware context engine
- visible context caching
- precise file editing and diffs
- grounded web research cards
- Windows-first desktop shell
- one strong multimodal pane

### v0.2 -- Workspace Expansion

- richer multimodal artifact handling
- diagram and image pipelines
- better memory and project context discovery
- stronger task planning for larger work
- first Google ecosystem bridges

### v0.5 -- Adaptive Intelligence

- tool metrics
- eval suite
- reflection loops
- prompt tuning based on measured outcomes

### v1.0 -- Gemini-Native Category Leader

- polished multimodal workspace
- stable Google ecosystem integrations
- benchmarkable gains over default Gemini tooling
- open-source community-ready release

## Success Criteria

Gkestral succeeds if users can say:
- "Gemini finally has a serious power tool."
- "This handles research + code + visuals in one flow."
- "This is more coherent than juggling AI Studio, browser tabs, and terminal tools manually."
- "This feels purpose-built for Gemini instead of adapted from another model ecosystem."

## What Would Impress Google

1. Clear evidence that Gemini's unique strengths are being translated into product value
2. Context caching made visible, measurable, and useful
3. Multimodal workflows that are genuinely better than text-only coding agents
4. Tight execution loop with strong UX and low friction
5. Practical bridges into the broader Google AI ecosystem
6. Open-source readiness and community credibility

## Quality Gate: T1

- full test suite
- eval coverage for core execution patterns
- security review for tool sandboxing and path safety
- Windows + Ubuntu + macOS CI
- documentation worthy of external adoption
- signed binary distribution

## Research Sources (2026-03-22)

Initial multi-model and market research completed across:
- competitive tool landscape
- Gemini API capabilities and constraints
- agent architecture patterns
- long-context and reflection strategies
- multimodal and grounded workflow opportunities

Further work should convert that research into subsystem decisions, benchmarks, and product constraints rather than leaving it as broad inspiration.
