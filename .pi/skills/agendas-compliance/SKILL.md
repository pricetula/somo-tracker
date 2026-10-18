---
name: agendas-compliance
description: Validate adherence to AGENTS.md contracts across backend, frontend, and docs. Used before PRs to ensure all contracts are respected.

# AGENTS.md Compliance Validation

## When to Run
This skill is triggered when:
- A PR touches backend, frontend, or public code
- User requests an "architecture review", "contract compliance", or "system audit"
- Before merging any feature branch

## Step 1 — Contract Verification

### Backend Contracts (backend/AGENTS.md)
- [ ] Migration testing required for every new migration
- [ ] Handler/service separation enforced (no raw SQL in handlers)
- [ ] Error responses use canonical APIError contract
- [ ] Academic period resolver used for all term/year resolution
- [ ] Swagger endpoint verification completed

### Frontend Contracts (frontend/AGENTS.md)
- [ ] No "dumb back buttons" (no ArrowLeft icons in features/pages)
- [ ] One component per file (no multi-component files)
- [ ] Flat layout tree (no div soup)
- [ ] No hardcoded hex colors or custom Tailwind classes
- [ ] Error handling follows canonical error contract
- [ ] Form implementations follow onboarding-form pattern
- [ ] Shadcn components are untouched (no manual modifications)
- [ ] Page layout uses vertical stacking (no tabs)
- [ ] Form implementations follow onboarding-form pattern
- [ ] Error boundaries wrap all major routes
- [ ] Async handlers have proper try/catch
- [ ] API mutations pass IDs in request body (not just URL params)

### Docs Contracts (docs/AGENTS.md)
- [ ] All new features use templates (gap.md, feature-spec.md)
- [ ] All changes must be documented in content/docs
- [ ] ADR follows standard format
- [ ] Gap analyses are complete and up to date

## Step 2 — Code Validation

### Backend Validation
- [ ] Every handler calls a service method (not direct DB access)
- [ ] No `if err != nil {}` without proper error propagation
- [ ] No `_ = someFunc()` silent error patterns
- [ ] No empty catch blocks
- [ ] All API responses follow error contract
- [ ] Migration tests exist for all new migrations

### Frontend Validation
- [ ] No back navigation buttons in features or pages
- [ ] No div wrappers around Shadcn components
- [ ] No hardcoded hex colors (use semantic CSS vars)
- [ ] No nested div layouts (use fragments or Tailwind spacing)
- [ ] All error states render `<Alert>` on `isError`
- [ ] All async handlers have try/catch
- [ ] All feature modules follow index.ts entry point pattern
- [ ] All components use pure Shadcn (no custom wrappers)

## Step 3 — Tool Integration

This skill can run:
- `make build-backend` (backend compliance)
- `npm run build:frontend` (frontend compliance)
- `pnpm run docs:build` (docs compliance)
- `pnpm run test:all` (all tests)

## Reporting
Findings should be categorized by:
- **Critical** – contract violations (e.g., missing migration test, empty catch)
- **High** – architecture violations (e.g., layer boundary crossings)
- **Medium** – style violations (e.g., div soup, hardcoded colors)
- **Low** - documentation gaps

## Integration Points
- Triggered by `/compliance` command
- Validates all AGENTS.md files (root, backend, frontend, public)
- Can be combined with security-audit and migration-testing skills
- Outputs `compliance-report.md` with detailed findings

## Prerequisites
- All AGENTS.md files must exist (root, backend, frontend, public)
- All build/test scripts must be executable (`make build-backend`, `npm run build:frontend`, etc.)
- All CI/CD pipelines must be configured

## Prerequisites
- `backend/AGENTS.md` (exists)
- `frontend/AGENTS.md` (exists)
- `public/AGENTS.md` (exists)
- `docs/CONTRIBUTING.md` (exists)