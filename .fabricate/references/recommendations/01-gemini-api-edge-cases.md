# Gemini API Behavioral Nuances & Edge Cases

> Source: Deep research across GitHub issues, developer forums, academic papers (2026-03-23)
> Confidence ratings: HIGH/MEDIUM/LOW per finding

---

## 1. Function Calling Behavior

### 1.1 Parallel function calling crashes on Gemini 2.x (HIGH)
Gemini API enforces strict 1:1 mapping between function call parts and response parts. When a tool returns binary content alongside text, the response gets split into multiple parts, violating this constraint -> 400 INVALID_ARGUMENT. Gemini 3.x handles multimodal function responses correctly; 2.x does not.
- Source: gemini-cli #16135

### 1.2 Parameter hallucination with large tool inventories (HIGH)
When 100+ tools registered simultaneously, Gemini injects parameters from unrelated tools into calls. Root cause: escaped quotes in tool descriptions corrupting schema parsing.
- Source: gemini-cli #16318, Fleece AI Tool Calling Benchmark
- **Action**: Keep simultaneous tool count under 20. Implement progressive tool discovery.

### 1.3 Function calling breaks with PDF attachments on Gemini API (HIGH)
Tool calling fails when PDF attachments present in Gemini API requests (not Vertex AI). Model halluccinates responses, claims tools return empty results, invents random UUIDs. Confirmed broken on 2.5-flash, 2.5-pro, 3-flash-preview, 3-pro-preview via Gemini API. Works correctly via Vertex AI.
- Source: Google AI Forum: "Tool Calling Completely Broken"
- **Workaround**: Use Vertex AI auth for multimodal+tool scenarios.

### 1.4 AUTO mode deprioritises tools in long conversations (MEDIUM)
In AUTO mode, Gemini progressively "forgets" tool definitions after 15-20+ turns or 100k+ tokens. Starts describing actions instead of executing them. ANY mode forces tool calls but prevents plain text responses.
- Source: obsidian-gemini #328
- **Fix**: Context compaction, not mode switching.

### 1.5 Benchmark comparison (HIGH)

| Benchmark | Gemini 3.1 Pro | Claude Opus 4.6 | GPT-5.2 |
|-----------|---------------|-----------------|---------|
| TAU2-Bench (multi-turn) | ~90%+ | ~90%+ | 98.7% |
| MCP-Atlas (cross-server) | 69.2% | 60.3% | -- |
| APEX-Agents (professional) | 33.5% | 29.8% | 23.0% |

Gemini leads at multi-service coordination. Claude leads at long-horizon autonomous operation.

### 1.6 VALIDATED mode for schema enforcement (MEDIUM)
Stricter than AUTO but less rigid than ANY. Constrains model to predict either function calls or natural language, enforces schema adherence.

---

## 2. Context Window Behavior at Scale

### 2.1 Effective capacity is 60-70% of advertised (HIGH)
"Context Rot" study (Chroma Research) tested 18 models including Gemini 2.5 Pro/Flash:
- Strong through 500 tokens on trivial tasks
- Notable decline 500-2,500 tokens on repeated-word tasks
- Severe degradation beyond 5,000-10,000 tokens on simple recall
- Gemini 2.5 Pro showed instability, generating random character sequences around 750 words
- Average recall at 1M tokens hovers around 60%
- Source: research.trychroma.com/context-rot

### 2.2 "Lost in the middle" U-curve is real (HIGH)
Information at beginning/end achieves 85-95% accuracy; middle drops to 76-82%. >30% drop when relevant info shifts from edges to center. Caused by RoPE attention score degradation.

### 2.3 App-layer truncation vs model capability (MEDIUM)
Gemini app loses memory at 120-150k tokens, but AI Studio retains at 500k+. Suggests app-level compression, not model failure.

### 2.4 Large context is counterproductive for coding without validation (HIGH)
Controlled test: Gemini with 2M context modified 31 files, introduced 260+ type errors. Multi-agent focused context: 3 files, zero errors. "Context engineering -- providing the right information, in the right format, at the right time."
- Source: hyperdev.matsuoka.com

### 2.5 Practical sweet spot
Keep context under 100k tokens for coding tasks. Use focused, relevant context. Place critical instructions at beginning AND end. Implement incremental validation.

---

## 3. Context Caching Real-World Behavior

### 3.1 Implicit caching minimums vary and change silently (HIGH)

| Model | Documented Min | Actual Min |
|-------|---------------|-----------|
| Gemini 2.5 Flash | 1,024 | 1,024 |
| Gemini 2.5 Pro | 2,048 | 4,096 (silently changed) |
| Gemini 3.x | 4,096 | 4,096 |

### 3.2 Timestamps in system prompts kill cache hits (HIGH)
Any timestamp with seconds-level precision prevents caching because prefix changes on every request.

### 3.3 Implicit caching offers no TTL control or guarantee (MEDIUM)
In testing, 6 of 12 identical requests missed the cache. No guarantee on persistence.

### 3.4 Explicit caching requires 32,768+ tokens (HIGH)
Cost: 10% of standard input. Default TTL: 1 hour. No min/max bounds on TTL.

### Recommendation
Structure system instructions + tool definitions as stable prefix (no timestamps, no dynamic content). Put per-request content at the end. Consider padding with stable reference content to hit caching threshold.

---

## 4. Streaming Reliability

### 4.1 Streams ending without finish_reason -- known, unresolved (HIGH)
Multiple gemini-cli issues. Retry logic in v0.8+ does NOT treat as retryable.
- Source: gemini-cli #10678, #7851, #8324

### 4.2 Endless streams that never send STOP (MEDIUM)
Gemini Pro gets stuck sending chunks indefinitely.
- Source: Google AI Forum
- **Action**: Client-side timeout (120s idle) + max token guard.

### 4.3 Interrupting stream corrupts subsequent function calling (MEDIUM)
Mid-stream interruption -> mismatched part counts -> 400 errors.
- Source: gemini-cli #4324

### 4.4 Latency spikes with gemini-3.1-pro (HIGH)
Simple prompts take 10+ minutes or hang. Specific to CLI/API interactions.

### Best retry strategy
Exponential backoff with jitter. Start 1s, double each retry, 10-30% random jitter. Retry 429/500. Do NOT retry 400 INVALID_ARGUMENT. Treat missing finish_reason as retryable.

---

## 5. Thinking Mode in Practice

### 5.1 Thinking on by default in Gemini 3 (HIGH)

| Level | Best For | Latency |
|-------|----------|---------|
| minimal | Chat, high-throughput | Minimal but not zero |
| low | Simple tasks | Low |
| medium | Balanced reasoning | Moderate |
| high (default 3.1 Pro, 3 Flash) | Complex reasoning | Significant |

### 5.2 Empty responses when thinking consumes output budget (HIGH)
When max_output_tokens set and thinking consumes most of budget, response text is empty. 200 OK with empty response.
- Source: python-genai #782
- **Action**: Set MaxOutputTokens to 16384+ or set thinking_level to "low" for coding.

### 5.3 Do NOT mix thinking_level and thinking_budget (HIGH)
Mutually exclusive. 400 error if both present.

### 5.4 Thought signatures in streaming (MEDIUM)
May arrive in final chunks with empty text fields. Check for signatures even when text blank. Appear at end, not beginning.

### 5.5 Thinking improves tool selection (MEDIUM)
Gemini 3 Pro with "high": 35% better on SWE-bench vs 2.5 Pro. Marginal improvement for simple 1-3 call routing. Significant latency cost.

---

## 6. Google Search Grounding

### 6.1 Grounding reduces hallucination by ~40% (MEDIUM)
Independent: open-domain hallucination 12% without, potentially 4% with grounding.

### 6.2 Exact grounding metadata format (HIGH)
```json
{
  "groundingMetadata": {
    "webSearchQueries": ["string"],
    "searchEntryPoint": {"renderedContent": "HTML/CSS string"},
    "groundingChunks": [{"web": {"uri": "string", "title": "string"}}],
    "groundingSupports": [{
      "segment": {"startIndex": N, "endIndex": N, "text": "string"},
      "groundingChunkIndices": [N]
    }]
  }
}
```

### 6.3 Search suggestions widget required by ToS (HIGH)
`searchEntryPoint.renderedContent` must be rendered. Not optional.

---

## 7. Model Routing

### 7.1 Flash BEATS Pro on coding at 75% lower cost (HIGH)
- Flash: 78% SWE-bench, $0.50/M input, 3x faster
- Pro: 76.2% SWE-bench, $2-4/M input, slower
- Source: GLBGPT, AI Free API

### 7.2 Pro deletes unrelated code (HIGH)
Multiple reports of Pro wiping large unrelated code sections.

### 7.3 customtools model variant (HIGH)
`gemini-3.1-pro-preview-customtools` prioritises custom tool declarations over bash. Standard variant defaults to shell commands. If >50% requests involve tools, use customtools.
- Source: Apiyi

### 7.4 Practical routing

| Task | Best Model |
|------|-----------|
| Iterative coding | Flash |
| Complex architecture | Pro + thinking "high" |
| Simple file ops | Flash Lite |
| Tool-heavy workflows | Pro customtools |
| Quick classification | Flash + thinking "minimal" |

---

## 8. Gemini-Specific Prompt Engineering

### 8.1 Use XML or Markdown consistently, never mix (HIGH)
```
<role>...</role>
<instructions>Numbered steps</instructions>
<constraints>Bulleted rules</constraints>
```

### 8.2 Temperature MUST be 1.0 for Gemini 3 (CRITICAL)
Lower values cause looping and degraded performance. Google explicitly warns.
- Source: Gemini 3 Developer Guide, Gemini 3 Prompting Guide

### 8.3 Place questions AFTER context data (HIGH)
Gemini anchors better when data precedes the question.

### 8.4 Gemini 3 prefers brevity (HIGH)
Verbose prompts hurt. Replace chain-of-thought with thinking_level: "high".

### 8.5 Agentic persistence directive (MEDIUM)
Add: "Continue working until the user's query is COMPLETELY resolved. If a tool fails, analyze the error and try different approaches."

---

## 9. Gemini 3.x Specifics

### 9.1 Thought signatures MANDATORY for function calls (CRITICAL)
- First functionCall MUST include thoughtSignature
- Parallel calls: only first gets signature; return parts in exact order
- Omitting -> 400 error
- Bypass for migrated conversations: dummy string "context_engineering_is_the_way_to_go"

### 9.2 Errors look like reasonable misunderstandings (MEDIUM)
Easier to debug, harder to detect automatically.

### 9.3 Computer use built-in to Gemini 3 (MEDIUM)
No separate model variant needed.

### 9.4 Multimodal function responses new in Gemini 3 (HIGH)
Function responses can contain images. Combined with built-in tools in single API calls.

---

## 10. CRITICAL Bugs in Current POC

1. **Temperature 0.3** in agent.go:79 -> WILL cause looping on Gemini 3. Set to 1.0.
2. **No ThoughtSignature field** in Part/FunctionCall structs -> ALL multi-step tool calling will fail on Gemini 3 with 400 errors.
3. **No sentinel event** after SSE scanner loop -> streams ending without finishReason close silently.
4. **MaxOutputTokens 8192** -> thinking can consume entire budget, producing empty responses. Raise to 16384+.
