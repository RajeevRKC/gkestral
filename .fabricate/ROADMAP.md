---
project: Gkestral
milestone: v0.1.0 -- v1.0.0
created: 2026-03-22
---

# Gkestral Roadmap

## Milestone 1: v0.1.0 -- The Foundation (Current)

> Single Go binary, working Wails app, real Gemini streaming, core tools, two-pane UI.

### Phase 01: Gemini Intelligence Core
**Focus:** Get the Gemini API integration RIGHT. Context caching, streaming, thinking, thought signatures, model routing. This is the entire product -- everything else is plumbing.

- [ ] Gemini REST client with SSE streaming (handle partial chunks, reconnect)
- [ ] Thought signature handling (mandatory for Gemini 3.x)
- [ ] Thinking mode with configurable budget/level per model family
- [ ] Context caching: stable/active split strategy with TTL management
- [ ] Model routing: Flash Lite (scan) -> Flash (code) -> Pro (architecture)
- [ ] Token counting and budget management
- [ ] Safety settings optimised for coding (no false blocks)
- [ ] System prompt engineering optimised for Gemini's behaviour
- [ ] Parallel function calling with id-based response matching
- [ ] Error handling: 429 retry, model fallback, stream recovery

### Phase 02: Tool Architecture
**Focus:** Build tools that maximise Gemini's agentic performance. Fuzzy editing, smart search, context-aware file reading.

- [ ] read_file with line ranges and token estimation
- [ ] write_file with diff confirmation
- [ ] edit with cascade: exact -> flexible -> fuzzy (Levenshtein) -> LLM-assisted
- [ ] list_dir with .gitignore/.gkestralignore respect
- [ ] search (ripgrep with fallback)
- [ ] run_command with approval flow
- [ ] git_status / git_diff
- [ ] think (force Deep Think mode for a query)
- [ ] Tool result compression (intelligent, not blunt truncation)
- [ ] Tool metrics collection (call count, success rate, duration)

### Phase 03: Wails Desktop App
**Focus:** Package the Path 3 UI in a proper Wails application. Cross-platform build.

- [ ] Wails v2 project scaffold (Go backend + web frontend)
- [ ] Terminal-inspired conversation pane (custom renderer, not xterm.js)
- [ ] Canvas pane: code viewer with syntax highlighting
- [ ] Canvas pane: diff viewer (side-by-side)
- [ ] Canvas pane: markdown renderer
- [ ] Canvas pane: think trace viewer (collapsible tree)
- [ ] WebSocket bridge between Go agent and frontend
- [ ] Resizable panes, tab system, status bar
- [ ] Cross-platform build: Windows (WebView2), Ubuntu (WebKitGTK), macOS (WebKit)
- [ ] Streaming text display with batched rendering (no flicker)
- [ ] Input with command history, Esc to cancel

### Phase 04: Context Management
**Focus:** Maximise Gemini's 1M context window efficiently. KESTRAL.md discovery, observation masking, smart context loading.

- [ ] KESTRAL.md hierarchical discovery (global -> project -> subdirectory)
- [ ] JIT context loading: lazy-load subdirectory context on tool access
- [ ] Observation masking: keep last N turns verbatim, mask older tool outputs
- [ ] .gkestralignore file parsing (gitignore syntax)
- [ ] Conversation history with curated view (strip invalid responses)
- [ ] Token budget visualisation in canvas pane
- [ ] Context cache status display (cached tokens, TTL, cost savings)

---

## Milestone 2: v0.2.0 -- The Differentiator

> Multimodal canvas, Google Search grounding, session persistence, eval suite.

### Phase 05: Multimodal Canvas
- [ ] Image viewer (Gemini Imagen output rendering)
- [ ] Map renderer (MapLibre GL for GIS coordinates)
- [ ] Mermaid diagram rendering
- [ ] Google Search grounding results as research cards
- [ ] Stitch AI output preview (HTML/CSS rendering)
- [ ] Screenshot input (paste image -> Gemini analyses)

### Phase 06: Google Search Grounding + Dual Agent
- [ ] Planner agent (Search grounding enabled) for research
- [ ] Actor agent (function calling) for code modification
- [ ] Agent routing: detect research vs coding intent
- [ ] Search grounding metadata display in canvas
- [ ] URL context tool for documentation fetching
- [ ] Source attribution in research cards

### Phase 07: Session Persistence & Memory
- [ ] SQLite session storage (modernc.org/sqlite, zero CGO)
- [ ] Session resume (`gkestral resume` command)
- [ ] Session-end reflection (Reflexion pattern: what worked, what didn't)
- [ ] Project memory extraction (patterns discovered, saved to KESTRAL.md)
- [ ] Three-tier memory: session (ephemeral) -> project (persistent) -> global (cross-project)

### Phase 08: Eval Suite & Benchmarking
- [ ] Eval framework in Go (EvalCase struct: setup, prompt, assert, budget)
- [ ] 10 benchmark cases: read/explain, find-bug, add-feature, refactor, multi-file
- [ ] `gkestral bench` command (cold start, binary size, token efficiency, solve rate)
- [ ] Regression detection: track zero-regression rate
- [ ] Metric dashboard in canvas pane
- [ ] Comparison mode: benchmark against previous versions

---

## Milestone 3: v0.3.0 -- Self-Improvement

> Autoresearch loop, prompt self-tuning, skill library, MCP client.

### Phase 09: MCP Client
- [ ] MCP client (stdio + HTTP transports)
- [ ] Tool discovery from MCP servers
- [ ] MCP tool execution with permission checks
- [ ] Configuration via KESTRAL.md or config file

### Phase 10: Autoresearch & Self-Improvement
- [ ] Tool call pattern analysis (identify bottlenecks, failures)
- [ ] Hypothesis generation (via Gemini: "how to improve tool X")
- [ ] Auto-improvement loop: modify tool -> eval -> accept/reject
- [ ] OPRO-style system prompt self-tuning
- [ ] Skill library: reusable tool call patterns from successful sessions
- [ ] Improvement archive (SICA pattern: store every iteration with metrics)

### Phase 11: Permission & Security
- [ ] Policy engine with TOML rules (system/user/workspace levels)
- [ ] Path validation (prevent operations outside workspace)
- [ ] Shell command approval with "always allow" patterns
- [ ] Sandbox exploration: Windows AppContainer, Linux seccomp
- [ ] Destructive operation detection and confirmation

---

## Milestone 4: v1.0.0 -- Ship Grade

> Distribution, documentation, community, Google showcase readiness.

### Phase 12: Distribution & Polish
- [ ] Signed binary distribution (Homebrew, Winget, apt/yum)
- [ ] GitHub Releases with cross-platform builds
- [ ] Documentation site
- [ ] Getting started guide
- [ ] Configuration reference
- [ ] Contributing guide

### Phase 13: Community & Showcase
- [ ] Open-source release (Apache 2.0)
- [ ] Benchmark comparison page (vs Gemini CLI, Aider, Claude Code)
- [ ] Demo videos / GIFs
- [ ] Google DevRel outreach materials
- [ ] Blog post: "Building a Gemini-native CLI in Go"
