# Mandan's Response to Brainstorm -- Vision Correction

Date: 2026-03-23
Context: Commander corrected Mandan's persistent "coding CLI" gravity

---

## The Correction

Mandan keeps pulling the product toward "coding tool that also does research."
Commander says it's the opposite: **"information tool where code runs behind the scenes."**

This is not a nuance. It is a fundamentally different product.

## What Commander Actually Said

1. Terminal is the interaction surface -- but NOT for coding. For information.
2. Tomorrow's AI is about interaction, enabling, and information. Not coding.
3. The student example: researching for exams, creating documents, saving to local workspace. No coding visible.
4. Code happens behind the scenes. When you create a DOCX, Python runs. When you extract PDF data, code runs. The user doesn't see it or care.
5. Someone who specifically wants to code CAN. But that's one mode, not the identity.
6. Subscription tier determines intensity. Free Google account = basic capability. Paid = more.
7. Single source of access replaces going to Gemini portal, Drive, Gmail, Maps separately.
8. Local-first: workspaces are folders. Sessions persist on disk. Documents stay local. Not lost in browser tabs.

## The Product Reframe

| What I Kept Saying | What Commander Means |
|-------------------|---------------------|
| Coding CLI with research | Information workbench where code is infrastructure |
| Developer-first | Anyone who works with information |
| "Screenshot to code" demo | "Research to document" demo |
| Tools = read_file, write_file, search | Tools = research, synthesize, create, extract, transform |
| Canvas shows code diffs | Canvas shows documents, research, images, maps |
| Terminal-inspired conversation | Natural conversation about objectives |

## The Real Product

A student opens Gkestral. They have a folder for their History course.

They say: "I need to write a 2000-word essay on the economic impact of the Silk Road. Here's my textbook chapter (drops PDF). Find me 5 more sources."

Gkestral:
1. Reads the PDF (multimodal)
2. Researches via Google Search grounding (5 cited sources appear as cards in canvas)
3. The student reviews, says "use sources 1, 3, and 5"
4. Gkestral synthesizes a structured essay with citations
5. Saves it as a DOCX in their course folder
6. The session persists -- they can come back tomorrow and refine

Behind the scenes: Python ran to read the PDF. Python ran to create the DOCX. Gemini handled the research and synthesis. The student saw none of the code. They just got their essay.

Now Commander opens Gkestral. He has 19 workspaces across carbon, construction, publishing, apps.

He says: "Check my Gmail for anything from Marcin about the CBAM India project. Cross-reference with the latest carbon market intel."

Gkestral:
1. Queries Gmail (Google Workspace integration)
2. Finds Marcin's emails
3. Searches Google for current CBAM India regulations (grounding)
4. Cross-references with Commander's carbon workspace data
5. Produces a brief in the canvas
6. Commander says "draft a response to Marcin"
7. Gkestral drafts the email, Commander approves, it sends

Behind the scenes: API calls, JSON parsing, document generation. Commander saw none of it.

A developer opens Gkestral. They have a Next.js project.

They say: "The auth is broken. Here's the error screenshot (drops image). Fix it."

Gkestral:
1. Reads the screenshot (multimodal)
2. Reads the codebase (context caching)
3. Identifies the bug
4. Shows a diff in the canvas
5. Developer approves
6. Code is applied

This is the SAME tool. The same loop. The same architecture. But coding is just one of its modes, not its identity.

## What This Changes About Architecture

The six subsystems from PROJECT.md still hold, but the weight shifts:

| Subsystem | Old Weight | New Weight |
|-----------|-----------|-----------|
| Execution Core | Heavy (tools, diffs, edits) | Light (runs behind scenes) |
| Context Engine | Heavy (codebase loading) | Heavy (any content: PDF, email, docs, code) |
| Research Engine | Medium (grounding as feature) | Heavy (primary workflow mode) |
| Multimodal Workspace | Medium (canvas as nice-to-have) | Heavy (this IS the product surface) |
| Gemini Integration | Heavy | Heavy (unchanged) |
| Integration Layer | Light (defer) | Heavy (Gmail, Drive are core, not v0.2) |

## What This Changes About v0.1

The first demo should NOT be "screenshot to code fix."
The first demo should be: **"I have a scattered objective. Gkestral helps me complete it."**

The simplest expression of this:
- User drops a PDF
- Asks a question about it
- Gkestral answers with grounded research
- User asks for a summary document
- Gkestral creates a DOCX in their workspace folder
- Session saved. Come back tomorrow.

That demo proves: multimodal input, research, synthesis, document output, local persistence. No code visible.

## The Sales Pitch to Google

Not: "We built a better Gemini CLI for developers."
But: **"We built the desktop product that makes a Google account the most productive subscription in the world."**

Free tier: basic Gemini, basic grounding, local workspace.
Paid tier: more tokens, caching, Drive/Gmail integration, advanced research.

Every Gkestral user is a Gemini API consumer. Every power user is a paid API consumer. Google wins.

## Mandan's Admission

I was wrong to keep pulling this toward coding. Commander sees the larger opportunity. The product that serves a student, a researcher, a founder, AND a developer from the same interface -- with code as invisible infrastructure -- is a much bigger idea than another coding CLI.

The vision is: **"Turn scattered context into finished work."**

The coding-only version of that vision is a subset. A small one.
