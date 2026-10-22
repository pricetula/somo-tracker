# Development Process Revamp Plan

**Status:** Draft → In Progress
**Owner:** Platform team
**Date:** 2026-10-22

## Problem statement
Current process is document-led and AI-agent-optimised with manual skill loading. Skills are markdown instructions, not executable bots. Compliance is convention based, not CI enforced. API contract duplication exists between `docs/openapi.yaml`, `backend/docs/swagger.json`, and frontend generated types.

## Goals
1. Keep document-led philosophy and contracts.
2. Make contracts executable via bot factory.
3. Single source of truth for API: `docs/openapi.yaml`.
4. Zero manual skill reading for developers.
5. Automated compliance before merge.

## Principles
- Docs first, code second — preserved.
- No commit by AI — preserved.
- Bots enforce, humans approve.

## Bot Factory Architecture

### Bots
- **docs-bot**: `pnpm docs:lint`, `pnpm audit:docs`, link validation, frontmatter, tooltipSummary <160.
- **api-bot**: Validates generate-api workflow, handler/service separation, swagger annotation matches router, `make generate` produces identical artifacts.
- **security-bot**: OWASP API Top 10 checks, BOLA/BFLA tests, rate limits, error contract, produces `security-report.md`.
- **migration-bot**: Every `backend/db/migrations/*.sql` must have `TestMigrator_*` with `//go:build integration`, schema object assertions.
- **ui-bot**: Pure-shadcn rules, no hex colors, no div soup, no back buttons, `pnpm lint` passes.

### Triggers
- Label based: `api`, `security`, `migration`, `ui`, `docs`.
- Path based: changes under `backend/internal/api/*` → api-bot + security-bot.
- PR size / risk based: auto-run all bots on `main` merge.

## Phase 1 - Foundations
1. Create `.github/workflows/bots.yml` with jobs for each bot.
2. Add CI gate `make compliance` that runs agendas-compliance checks via script.
3. Add `Makefile compliance` target.
4. Create `scripts/compliance.sh` to enforce error-contract, forbidden patterns.
5. Unify API generation: `make generate` must be idempotent and fail if `docs/openapi.yaml` diverges.

## Phase 2 - Automation
1. Add pre-commit hooks via lefthook for error-contract lint and docs lint.
2. Auto-generate Postman collection from openapi on CI.
3. Bot comments on PR with findings categorized Critical/High/Medium/Low.

## Phase 3 - Cleanup
1. Reduce planning docs that never become features; auto-archive stale gaps >90 days.
2. Add `docs/features/README.md` auto-status board.
3. Migrate existing skills to executable check scripts under `scripts/bots/`.

## Success criteria
- PR cannot merge without all bots green.
- `make generate` produces zero diff on main.
- No manual skill reading required for new feature workflow.
- Gap count decreases 20% per quarter.

## Next actions
- [ ] Create `.github/workflows/bots.yml`
- [ ] Create `scripts/compliance.sh`
- [ ] Update root AGENTS.md with bot factory
- [ ] Add `make compliance` target
