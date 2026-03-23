# Self-Improving Agent Architecture & Autoresearch Patterns

> Source: Research across academic papers, open-source projects, and industry reports (2026-03-22)

---

## 1. How Current Tools Learn

| Tool | Context File | Learning Mechanism |
|------|-------------|-------------------|
| Claude Code | CLAUDE.md | User-curated, hierarchical discovery |
| Gemini CLI | GEMINI.md | Same pattern, hierarchical merge |
| Aider | .aider.conf.yml + AGENTS.md | Config-based |
| Codex CLI | AGENTS.md | Community convention |
| Cursor | .cursor/rules | Rule files per directory |

**Gap:** None automatically improve their own context files. Improvement loop is human-in-the-loop.

## 2. Six Self-Improvement Mechanisms (Nakajima Taxonomy)

1. **Self-Reflection (runtime)** -- Reflexion pattern. Critique, store, retry. Ephemeral.
2. **Self-Correction Training** -- STaR/RISE. Requires training infra. Not applicable for CLI.
3. **Self-Generated Data** -- Voyager's skill library. Agent creates training examples from success.
4. **Self-Adapting Models** -- SEAL. Too heavy for CLI.
5. **Self-Improving Code Agents (SICA)** -- Agent edits own scaffolding. 17% -> 53% on SWE-bench. **Most relevant.**
6. **Embodied Self-Improvement** -- Via test execution feedback.

## 3. SICA Results (ICLR 2025) -- Key Reference

- Agent edits its own Python scaffolding across 15 iterations
- Composite utility: `U = 0.5(benchmark) + 0.25(cost) + 0.25(speed)`
- Self-discovered: diff-based editing, AST symbol locators, ripgrep context summarizers, hybrid search
- Diminishing returns on reasoning -- biggest gains on agentic/tool-use tasks
- Archive stores every iteration with full metrics

**Gkestral relevance:** HIGH. Go tools are the direct analogue of SICA's editable scaffolding.

## 4. Karpathy's Autoresearch (March 2026, 42k stars)

Three files: `prepare.py` (immutable), `train.py` (agent edits), `program.md` (human edits).
Loop: edit -> train 5min -> measure val_bpb -> keep/discard -> repeat.
~12 experiments/hour, ~100 overnight.

**Key insight:** "The human shifts from experimenter to experimental designer."

### Gkestral adaptation:
```
1. Record tool call patterns from sessions (metrics.go)
2. Identify bottlenecks: slow searches, failed edits, excessive iterations
3. Generate improvement hypotheses (via Gemini)
4. Apply improvement to tool implementation
5. Run eval suite against benchmarks
6. Accept if metrics improve, reject otherwise
7. Repeat
```

## 5. Metrics to Track

| Metric | What | How |
|--------|------|-----|
| Tool call accuracy | Useful results? | Model's next action after result |
| Edit success rate | Edits survive? | Write -> subsequent read of same file |
| User acceptance | Response accepted? | Explicit feedback or continued conversation |
| Iteration count | Loops per task | Counter in agent |
| Time-to-resolution | Wall clock | Timestamps |
| Token efficiency | Tokens per task | API metadata |
| Cache hit rate | KV reuse | API headers |

## 6. Context Learning -- Three-Tier Architecture

### Tier 1: Session Memory (Ephemeral)
- Current conversation history
- Compression via **observation masking** (JetBrains: superior to LLM summarization for coding)
- Keep last N turns verbatim, mask older tool outputs
- JetBrains: 50%+ cost reduction, equivalent or better solve rates
- **Key: LLM summarization causes "trajectory elongation" -- 13-15% longer runs**

### Tier 2: Project Memory (Persistent per-project)
- KESTRAL.md files (hierarchical)
- Discovered patterns, effective tool chains
- Updated at session end

### Tier 3: Global Memory (Cross-project)
- Skill library: reusable patterns
- User preferences
- Error patterns and fixes
- Promotion from project memory

### Factory.ai's Anchored Iterative Summarization (Best-in-class)
- Structured summaries: session intent, file mods, decisions, next steps
- New info merges into persistent summaries
- Scored 3.70 vs Anthropic 3.44 and OpenAI 3.35

## 7. RAG vs Full Context vs Agentic Search

Industry moving AWAY from RAG for coding agents.

| Approach | Best For | Weakness |
|----------|---------|----------|
| Full context | Small-medium codebases (<100k tokens) | Expensive, breaks on large repos |
| RAG | Document retrieval, knowledge bases | Context amnesia, 87% enterprise ROI failure |
| Agentic search | Coding tasks | Higher inference cost, superior reasoning |

**Gkestral's current tool chain (list_dir + search + read_file) IS the agentic search pattern.** Correct architecture.

## 8. Prompt Optimization Loops

### DSPy (Stanford)
Replace hand-written prompts with signatures + optimizers.
- BootstrapFewShot: generates successful examples
- MIPRO: Bayesian optimization, 100-300 evals vs 500-1000 brute force

### OPRO (Google DeepMind)
LLM as optimizer. Feed previous prompt-score pairs, ask for better prompt. Iterative hill-climbing.

### TextGrad (Stanford/Nature 2025)
"Textual gradients" -- natural language critiques as optimization signal.

### Phased approach for Gkestral:
1. v0.1: Manual KESTRAL.md
2. v0.3: Session-end reflection
3. v0.5: OPRO-style prompt self-tuning with A/B testing
4. v1.0: DSPy-inspired signature optimization for tool descriptions

## 9. Agent Loop Patterns

| Pattern | Coding Fit | Cost | Adaptability |
|---------|-----------|------|-------------|
| ReAct | Good default | Medium | High |
| Reflexion | Best for retry-heavy | Higher | Very high |
| LATS (tree search) | Overkill | Very high | Maximum |
| Plan-and-Execute | Well-defined tasks | Higher upfront | Low |
| **Hybrid** | **Best overall** | Medium-high | High |

### Recommended evolution:
- v0.1: Basic ReAct
- v0.2: ReAct + observation masking
- v0.3: ReAct + Reflexion
- v0.4: Hybrid Plan-and-Execute for complex tasks
- v0.5: Autoresearch loop

## 10. Eval Suite Structure

```
evals/
  cases/
    01-read-and-explain/
    02-find-bug/
    03-add-feature/
    04-refactor/
    05-multi-file-edit/
  runner.go
  scorer.go
  regression.go
```

---

## Sources

- Karpathy's AutoResearch: github.com/karpathy/autoresearch
- SICA (ICLR 2025): arxiv.org/html/2504.15228v2
- Reflexion (NeurIPS 2023): github.com/noahshinn/reflexion
- DSPy: github.com/stanfordnlp/dspy
- TextGrad (Nature 2025): github.com/zou-group/textgrad
- JetBrains Context Management: blog.jetbrains.com/research/2025/12
- Factory.ai Compression: factory.ai/news/evaluating-compression
- TDAD (2026): arxiv.org/html/2603.17973
- SWE-bench: swebench.com
