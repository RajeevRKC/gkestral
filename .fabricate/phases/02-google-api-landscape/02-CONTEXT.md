---
phase: 2
title: "Google API & SDK Landscape"
status: active
created: 2026-03-24
---

# Phase 02 Context: Google API & SDK Landscape

## Goal

Map the full Google API surface for information access. Understand what a single
OAuth token can unlock, what costs what, and how to build progressive permissioning.

## What Phase 01 Delivered

- Complete Gemini REST client library (`internal/gemini/`, 13 modules)
- SSE streaming, context caching, function calling, grounding, model routing
- 143 tests, 86.1% coverage
- Search grounding already implemented (reuse in Phase 02)

## Phase 02 Scope (from ROADMAP + REQUIREMENTS)

**Code deliverables:**
1. OAuth 2.0 desktop flow (PKCE, token storage, refresh, scope escalation)
2. Drive API v3 client (browse, download, upload, metadata)
3. Gmail API client (list, search, read, import as context)
4. Phase 01 deferred fixes (CachedContents pagination, DynamicRetrievalConfig)

**Reference deliverables:**
1. Google API strategy document (rate limits, quotas, pricing, scope economics)
2. Prompt engineering playbook (Gemini-specific patterns, agent architecture)

## Key Constraints

- OAuth requires Google Cloud project with consent screen (Commander must set up)
- Desktop PKCE flow (no server-side redirect)
- Token storage must be secure (OS keyring or encrypted file)
- Integration tests need real Google account (build-tag gated)
- Google APIs use `golang.org/x/oauth2` -- acceptable external dependency
- Drive/Gmail use REST (avoid heavy google-api-go-client SDK if possible)

## Architecture Decision

```
internal/
  gemini/     -- Gemini API client (Phase 01, complete)
  google/     -- Google service clients (Phase 02)
    auth/     -- OAuth 2.0 desktop flow, token management
    drive/    -- Drive API v3 client
    gmail/    -- Gmail API client
docs/
  gemini-mastery-reference.md      -- Phase 01
  google-api-strategy.md           -- Phase 02
  prompt-engineering-playbook.md   -- Phase 02
```

## Deferred Items from Phase 01

- ISS-001: CachedContents list pagination
- ISS-002: DynamicRetrievalConfig placement validation
