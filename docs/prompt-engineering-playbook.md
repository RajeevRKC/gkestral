# Prompt Engineering Playbook -- Gemini-Specific Patterns

> Phase 02 deliverable. Practical patterns grounded in Phase 01 implementation
> experience and Phase 02 agent architecture design.

---

## 1. System Prompt Patterns

### What Gemini Responds To

From Phase 01 `system.go` implementation and testing:

**Effective patterns:**
- Role definition with clear boundaries ("You are a research assistant. You do NOT write code.")
- Explicit output format templates (JSON, markdown, structured sections)
- Constraint lists with numbered items
- Temperature enforcement notes ("Always use temperature 1.0 for Gemini 3.x")
- Tool usage guidance ("Use search grounding for factual claims")

**Diminishing returns beyond ~2000 tokens.** Long system prompts get partially ignored.
Keep the core directive in the first 500 tokens.

**What Gemini ignores or handles poorly:**
- Negative instructions without alternatives ("Don't be verbose" -- say "Be concise, max 3 sentences")
- Abstract personality descriptions ("Be creative" -- say "Generate 3 alternative approaches")
- Instructions that conflict with safety training

### System Prompt Template

```
You are {ROLE} working on {CONTEXT}.

## Task
{Clear statement of what to do}

## Constraints
1. {Constraint with measurable criteria}
2. {Format specification}
3. {Length/scope limits}

## Output Format
{Exact format template with field names}

## Tools Available
- {tool_name}: Use when {specific trigger}
- search_grounding: Use for any factual claim that needs verification
```

### Temperature Rules

| Model Family | Recommended | Reason |
|-------------|------------|--------|
| Gemini 3.x | 1.0 (mandatory) | Lower values cause response looping |
| Gemini 2.5 | 0.7-1.0 | Flexible, lower for deterministic tasks |

Enforced in `system.go` via `EnforceTemperature()`.

---

## 2. Few-Shot vs Zero-Shot

### When Few-Shot Helps

| Task Type | Zero-Shot | Few-Shot | Reason |
|-----------|-----------|----------|--------|
| Classification | Good | Better | Examples anchor categories |
| Format matching | Medium | Excellent | Model mimics example structure |
| Domain jargon | Poor | Good | Examples teach vocabulary |
| Code generation | Excellent | Marginal | Models already know code patterns |
| Summarisation | Excellent | Marginal | Generic skill, examples waste tokens |
| Translation | Excellent | Marginal | Built-in capability |

### Token Cost Trade-off

Each few-shot example consumes context window tokens:
- 3 examples x 200 tokens = 600 tokens/request
- At gemini-2.5-flash rates: $0.000045/request overhead
- At 1000 requests/day: $0.045/day

**Recommendation:** Zero-shot default. Add few-shot only for classification,
format matching, and domain-specific patterns where quality measurably improves.

### Few-Shot Pattern

```go
// In system prompt:
msgs := []gemini.Message{
    {Role: "user", Parts: []gemini.Part{{Text: "Classify: 'The Q3 revenue was $4.2M'"}}},
    {Role: "model", Parts: []gemini.Part{{Text: `{"category": "financial", "confidence": 0.95}`}}},
    {Role: "user", Parts: []gemini.Part{{Text: "Classify: 'Meeting rescheduled to Thursday'"}}},
    {Role: "model", Parts: []gemini.Part{{Text: `{"category": "scheduling", "confidence": 0.92}`}}},
    // Actual query:
    {Role: "user", Parts: []gemini.Part{{Text: "Classify: '" + userInput + "'"}}},
}
```

---

## 3. Chain-of-Thought Strategy

### Gemini 3.x: Built-in Thinking Mode

Do NOT prompt-hack chain-of-thought. Use the native thinking feature:

```go
// From Phase 01 thought.go:
config := gemini.ThinkingConfig{
    Level: "medium",   // off, low, medium, high
    Budget: 8192,      // Max thinking tokens
}
gemini.ApplyThinkingConfig(request, config)
```

**When to enable thinking:**
- Complex multi-step reasoning
- Mathematical/logical problems
- Architecture and design decisions
- Debugging and root cause analysis

**When to skip (Level: "off"):**
- Simple extraction ("What is the date in this email?")
- Format conversion (text -> JSON)
- Translation
- Summarisation of short content

### Gemini 2.5: Explicit CoT Prompting

Add to system prompt: "Think step by step before answering."

```
## Reasoning Protocol
Before giving your answer:
1. Identify the key facts
2. Note any ambiguities
3. Consider alternative interpretations
4. State your confidence level
5. Then provide your answer
```

### Budget Control

| Thinking Level | Approx. Tokens | Use Case |
|---------------|----------------|----------|
| off | 0 | Simple extraction, formatting |
| low | 1024 | Moderate reasoning |
| medium | 4096 | Complex analysis |
| high | 8192+ | Deep reasoning, planning |

Thinking tokens are billed at input rates. Budget directly impacts cost.

---

## 4. Tool Orchestration Patterns

### Pattern 1: Sequential (Default)

```
User query -> Model reasons -> Tool call -> Result -> Model reasons -> Response
```

Best for: single-tool tasks, file operations, simple lookups.

### Pattern 2: Parallel

```
User query -> Model reasons -> [Tool A, Tool B, Tool C] -> Results -> Response
```

Gemini 3.x supports parallel function calls with `id`-based matching.
Best for: gathering data from multiple sources simultaneously.

```go
// From Phase 01 tools.go:
results := gemini.DispatchParallel(ctx, toolCalls, dispatcher)
// Each result matched back by toolCall.ID
```

### Pattern 3: Research-then-Execute

```
Query -> Search grounding -> Grounded context -> Tool calls -> Response
```

Best for: factual tasks that need current information before acting.

```go
// Enable search grounding on the research request:
gemini.EnableSearchGrounding(researchRequest)
// Extract citations:
metadata := gemini.ExtractGroundingMetadata(response)
citations := gemini.ExtractCitationsFromMetadata(metadata)
```

### Pattern 4: Observation Masking

From JetBrains research: 50%+ cost reduction with equivalent solve rates.

Instead of feeding entire tool output back to the model, mask irrelevant parts:
```
Tool output: 500 lines of file listing
Masked output: "Found 3 matching files: report.pdf, data.csv, notes.md"
```

Gkestral should implement this at the tool response layer -- summarise large
outputs before injecting into context.

### Error Recovery

```
Tool call -> Error
  -> If retryable (429, 503): exponential backoff (retry.go)
  -> If user error (404, invalid args): inform model, let it retry with corrected args
  -> If persistent: escalate to user
```

---

## 5. Agent Architecture Patterns

### Gkestral's Model: Architect & Surgeon

```
┌─────────────────────────────────────────────────┐
│  ARCHITECT (broad understanding)                  │
│  Model: gemini-3.1-pro-preview or 2.5-pro        │
│  Context: Full workspace (up to 1M tokens)        │
│  Role: Understand, plan, decide                   │
│                                                   │
│  Reads: codebase, docs, research, user history    │
│  Outputs: plan, analysis, recommendations         │
└──────────────────┬──────────────────────────────┘
                   │ Plan + context summary
┌──────────────────▼──────────────────────────────┐
│  SURGEON (precise execution)                      │
│  Model: gemini-3.1-flash or 2.5-flash             │
│  Context: Focused subset (target files + plan)    │
│  Role: Execute one bounded change at a time       │
│                                                   │
│  Reads: specific files, plan step, constraints    │
│  Outputs: edits, tool calls, status updates       │
└─────────────────────────────────────────────────┘
```

**Key insight:** The Architect consumes the 1M input window to understand deeply.
The Surgeon works within the 8K output window to edit precisely. The asymmetry
between input capacity and output capacity IS the design, not a limitation.

### Pattern: Planner/Executor Split

```go
// Phase 1: Plan with Pro model (deep reasoning)
planRequest := gemini.NewRequest(gemini.WithModel("gemini-3.1-pro-preview"))
planRequest.SystemInstruction = "Create a step-by-step execution plan..."
plan := model.Generate(ctx, planRequest)

// Phase 2: Execute each step with Flash model (fast, cheap)
for _, step := range plan.Steps {
    execRequest := gemini.NewRequest(gemini.WithModel("gemini-3.1-flash"))
    execRequest.SystemInstruction = "Execute this single step: " + step
    result := model.Generate(ctx, execRequest)
}
```

**Cost efficiency:** Pro at $2.50/M input for planning, Flash at $0.15/M for execution.
A 10-step plan costs ~$0.005 to plan and ~$0.015 to execute = $0.02 total.

### Pattern: Research/Action Split

```
Research phase (grounding enabled):
  -> "What are the latest API changes for service X?"
  -> Grounded response with citations

Action phase (tools enabled):
  -> "Based on the research, update the client code"
  -> File edits, code generation
```

**Never mix grounding and code tools in the same request on 2.5 models.**
On 3.x, use VALIDATED mode with `includeServerSideToolInvocations`.

### Pattern: Context-Aware Model Routing

From Phase 01 `router.go`:

```go
// Automatic model selection based on task type:
model := gemini.RouteModel(router, gemini.TaskDeepReasoning)
// Returns gemini-3.1-pro-preview with thinking enabled

model := gemini.RouteModel(router, gemini.TaskFastExtraction)
// Returns gemini-3.1-flash with thinking off
```

---

## 6. Gemini-Specific Gotchas

### Temperature Looping (CRITICAL)
- **Problem:** Gemini 3.x with temperature < 1.0 enters repetitive output loops
- **Solution:** Always set temperature to 1.0 for 3.x models
- **Enforcement:** `EnforceTemperature()` in system.go

### Thought Signature Circulation (CRITICAL)
- **Problem:** Gemini 3.x requires thought parts to be circulated back in multi-turn conversations
- **Solution:** Extract via `ExtractThoughtParts()`, circulate via `CirculateThoughts()`
- **Consequence of failure:** Model loses reasoning context, degrades in multi-turn

### Context Caching is Prefix-Only
- **Problem:** Only the PREFIX of the request can be cached. Variable content must come after.
- **Solution:** Structure requests as: [system prompt | stable context | cached docs] + [user query]
- **Ordering matters:** changing the prefix invalidates the cache

### Safety Settings + Grounding Interaction
- **Problem:** aggressive safety settings can block grounded responses that quote controversial sources
- **Solution:** use `BLOCK_ONLY_HIGH` for research tasks, `BLOCK_MEDIUM_AND_ABOVE` for user-facing output

### Long System Prompts
- **Problem:** Gemini partially ignores instructions beyond ~2000 tokens in system prompts
- **Solution:** Keep core directives in first 500 tokens. Use context caching for reference material instead of stuffing it into the system prompt.

### Google Search Grounding + Function Calling
- **Gemini 2.5:** CANNOT combine in same request
- **Gemini 3.x:** CAN combine with `VALIDATED` mode + `includeServerSideToolInvocations`
- **Implementation:** Separate tool entries in the request (from Phase 01 grounding.go fix)

### include_granted_scopes for OAuth
- **Problem:** Without `include_granted_scopes=true`, incremental OAuth drops previous scopes
- **Solution:** Always include in auth URL for progressive permissioning

---

## Appendix: Prompt Template Library

### Template 1: Document Summarisation
```
Summarise the following document in {N} bullet points.
Focus on: key decisions, action items, and deadlines.
Format: markdown bullet list, each point max 2 sentences.
```

### Template 2: Email Context Import
```
Extract the key information from this email thread.
Output format:
- **Topic:** one-line summary
- **Participants:** names and roles
- **Key Points:** 3-5 bullets
- **Action Items:** numbered list with owners
- **Deadlines:** any mentioned dates
```

### Template 3: Research Synthesis
```
Given the following sources, synthesise a brief on {TOPIC}.
For each claim, cite the source number [1], [2], etc.
Structure: Executive Summary (3 sentences), Key Findings (bullets),
Open Questions (what we still don't know), Recommended Actions.
```

### Template 4: File Analysis
```
Analyse this file and answer: {QUESTION}
Be specific -- reference line numbers, section headers, or data points.
If the answer isn't in the file, say "Not found in document" rather than speculating.
```

### Template 5: Multi-Source Comparison
```
Compare these {N} sources on {DIMENSIONS}.
Output a comparison table with sources as columns and dimensions as rows.
After the table, state which source is strongest for each dimension and why.
```
