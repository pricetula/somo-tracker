# Phase 4 — Requirements

Functional and non-functional requirements for Somotracker.

## Status

draft — bootstrapped from existing `AGENTS.md` rules. These are **requirements already encoded in code review rules**, not new inventions.

## Requirement ID scheme

| Prefix | Source | Scope |
|--------|--------|-------|
| `BE-001`…`BE-045` | `backend/AGENTS.md` | Backend rules (structure, handlers, data, tenancy, API, jobs, security, quality, migrations) |
| `FE-001`…`FE-027` | `frontend/AGENTS.md` | Frontend rules (architecture, data, layout, UI, domain, naming) |
| `FE-AUTH-01`…`FE-AUTH-03` | `frontend/AGENTS.md` addendum | Auth guard and server-only imports |

## How to read this catalog

- Each ID below links (by ID) to its source rule in `backend/AGENTS.md` or `frontend/AGENTS.md`. Rule text is not duplicated here — one fact lives in one place.
- `docs/specifications/registry.md` maps specifications to the FR/NFR IDs they satisfy.
- Until a specification is approved, AGENTS.md rules remain authoritative. Agents must not build requirements without Given/When/Then acceptance criteria (`FE-026`, `BE-035`).

## Backend requirements

See `docs/requirements/backend.md`.

## Frontend requirements

See `docs/requirements/frontend.md`.

## Implied / unmet requirements (gaps)

See `docs/requirements/gaps.md`.
