# Spec-Driven Development (SDD)

## Purpose

A single, well-maintained process doc. Every feature starts as a spec, is approved by a human, implemented by an agent against acceptance criteria, then verified by a human. Code never leads docs.

## Status lifecycle

```
draft ──[human approves]──> approved ──[agent implements + tests]──> implemented ──[human verifies]──> verified
                                                                   ↑
                                                                   └─[spec found wrong]──┘
                                                                       (update spec + code in same commit, revert to approved)
```

| Status | Gate | Who |
|--------|------|-----|
| `draft` | Written from `_template.md` | Human |
| `approved` | Reviewed; ACs are testable | Human |
| `implemented` | Code + tests complete; `make compliance` green | Agent |
| `verified` | Human reviewed tests trace to AC IDs | Human |
| `deprecated` | Superseded by another spec | Human |

## The loop

### 1. Human writes the spec (`draft`)
Create `docs/features/FEATURE-xxx.md` from `_template.md`. Define:
- Goal (1–2 sentences)
- Out of scope
- Domain references (links only, to `docs/domain/model.md`)
- Acceptance criteria (numbered `AC-1`, `AC-2`… — each must be testable)
- API changes (pointer to `docs/api/openapi.yaml` diff or "none")
- Backend notes / Frontend notes (implementation hints only)
- Verification (commands/tests citing AC IDs)

### 2. Human approves (`approved`)
Flip `status:` to `approved`. Add an entry to `docs/features/REGISTRY.md`. No code may be written against a `draft` spec.

### 3. Agent implements (`implemented`)
- Update `docs/api/openapi.yaml` if the API changes.
- Run `make generate` (sqlc + swag + openapi-typescript) — must be idempotent.
- Implement backend + frontend + tests. Every test must cite AC IDs (e.g., `Test_AC-1_...`).
- Run `make compliance`. It must pass.
- Flip `status:` to `implemented`, set `implemented_by` and `implemented_at`.
- Commit spec + code + tests together.

### 4. Human verifies (`verified`)
Human reviews tests against each AC. Flips `status:` to `verified`, sets `verified_by` and `verified_at`. Merges to `develop`.

### 5. Spec revision mid-implementation
If implementation reveals the spec is wrong:
1. Edit `docs/features/FEATURE-xxx.md` (revert `status:` to `approved` if it was `implemented`).
2. Update code + tests to match.
3. Run `make compliance`.
4. Commit spec + code + tests in the same commit.
5. Return `status:` to `implemented`.

## Traceability

Every acceptance criterion must have at least one test citing its AC ID. This makes `/review` and future audits deterministic: `grep AC-1` across tests shows coverage.

## Compliance gate

`make compliance` must pass before any spec moves to `implemented`. No exceptions.
