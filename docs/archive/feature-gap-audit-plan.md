# Feature & Gap Audit Plan — Somotracker

**Purpose:** Systematic discovery of features needed and gaps to address for planning, excluding auth and bulk job imports.

**Status:** Draft
**Created:** 2026-10-15
**Owner:** Platform team

## Context

Project is document-led. Code must follow docs. Existing inventory:
- `docs/features/` — formal specs
- `docs/gaps/` — known missing/broken work
- `docs/planning/` — pre-spec analysis
- Frontend: Next.js App Router, feature modules under `src/features/`
- Backend: Fiber REST API, routes in `backend/internal/api/router.go`

Golden rule: Never commit. Changes remain unstaged.

## Approach

### Phase 1 — Inventory

1. **Frontend routes**
   - Enumerate all `src/app/(dashboard)/*/page.tsx` and modal routes
   - Map each route to its feature module via imports in `page.tsx`
   - Record UI state: Implemented / Placeholder / Skeleton

2. **API surface used by frontend**
   - Parse `frontend/src/lib/api/*.ts` for `api.get/post/patch/delete` calls
   - Extract endpoint paths, HTTP method, feature area
   - Build list of frontend-expected endpoints

3. **Backend API reality**
   - Parse `backend/internal/api/router.go` protected routes
   - List method + path + handler struct
   - Note which handlers are `nil` or stubbed

4. **Docs coverage**
   - List `docs/features/FEATURE-*.md` and their status
   - List `docs/gaps/GAP-*.md` and severity/status
   - Map docs to feature modules

### Phase 2 — Gap Analysis Matrix

For each feature area, produce:

| Feature Area | Frontend Page | Feature Module | Frontend Expected API | Backend Handler Exists | Schema Ready | Docs Feature | Docs Gap | Status |
|---|---|---|---|---|---|---|---|---|

Status categories:
- **Complete:** UI + API + Schema + Docs
- **Partial UI:** UI exists, API missing/placeholder
- **Partial API:** API exists, UI missing/placeholder
- **Missing:** Neither UI nor API
- **Placeholder:** UI shows static/placeholder component
- **Spec Only:** Feature spec exists, no code

### Phase 3 — Prioritization

Generate prioritized gap list for planning:
- Critical blockers: timetable class assignment UI, conflict detection, student detail placeholder
- High: student schema fields, listing enhancements, student timetable view
- Medium: guardian linking, notes, curriculum deep links

Output artifacts:
- `docs/planning/feature-gap-audit-2026-10-15.md` — full matrix
- Update `docs/gaps/README.md` index
- Create new `GAP-*.md` files per template for undiscovered gaps
- Link gaps to planning docs and feature specs

## Feature Areas to Audit

Exclude: auth, bulk job imports

Include:
- Students — list, detail, add, enrollments, schema
- Guardians — list, detail, invite, linking
- Teachers — list, detail, invite, summary
- Admins — list, detail, invite
- Finance — list, detail, invite
- Classes — list, detail, create, enrollments
- Timetable — list, templates, class slots, assign, substitutions, conflict detection
- Attendance — sessions, create
- Curriculum — curriculum, subjects, topics, sub-topics
- Events — list, create, update, delete
- Rooms — list, create, update, delete
- Streams — list, create, update, delete
- Academic Periods — list, create
- Grades — list
- Schools — list, create, set active
- Nurses — page exists, no feature module? Check

## Automation Helpers

- `grep -r "api\." frontend/src/lib/api --include="*.ts" | grep "/api/"` → endpoint list
- `grep -r "protected.Get\|protected.Post\|protected.Patch\|protected.Delete" backend/internal/api/router.go` → backend routes
- `find frontend/src/app/\(dashboard\) -name page.tsx` → route list
- Cross-reference via script to CSV

## Next Steps

1. Approve approach
2. Generate initial inventory matrix
3. Review with team, open new gaps as needed
4. Move accepted gaps to feature specs

## References

- `docs/CONTRIBUTING.md`
- `docs/planning/student-module-roadmap.md`
- `docs/gaps/README.md`
- `docs/features/README.md`
