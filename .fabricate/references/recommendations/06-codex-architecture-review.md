# Codex CLI (GPT-5.4) Architecture Analysis

> Source: Codex handshake review, locked to gpt-5.4 (2026-03-22)

---

## Key Findings

### 1. Language Verdict: Rust (Overruled by Commander -> Go)
Codex argued sub-5MB binary and sub-100ms startup constraints favor Rust. Commander chose Go based on track record. Decision stands.

### 2. Context Window Correction
Current Gemini 2.5/3 models: 1,048,576 tokens (1M), not 2M. Design for 1M now, 2M-ready later.

### 3. Architecture: Single-Process Agent Loop
```
UI -> Agent Runtime -> Tool Router -> Model Client -> Local State
```
One bounded loop: inspect, plan, tool-call, patch, validate, summarize. Not a server-like async stack.

### 4. Ten Minimum Viable Tools
1. list_dir
2. glob/search
3. read_file (range)
4. write_file
5. apply_patch
6. run_shell
7. git_status/diff
8. attach_image
9. mcp_call
10. think (Deep Think)

### 5. Gemini Integration Strategy
- Raw `generateContent` + SSE streaming
- Model routing by task complexity (flash-lite scanning, flash coding, pro architecture)
- Thinking budgets as first-class knob
- Context caching for stable prefixes

### 6. Ten-Module Architecture
```
ui, config, agent, promptpack, repo, tools_local, tools_mcp, gemini, session, policy
```

### 7. Long Context Strategy
- Never load entire repo into RAM
- Symbol graph + file summaries as default memory
- Tiered prompt assembly
- Use context window as budget ceiling, not fill target
- Push stable context into Gemini cache

### 8. MCP: Essential for v1
Thin -- stdio transport first, Streamable HTTP second. Tools only (defer resources/prompts).

### 9. What to Steal from Other Tools
- Permission modes from Claude Code
- Repo map from Aider
- Lazy tool activation from Goose
- CLI/core layering from Gemini CLI

### 10. What to Exclude
- No IDE companion
- No browser automation
- No cloud sync/telemetry
- No multi-agent swarm
- No embedded vector DB
- No AST packs for 20 languages
- No slash-command taxonomy

### 11. What Impresses Google
Benchmark publicly against their Node CLI on: cold start, binary size, token efficiency, large-repo bugfix success. Position as "the fastest serious Gemini coding agent."

---

## Codex Offered Follow-Up
Concrete v1 spec (command surface, config shape, approval model, 6-week plan) available on request.
