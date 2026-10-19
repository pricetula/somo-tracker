# Gap: Health / Nurse Domain — Orphan Schema with Zero Implementation

**Area:** backend | frontend
**Severity:** High
**Status:** Open
**Owner:** Platform team
**Created:** 2026-10-15
**Related Issues:** #

## Description
Migration `000021_add_health_behaviour_vaccination_and_communication.up.sql` creates six production tables (`incident_types`, `health_notes`, `behaviour_incidents`, `vaccination_records`, `communication_threads`, `thread_participants`) with RLS policies, yet no backend service, no REST handler, no feature module, and no working UI exist. The nav route `/dashboard/nurses` renders `div>coming soon</div>`.

This is the worst kind of debt: a committed schema promises a capability that is not delivered.

## Current Behavior
- `grep -rni 'health\|nurse\|behaviour\|vaccination' backend/internal/services/*.go` → 0 hits.
- `grep -rni 'health\|behaviour' backend/internal/api/*.go` → 0 hits.
- `frontend/src/app/(dashboard)/nurses/page.tsx` → `return <div>coming soon</div>;`
- `router.go` has no `/health` or `/nurses` protected route.
- DB migration 021 is applied in production builds (embedded via `db/embed_migrations.go`).

## Expected Behavior
Either (A) build the health/nurse module (screening register, behaviour log, vaccination tracking, guardian communication threads) with service + handler + feature module + page, or (B) drop the migration, drop the nav route, and document the deferral in an ADR.

No middle ground — partial schema with a stub page damages trust.

## Impact
- School nurses / health officers have no workspace.
- Vaccination and screening records cannot be captured (severe compliance gap in Kenyan school systems).
- Parent communication threads for health events (e.g., allergic reaction, illness) have no UI, despite the schema supporting them.
- The visible "Nurses" nav item promises functionality that does not exist.

## Reproduction Steps
1. Inspect `backend/db/migrations/000021_add_health_behaviour_vaccination_and_communication.up.sql` → 6 `CREATE TABLE` statements.
2. Inspect `backend/internal/services/` → no health-related `.go` files.
3. Visit `/dashboard/nurses` in the app → stub page.
4. Query `pg_enum`, `pg_tables` for `incident_types` → exists; query backend endpoints for `/api/health` or `/api/nurses` → 404.

## Proposed Solutions
- Option A: Build health module (service, handler, feature `features/health/`, dashboard route). Reuse existing bulk-job + SSE machinery for screening batches.
- Option B: Deferral. Delete migration 021 (down + up), remove `/nurses` nav, document in `docs/adr/GAP-health-deferred.md`.

## Decision
Pending. If not accepted for the next sprint, choose Option B to prevent further orphan-schema drift.

## Acceptance Criteria
- [ ] Decision recorded (build vs defer) in this gap.
- [ ] If build: `HealthService`, `HealthHandler`, `features/health/` module exist; routes registered; page is not a stub.
- [ ] If defer: migration 021 removed; `/nurses` link removed from shell/layout; ADR documents reason.
- [ ] No live `CREATE TABLE` without a matching service/handler/page pair.

## Links
- Migration: `backend/db/migrations/000021_add_health_behaviour_vaccination_and_communication.up.sql`
- Stub page: `frontend/src/app/(dashboard)/nurses/page.tsx`
- Related audit finding: assessment/docs audit (2026-10-15) — orphan schema section
