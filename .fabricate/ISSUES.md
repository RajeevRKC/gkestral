---
project: Gkestral
updated: 2026-03-24
---

# Issues & Deferred Items

## Open

### ISS-001: List pagination for CachedContents API
- **Source:** M2.7 handshake finding (Phase 01)
- **Severity:** IMPORTANT
- **Target:** Phase 02
- **Description:** CachedContents List endpoint returns paginated results. Current implementation does not handle `nextPageToken`. Need to add pagination support when building cache management features.

### ISS-002: DynamicRetrievalConfig placement validation
- **Source:** Gemini Deep Think handshake finding (Phase 01)
- **Severity:** MEDIUM
- **Target:** Phase 02 (API integration testing)
- **Description:** DynamicRetrievalConfig may need to be at tool level vs request level. Validate against live API when building grounding integration.

### ISS-003: Race detection requires CI with CGO
- **Source:** AC-13 (Phase 01)
- **Severity:** MEDIUM
- **Target:** Phase 03+ (CI pipeline setup)
- **Description:** Windows development environment has CGO disabled, preventing `-race` flag. Need CI pipeline with CGO-enabled Go for race detection. No races detected in code review, but automated testing needed.

### ISS-004: Codex CLI AR-3 final round stalled
- **Source:** Three-way handshake (Phase 01)
- **Severity:** LOW
- **Target:** Phase 02 AR-3
- **Description:** Codex CLI stalled during reference file reading in Phase 01 final handshake. Earlier AR-1 rounds completed successfully. Capture remaining review in Phase 02 handshake cycle.

## Closed

(None yet)
