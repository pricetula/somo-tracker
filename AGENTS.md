# Somotracker — Root Agent Contract

**Project:** TODO(human): one sentence product description

## Document-led workflow
1. Human writes/approves spec in docs/
2. Agent implements only specs with status: approved
3. Code follows docs, never the reverse
4. Changes verified via `make compliance` and `/review`
5. Status moves to implemented → verified by human

## Read before acting
| Task | Read first |
|------|------------|
| New feature | docs/product/brief.md, docs/domain/, docs/features/_template.md, layer AGENTS.md |
| New API | docs/api/openapi.yaml, backend/AGENTS.md, docs/architecture/overview.md |
| New UI | frontend/AGENTS.md, docs/architecture/overview.md |
| Domain change | docs/domain/glossary.md, docs/domain/model.md, docs/domain/rules/ |
| Architecture | docs/adr/, docs/architecture/overview.md |
| Docs update | docs/README.md, docs/workflow/lifecycle.md |

## Commit policy
* Agents may run `git add` and `git commit` locally with conventional messages.
* Never push to `main` or `develop`. PRs target `develop`.
* All commits must pass `make compliance` and lefthook pre-commit checks.

## Implementation rules
* Implement only specs with `status: approved`. Do not implement draft/in-progress.
* If a spec is ambiguous or contradicts a domain rule or ADR, stop and ask human. Do not guess.
* One fact lives in one place. Link, do not copy.

See layer AGENTS.md for rules. See docs/README.md for where to write what.
