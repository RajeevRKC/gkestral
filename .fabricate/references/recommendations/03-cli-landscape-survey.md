# Open-Source Coding CLI Landscape Survey

> Source: 34-tool survey across all major agentic coding CLIs (2026-03-22)

---

## Tier 1: Major Players (10k+ Stars)

| Tool | Lang | Stars | Models | MCP | Lean? |
|------|------|-------|--------|-----|-------|
| **OpenCode** | Go (+Bun) | 122k | 75+ providers | Yes | Medium (Go + Bun) |
| **Gemini CLI** | TypeScript | 98k | Gemini only | Yes | No (~150MB, Node.js) |
| **Codex CLI** | Rust | 66k | OpenAI | Yes | Yes (native binary) |
| **Open Interpreter** | Python | 63k | Multi | No | No (Python) |
| **Aider** | Python | 42k | 100+ models | No | No (Python) |
| **Goose** | Rust | 33k | Model-agnostic | Yes (core) | Medium (Rust + Electron) |
| **Continue CLI** | TypeScript | 32k | Any provider | Yes | No (Node.js) |
| **Pi (oh-my-pi)** | TS + Rust | 25k | Multi | Yes | Medium |
| **Crush** | Go | 21.6k | Multi | Yes | Yes (single Go binary) |
| **Fabric** | Go | ~24k | 20+ providers | No | Yes (single Go binary) |

## Go Implementations (Deep Dive)

| Tool | Stars | Key Lesson |
|------|-------|-----------|
| **OpenCode** | 122k | Gold standard Go CLI agent. Bubble Tea TUI, client/server, SQLite, LSP |
| **Crush** | 21.6k | Charmbracelet stack. Clean MCP, polished TUI |
| **PicoClaw** | 25.3k | Ultra-lean: ~7500 LOC, <10MB RAM |
| **Fabric** | 24k | Novel pattern-based architecture (200+ prompt patterns) |
| **mods** | ~4k | Cleanest pipe-friendly design |

## Truly Lean Tools

| Tool | Lang | Binary | RAM | Boot |
|------|------|--------|-----|------|
| NullClaw | Zig | 678KB | 1MB | 2ms |
| ZeroClaw | Rust | <5MB | <5MB | <10ms |
| PicoClaw | Go | ~8MB | <10MB | <1s |
| aichat | Rust | ~10MB | Low | Fast |
| Crush | Go | ~15MB | Low | Fast |

## Architecture Patterns

1. **Agent Loop + Tool Harness** -- ReAct pattern. Dominant. All serious tools use this.
2. **Client/Server Split** -- OpenCode, Goose. Enables remote operation, multiple clients.
3. **Single-Binary CLI** -- Codex, Crush, Fabric. Go/Rust. Best distribution story.
4. **Plugin/Extension** -- llm, gptme. Core thin, capabilities via plugins.
5. **Context Engineering** -- Aider (repo map), OpenCode (LSP), Plandex (2M context).
6. **Diff/Sandbox Model** -- Aider (git commits), Plandex (diff sandbox), Codex (seccomp).

## Gemini Native Support (Out of Box)

OpenCode, aichat, Aider, Goose, Crush, Fabric, mods, gptme, ForgeCode, PicoClaw, IronClaw, Kode CLI

## MCP Support

Claude Code, Gemini CLI, Codex CLI, OpenCode, Goose, Crush, mods, gptme, Continue CLI, Pi, Kode CLI

---

## Key Takeaway for Gkestral

The landscape is crowded for "generic coding CLIs." Gkestral's differentiation MUST come from Gemini-native capabilities (caching, grounding, multimodal), not from being yet another agent loop with file tools. The execution layer should match the best tools; the intelligence and research layers should exceed them.
