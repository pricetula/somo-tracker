# Docs Index

Somotracker uses **spec-driven development (SDD)**. Docs are the contract; code follows docs.

## Status legend

| Status | Meaning | Who acts |
|--------|---------|----------|
| `draft` | Spec being written; not actionable | Human |
| `approved` | Cleared for implementation | Human |
| `implemented` | Code + tests complete; `make compliance` green | Agent |
| `verified` | Human reviewed tests against ACs | Human |
| `deprecated` | Superseded; do not implement | Human |

## Where to write what

| Need | File |
|------|------|
| SDD process | `docs/sdd.md` |
| Feature status index | `docs/features/REGISTRY.md` |
| New feature spec | `docs/features/FEATURE-xxx.md` (from `_template.md`) |
| Domain entities & invariants | `docs/domain/model.md` |
| Domain terms | `docs/domain/glossary.md` |
| API contract | `docs/api/openapi.yaml` |
| Architecture decisions | `docs/adr/` |

## Rule

Implement **only** specs with `status: approved`. If implementation reveals the spec is wrong, update the spec in the same commit as the code change. One fact lives in one place — link, do not copy.
