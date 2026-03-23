# Gemini CLI Source Code Architecture Study

> Source: Deep analysis of github.com/google-gemini/gemini-cli (2026-03-22)

---

## 1. Architecture Overview

TypeScript monorepo, 7 packages:
- **core** -- API orchestration, tool execution, prompt construction, context management
- **cli** -- React/Ink TUI, input handling, display
- **sdk** -- Programmatic embedding
- **a2a-server** -- Agent-to-Agent (experimental)
- **devtools** -- Network/Console inspector
- **vscode-ide-companion** -- VS Code extension
- **test-utils** -- Shared test utilities

## 2. Agent Loop

### Entry: `sendMessageStream()` (client.ts:867)
1. Fires BeforeAgent hooks
2. Delegates to `processTurn()`

### Turn: `processTurn()` (client.ts:584)
1. Check session turn count (MAX_TURNS = 100)
2. Attempt chat compression if token budget strained
3. Estimate request tokens vs remaining budget
4. Run loop detection (LoopDetectionService)
5. Route to model via ModelRouterService
6. Create Turn instance, stream response
7. On tool calls: yield ToolCallRequest events
8. On "next speaker = model": recursive call with "Please continue"

### Turn Execution: `Turn.run()` (turn.ts:253)
- Wraps `GeminiChat.sendMessageStream()`
- Processes raw stream into typed events (Content, Thought, ToolCallRequest, Finished, Error)

### Tool Scheduling
CoreToolScheduler: state machine per call (Validating -> Scheduled -> Executing -> Success/Error/Cancelled). MessageBus for confirmation flow.

**Key design:** AsyncGenerator for event emission. Core emits, CLI renders. Clean separation.

## 3. GEMINI.md System

### Three tiers:
- **Global:** `~/.gemini/GEMINI.md`
- **Extension:** From installed extensions
- **Project:** `./GEMINI.md`

Flattened with separators when injected into system prompt.

### JIT (Just-In-Time) Subdirectory Context (jit-context.ts)
When "high-intent" tools (read_file, write_file, edit, list_directory, read_many_files) access a path, `discoverJitContext()` walks up looking for additional GEMINI.md files. Appended to tool output as `--- Newly Discovered Project Context ---`.

### Memory Tool (memoryTool.ts)
Model can persist facts to `~/.gemini/GEMINI.md` via SaveMemory tool. Appended under `## Gemini Added Memories`.

## 4. Built-in Tools

| Tool | Key Detail |
|------|-----------|
| read_file | Line ranges, images/PDFs |
| read_many_files | Batch reading |
| write_file | Full writes with diff confirmation |
| **edit** | **40KB file. Cascade: exact -> flexible -> regex -> fuzzy (Levenshtein 10%) -> LLM-assisted** |
| run_shell_command | Background process support |
| glob | Pattern matching |
| grep | Uses ripgrep under the hood |
| list_directory | BFS, 200 item cap |
| google_web_search | Grounded search via API |
| web_fetch | Fetch and parse pages |
| save_memory | Persist to GEMINI.md |
| ask_user | Request input |
| activate_skill | Load skill into context |

### Edit Tool Sophistication
- Exact match
- Flexible whitespace-normalized matching
- Regex matching
- Fuzzy matching (Levenshtein, 10% weighted threshold)
- Omission placeholder detection ("// ... rest of code")
- **LLM-assisted correction** (FixLLMEditWithInstruction) when all strategies fail
- IDE integration for VS Code diffs

## 5. Context Management

### Directory Structure (getFolderStructure.ts)
- BFS traversal, 200 item hard cap
- Respects .gitignore and .geminiignore
- Ignores node_modules, .git, dist, __pycache__

### Token Budget
- Default: 1,048,576 tokens (1M)
- Thinking budget: 8,192 tokens

### Chat Compression (chatCompressionService.ts)
- Triggers at **50%** of model's token limit
- Preserves last **30%** of history
- Summarizes older portion via Flash (speed)
- Truncates old tool outputs beyond **50,000 token budget** -> saves to temp files

### NO code intelligence
No tree-sitter, no LSP, no semantic understanding. Pure grep + file reads. VS Code companion sends IDE context as JSON.

## 6. Permission Model

### Policy Engine (types.ts)
- ALLOW, DENY, ASK_USER decisions
- Approval modes: DEFAULT, AUTO_EDIT, YOLO, PLAN
- Rules match: toolName, mcpName, argsPattern (regex), toolAnnotations, priority
- TOML policy files at system/user/workspace/extension/runtime levels

### Sandbox
- macOS: sandbox-exec (seatbelt)
- Linux: LXC support
- **Windows: NO sandbox** (PoC in issue #20780)

## 7. What They Do Poorly

### Windows -- Genuinely Broken
- #20968: PowerShell output encoding (BOM corruption)
- #18896: Screen glitching and flickering
- #18899: Auto-update corrupts installation
- #21340: Uses outdated powershell.exe, BOM bricks configs
- #20780: No sandbox at all
- #5305: Meta-issue "Robust Windows Support" -- still open from Dec 2025

### Loop Detection -- Recurring
Multiple duplicates (#23467, #23439, #22136, #17984). LLM-based detector doesn't catch all cases.

### Context Management -- Compression is Lossy
50K budget means large tool outputs truncated. Model can re-read but adds turns.

### No Code Intelligence
No tree-sitter, no LSP. Significant gap vs Claude Code.

### MCP Client -- Massive
74KB file. Suggests significant edge case difficulties.

### Memory -- Append-Only Flat Markdown
No structured storage, no indexing, no retrieval.

## 8. Patterns Worth Adopting

1. **JIT Context Discovery** -- Lazy-load subdirectory context on tool access
2. **Edit Fuzzy Matching Cascade** -- exact -> flexible -> regex -> fuzzy -> LLM
3. **Tool Declarative/Invocation Split** -- Definition (static) vs Invocation (per-call)
4. **Policy Engine with TOML** -- Multi-level priority-based rules
5. **Curated vs Comprehensive History** -- Two views (API vs debug)
6. **Generator-Based Event Stream** -- AsyncGenerator for clean core/UI separation
7. **Loop Detection with LLM** -- Cheap model periodically checks stuck state
8. **Hook System** -- Before/After hooks for Agent, Tool, Model, Compress

## 9. Patterns We Should Do Better

1. **No persona system** -- just "You are Gemini CLI"
2. **Flat GEMINI.md** -- no structured memory, no retrieval
3. **No orchestrator governance** -- flat agent
4. **No project lifecycle** -- basic plan mode only
5. **Windows is broken** -- we can own Windows
6. **No code intelligence** -- opportunity for tree-sitter/LSP
