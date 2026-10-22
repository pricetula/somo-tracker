# Contributing to Somotracker — Document-Led Development

This project is document-led. Code follows docs, never the other way around.

**Bot Factory:** See `docs/REVAMP-PLAN.md`. PRs are enforced by bots: docs-bot, api-bot, security-bot, migration-bot, ui-bot. Run `make compliance` locally before pushing.

## Folder intent

```
docs/
  CONTRIBUTING.md        # this file
  ROADMAP.md            # high-level product roadmap
  reference/            # immutable reference info
    backend_schema.md
  planning/             # active roadmaps only
  features/             # formal feature specs
    README.md
    FEATURE-*.md
  gaps/                 # known missing / broken work
    README.md
    GAP-*.md
  templates/            # gap.md, feature-spec.md, adr.md
  adr/                  # architecture decision records
  seed/                 # seeding / authoring data, not documentation
    cbc/*.json
    teachers.csv
  archive/              # moved obsolete checklists, audits, stale planning
```

A non-dev should be able to understand what each folder contains from its name alone.

## Workflow

### 1. Discover → Gaps
When something is missing, broken or incomplete:
1. Create `docs/gaps/GAP-<slug>.md` from `docs/templates/gap.md`
2. Fill Area, Severity, Status, Description, Current vs Expected, Impact, Repro steps
3. Update `docs/gaps/README.md` index

Gaps are the backlog.

### 2. Plan → Planning
When a gap needs investigation before a spec:
1. Create analysis in `docs/planning/` e.g. `gap-analysis.md`, `roadmap.md`, checklist
2. Link related gaps and features
Planning docs are working material, not formal specs.

### 3. Specify → Features
When a gap is accepted for work:
1. Copy `docs/templates/feature-spec.md` → `docs/features/FEATURE-<slug>.md`
2. Problem, Solution, User Stories, Acceptance Criteria, API contracts, UI notes
3. Link related gaps and planning docs
4. Update `docs/features/README.md` status: Proposed → Accepted → In Progress → Done

If architecture is affected, create an ADR in `docs/adr/` first.

### 4. Implement
* Implement to the spec exactly
* Backend error contract: `backend/internal/api/errors.go`
* Frontend error handling: `frontend/src/lib/api/client.ts`
* Agent assists with commit/push upon explicit user instruction.

### 5. Small changes to existing features
* **Tweak < 1 day, no new API:** Update existing `FEATURE-*.md` with a changelog note
* **New field / endpoint:** Open a Gap → short planning note → amend Feature spec with Addendum
* **Breaking change:** New Feature spec + ADR

### 6. Reference updates
* Update `docs/reference/backend_schema.md` when schema changes
* Keep all links relative and start with `docs/`

## Naming & linking conventions
* Gaps: `GAP-<kebab-slug>.md`
* Features: `FEATURE-<kebab-slug>.md`
* ADRs: `ADR-YYYY-MM-DD-<title>.md`
* Always link using `docs/<folder>/filename.md`

## Principles
* Document first, code second
* Every error must be returned up the call stack with context added OR logged and acted upon. Never both, never neither.
* No dumb back buttons in app pages
* Keep `docs/` self-explanatory. No data files in doc folders except `docs/seed/`

Questions? Open a planning doc in `docs/planning/` and link related gaps/features.
