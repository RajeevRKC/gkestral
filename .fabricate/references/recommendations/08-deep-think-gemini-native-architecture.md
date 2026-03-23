# What Makes a Tool "Gemini-Native" vs "Gemini-Compatible"

> Source: Gemini 3.1 Pro Deep Think, Aletheia GVR (2026-03-23)
> 4 passes: Generate 81s + Verify 87s + Revise 71s + Re-Verify 61s | Total: 300s
> Confidence: MEDIUM (MINOR_FIX triggered Revise; all resolved after 1 cycle)

---

## Verdict

The current POC is architecturally a GPT-4 wrapper -- a standard ReAct loop pointing at a Gemini endpoint. The 32k truncation, blind write_file, missing multimodal/caching/grounding, and single-model loop collectively mean zero Gemini-specific capabilities are exploited. Becoming "Gemini-native" requires restructuring around seven differential capabilities, with **Context Caching** as the linchpin.

---

## Seven Dimensions of Gemini-Native Architecture

### 1. Context Window Asymmetry (1M In / 8K Out) [REVISED]

**Core insight:** The 1M input enables holistic reasoning, but ONLY when backed by Context Caching. Without caching, this is a cost/latency catastrophe. The 8K output means write_file must die -- surgical diffs are mandatory.

- Remove 32k truncation in toolReadFile
- Replace write_file with apply_diff / search_and_replace
- Full-context loading gated behind CacheManager being active
- UX shifts from "find and fix" to "you already have everything -- architect the change"
- Current tools use RAG designed for 128-200K limits. RAG destroys holistic understanding that 1M enables.

### 2. Context Caching as Architectural Primitive

**Core insight:** Caching transforms the LLM from stateless API into persistent shared memory at 90% cost reduction. Also enables system prompt caching across turns.

- New cache.go module with fsnotify file watcher for automatic invalidation
- GenerateRequest references cachedContent URI instead of re-sending tokens
- UI displays "Workspace Cache: Warm (450k tokens) -- Cost/query: $0.01"
- Users can "Pin" specific docs or guidelines into persistent cache
- Current tools: caching is invisible backend optimization. Gkestral: first-class UI citizen.

### 3. Grounding as First-Class Workflow [REVISED]

**Core insight:** Native grounding turns agent into active researcher with current documentation. Supplements but does NOT replace local search tools.

- Agent loop distinguishes "Research Mode" (grounding) from "Execute Mode" (local tools)
- "Migrate to React 19" -> grounded research report with citations -> user approves -> execution
- Current tools give model bash curl or Brave search -- slow, token-heavy, scraping-prone
- Gkestral: native grounding with rendered citation metadata in frontend

### 4. Multimodal Input via File API [REVISED]

**Core insight:** Software engineering is visual. Gemini processes images, PDFs, audio, video natively.

- Use File API for out-of-band uploads (NOT base64 InlineData which bloats JSON)
- Chat input becomes universal dropzone
- Drop Figma screenshot: "Make the UI look like this"
- Drop PDF spec: "Implement section 4.2"
- Images and PDFs are first-class workspace citizens, cached alongside code

### 5. Thinking as Tunable Resource

**Core insight:** Deep reasoning for planning, cheap execution for syntax fixes.

- Pro/Thinking for Plan phase; Flash for Act phase
- "Deep Think" toggle in UI
- Thinking traces streamed in collapsible component
- Think deeply about the plan (expensive), execute cheaply (Flash), validate results (cheap)

### 6. Execution Safety: The "Boring Reliable" Layer

**Core insight:** LLMs are magical at synthesis, terrible at deterministic file operations. Execution must be constrained by strict backend logic.

- Gemini outputs standard diffs -> Go backend parses -> dry-run validation -> syntax check -> approval gate -> disk write
- Every file change as diff view. User clicks "Approve"
- Adopt Claude Code's execution discipline for the boring layer
- The magic is in context/reasoning; the boring is in Go binary enforcing strict validation

### 7. Cloud-Agnostic by Design [REVISED]

**Core insight:** Gkestral is a lean assistant. Cloud-specific integrations are scope creep.

- run_command is sufficient for any cloud deployment
- Works equally for Firebase, AWS, Azure, Vercel, bare metal
- If ecosystem integration becomes valuable, implement via plugin, not core tools

---

## Three Additional Findings from Verification

### 8. Parallel Function Calling
Gemini returns multiple tool calls simultaneously, but sequential processing is a latency anti-pattern. Use errgroup/goroutine fan-out.

### 9. Structured Outputs
Missing responseSchema and responseMimeType support means no deterministic JSON execution plans. Essential for Plan phase.

### 10. API Version Strategy
Hardcoded v1beta endpoint. Production needs configurable version with migration path to v1.

---

## Verified Blueprint: Two-Tiered Architecture

```
USER REQUEST
    |
    v
[ORCHESTRATOR] (Go Backend)
    |-- Manages ContextCache (fsnotify + File API)
    |-- Validates execution plans
    |-- Applies diffs with syntax checking
    |-- Runs parallel tool execution (errgroup)
    |
    |-- routes to -->
    |
[BRAIN] (Gemini Pro/Thinking)           [HANDS] (Gemini Flash)
    |-- Cached workspace context            |-- Execution Plan (JSON)
    |-- Multimodal inputs (File API)        |-- apply_diff, search_files, run_command
    |-- Google Search Grounding             |-- Parallel function calling
    |-- Deep thinking for plans             |-- Fast, cheap iterations
    |                                       |
    |-- outputs Structured JSON Plan ------>|
```

**Flow:** User Request -> Brain thinks (deep, cached, grounded) -> Brain outputs structured JSON Plan -> Go Backend validates -> Hands execute in parallel (fast, cheap) -> Go Backend applies diffs with validation -> Loop until complete.

---

## Priority Actions

### v0.1 Immediate
1. Replace write_file with apply_diff / search_and_replace
2. Remove 32k truncation -- trust context window
3. Add responseSchema/responseMimeType for structured outputs
4. Parallelize tool execution (errgroup)
5. Add FileData to Part struct for multimodal
6. Wrap file reads in XML tags for attention routing

### v0.2 Gemini-Native Leap
1. Implement Context Caching (cache.go + fsnotify)
2. Gate full-codebase loading behind active cache
3. Add Google Search grounding
4. Bifurcate agent: Plan (Pro/Thinking) + Execute (Flash)
5. Stream thinking traces in collapsible UI
6. Configurable API version
