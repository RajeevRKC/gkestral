# Google Ecosystem Integration Opportunities

> Source: Deep research across Google APIs, SDKs, and developer tools (2026-03-23)

---

## Integration Assessment Summary

### Available NOW for v0.1 (zero new deps, zero new auth)

| Integration | How | Value |
|------------|-----|-------|
| **Google Search Grounding** | Add `"tools": [{"googleSearch": {}}]` to requests | HIGH -- verified, cited answers from live search |
| **Gemini Code Execution** | Add `"tools": [{"codeExecution": {}}]` to requests | MEDIUM -- built-in Python sandbox |
| **Native Image Generation** | Switch to `gemini-3.1-flash-image-preview` model | HIGH -- images inline in conversations |

### High-Value for v0.2 (one additional API key or OAuth)

| Integration | Auth | Value | Complexity |
|------------|------|-------|-----------|
| **Stitch SDK** | API key from Google Labs | HIGH -- generate UI from text, get HTML/CSS | Medium |
| **NotebookLM** | OAuth or `notebooklm-py` library | HIGH -- research pack synthesis, audio overviews | Medium (fragile unofficial lib) |
| **Gemini Maps Grounding** | Same Gemini API key | MEDIUM -- 250M+ places, coordinates, widgets | Easy |

### Skip Entirely

| Integration | Why Skip |
|------------|---------|
| **Gems** | System instructions do the same thing |
| **AI Studio Build tab** | Browser-only, no API |
| **Antigravity** | Not stable enough |

### Defer to Later

| Integration | Why Defer |
|------------|----------|
| **Firebase** | Overkill for a CLI tool at this stage |
| **Chrome extension bridge** | Grounding covers most research needs |
| **Vertex AI** | Enterprise tier, premature for v0.x |

---

## Detailed Analysis

### 1. Google Search Grounding

**Status:** Production, same API key
**Value:** HIGH
**Implementation:** Add to tools array in request

The grounding metadata provides:
- `webSearchQueries` -- what searches were executed
- `groundingChunks` -- source URIs and titles
- `groundingSupports` -- maps response text segments to sources (character-level attribution)
- `searchEntryPoint.renderedContent` -- REQUIRED by ToS to display

**Conflict with function calling:**
- Gemini 2.5: CANNOT combine with function calling in same request
- Gemini 3: CAN combine with `includeServerSideToolInvocations: true` + VALIDATED mode

**Pricing:**
- Gemini 3: $14/1,000 queries; free: 5,000 prompts/month
- Gemini 2.5: $35/1,000 prompts; free: 1,500 RPD

**Gkestral design:** Use dual-agent architecture on Gemini 2.5 (Planner w/ grounding, Actor w/ tools). On Gemini 3, combine in single request.

### 2. Gemini Code Execution

**Status:** Production, same API key
**Value:** MEDIUM
**Implementation:** Add to tools array

- Python only (can generate other languages but only executes Python)
- 30 second runtime limit, up to 5 retries
- 50+ libraries (pandas, numpy, scipy, matplotlib, scikit-learn, tensorflow, etc.)
- NO network access (sandboxed)
- Image output via matplotlib only
- Gemini 3: CAN combine with function calling
- Gemini 2.5: CANNOT combine

**Gkestral design:** Useful for data analysis, validation, and computation tasks. Not useful for running the user's actual code.

### 3. Native Image Generation

**Status:** Preview
**Model:** `gemini-3.1-flash-image-preview` or `gemini-3-pro-image-preview`
**Value:** HIGH
**Implementation:** Same API, different model name

Generate images inline in conversations. No separate Imagen API call needed. The model switches seamlessly between text and image output.

**Gkestral design:** Canvas pane renders generated images. User asks "generate a diagram of this architecture" and it appears in the right pane.

### 4. Stitch SDK

**Status:** SDK available from Google Labs
**Auth:** API key
**Value:** HIGH
**Complexity:** Medium

Generate UI components from text prompts. Returns HTML/CSS that can be rendered directly in the canvas pane. Can iterate on designs conversationally.

**Gkestral design:** "Design a login page for this app" -> Stitch generates UI -> canvas renders preview -> user approves -> Gkestral extracts HTML/CSS into project files.

### 5. NotebookLM

**Status:** Alpha Enterprise API + unofficial `notebooklm-py` library
**Auth:** OAuth (official) or API key (unofficial)
**Value:** HIGH (for research workflows)
**Complexity:** Medium (fragile)

The unofficial library exposes:
- Create notebooks with sources (PDF, URL, text)
- Generate audio overviews (4 formats)
- Generate video summaries
- Generate slides and mind maps
- Research agent queries against sources

**Gkestral design:** "Research mangrove DMRV methodologies" -> Gkestral creates NotebookLM notebook with relevant sources -> queries for synthesis -> displays research cards in canvas.

### 6. Gemini Maps Grounding

**Status:** Production
**Auth:** Same Gemini API key
**Value:** MEDIUM
**Pricing:** $25/1K prompts, 500/day free

Natural language queries against 250M+ places. Returns place IDs, coordinates, widget tokens.

**Gkestral design:** GIS-related queries render map markers and place information in canvas.

---

## The Strategic Moat

The combination of:
- Google Search grounding
- Maps grounding
- NotebookLM synthesis
- Native image generation
- Stitch UI generation

...creates workflows that Claude Code, Cursor, and Windsurf literally cannot replicate. These are proprietary Google data sources accessible only through Gemini.

**Cost at v0.1-v0.2: $0** (all within free tiers)

---

## Sources

- Gemini API docs: ai.google.dev/gemini-api/docs
- Google AI Forum discussions
- notebooklm-py: github.com/nicholasgcoles/notebooklm-py
- Stitch SDK: Google Labs
- MapLibre GL: maplibre.org
