# Phase 5 — Specifications

Detailed, implementable feature specifications for Somotracker.

## Status

empty — no specifications exist yet. None are approved.

## Specification lifecycle

| Status | Meaning | Who may set it |
|--------|---------|----------------|
| draft | Authoring; not for implementation | Author (human or agent) |
| proposed | Submitted for review | Author |
| approved | Authorized for implementation | Human only (agents must not approve their own specs) |
| implemented | Code written; compliance passing | Agent (after implementation) |
| verified by human | Reviewed and accepted | Human only |
| rejected | Will not be implemented | Human only |

## Document-led workflow (root AGENTS.md)

1. Human writes/approves a specification with `status: approved`.
2. Agents implement only approved specifications. Draft / in-progress / proposed specs are **not** implemented.
3. Code follows documentation; never the reverse.
4. Verify with `make compliance` and resolve review feedback.
5. Status moves `implemented → verified by human`.

## Specification template

Each specification is a single Markdown file under `docs/specifications/` with this frontmatter:

```markdown
---
id: SOM-001            # unique spec id (FE/BE prefixed where it maps to one feature)
title: ...             # short title
status: draft          # draft | proposed | approved | implemented | verified by human | rejected
author: ...
date: YYYY-MM-DD
updated: YYYY-MM-DD
fr_ids: [BE-016, FE-007]   # requirements satisfied (docs/requirements/)
nf_ids: [BE-031, BE-032]
api_contract: docs/api/openapi.yaml  # changed? yes/no
migrations: backend/migrations/      # new/changed migrations, if any
generated_files: []                  # sqlc/swag/openapi-typescript outputs affected
---

## 1. Summary

## 2. Requirements (FR/NFR IDs with rationale)

## 3. Domain model affected (entities, business rules)

## 4. API contract (endpoints, request/response DTOs, errors)

## 5. Workflow (Given/When/Then acceptance criteria)

## 6. Implementation plan (files to touch, by application)

## 7. Tests required (per BE-036 / FE-026 / FE-027)

## 8. Open questions

## 9. Links
```

## Open questions blocking specifications

- Authz permission matrix is unimplemented (`platform/authz` primitives exist; matrix data and RBAC wiring needed) — see `docs/requirements/gaps.md`.
- Stytch organization → tenant/school mapping not fully spelled out in code — confirm before organization-selection workflows.

## Registry

See `docs/specifications/registry.md` — the living index of all specifications and their status.
