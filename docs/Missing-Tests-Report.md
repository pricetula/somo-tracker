Missing Tests Report — Somotracker Backend

Contract baseline (from backend/AGENTS.md §5)

Every functional domain package under internal/ MUST ship both:

1.  \*\_service_test.go — unit tests, no network/disk/DB, in-memory mocks. Runs under go test -short. Completes in ms.
2.  \*\_repository_test.go — integration tests against a live Postgres instance (verified: existing tests use pgxpool + testcontainers-go with
    postgres:16-alpine).

handler_test.go is not contractually required but is the convention in ~half the codebase; flagged as optional.

cmd/api/main.go has no tests (acceptable — wiring/fx-only entry point, but flagged).

> **Status (June 2026):** Items 1 & 2 below have been resolved since this report was first written. See the "Resolution log" at the bottom for the latest state.
> Items 3, 4, and the advisory section remain in-flight.

────────────────────────────────────────────────────────────────────────────────

Critical gaps (contract violations)

### 1. internal/reports — missing repository_test.go

- Has domain.go, handler.go, module.go, service.go, service_test.go.
- service_test.go exists but service.go likely depends on a Repository interface — confirm whether it's currently injected (good) or whether the service
  layer exercises SQL directly (would also be a layering violation per AGENTS.md §1).
- Action: Add reports/repository_test.go against a live Postgres testcontainer. Verify the service-layer mock pattern (interface declared consumer-side
  per AGENTS.md §3).

### 2. internal/auth — missing repository_test.go

- Has repository.go (so SQL exists) but only integration_test.go + service_test.go.
- integration_test.go exercises Stytch error scenarios and uses httptest, not a live DB. It is not a substitute for the contractually-required
  repository_test.go.
- Action: Add auth/repository_test.go against live Postgres. Exercise every method declared in repository.go (insert/find/lookup flows, sentinel-error
  mapping: ErrNotFound from sql.ErrNoRows, etc.).

### 3. internal/imports — missing repository_test.go

- Has repository.go (SQL lives here) but only integration_test.go + service_test.go.
- integration_test.go does spin up a real Postgres + Redis (good), but its scope is the service/handler/orchestration layer (asynq workers, CSV imports),
  not the repository SQL surface itself.
- Action: Add imports/repository_test.go covering each query in repository.go (e.g., upsert student/teacher cohorts, RLS-aware reads, dedupe behaviour).
  Reuse the testcontainer pattern from peers.

### 4. internal/resources — empty directory

- Directory exists but contains zero Go files. Either delete the directory (it shouldn't exist as a placeholder) or scaffold the package
  (domain/repository/service/handler + tests).
- Action: Confirm intent. If unused, remove. If planned, file an issue.

> **Resolution:** Per owner decision (June 2026), the empty directory is **kept as a placeholder for future work**. No tests required. Tracked as an
> outstanding "scaffold" ticket outside this report's scope.

────────────────────────────────────────────────────────────────────────────────

Non-functional gaps (non-contractual but conventional)

### Missing handler_test.go (13 packages)

These all have handler.go but no HTTP-level test suite. AGENTS.md doesn't mandate them, but coverage is uneven across the codebase:

- internal/attendance
- internal/behavior
- internal/cbcstreams
- internal/cbctimetableslots
- internal/cohortpositions
- internal/health
- internal/imports
- internal/parents
- internal/reports
- internal/teacherdeliverysummaries
- internal/teacherperformance
- internal/teachers
- internal/teacherworkloadsummaries
- internal/timetablestructucstructurestructure (typo check: timetablestructure)

Action: Out of contract — decide project-wide. Either lower to "advisory" or add httptest-based handler tests using a fake Service. Many peers
(academicyears, assessments, billing, members, …) already ship handler tests, so adding these would normalise coverage.

### Missing worker tests (3 packages)

worker.go exists in:

- internal/assessments/worker.go
- internal/attendance/worker.go
- internal/cbctimetableslots/worker.go

These contain background-job logic (asynq-style, presumably). No worker_test.go exists for any of them. While not contractually required, workers are a
frequent source of regressions and ideally warrant unit tests with mocked repositories + a fake asynq client.

### cmd/api/main.go — no tests

Wiring/fx-only. Defensive priority is low. Action: none required, but a smoke test that boots the fx container with mocked config would catch fx.Provide
misregistrations (which AGENTS.md §3 explicitly forbids).

### internal/database/migration_test.go — exists, confirm scope

Not a domain package — this is the migration runner. It has a test (good). No action.

────────────────────────────────────────────────────────────────────────────────

Quality observations on existing tests

After scanning the existing repository_test.go files:

- All 21 packages with repository_test.go correctly use pgxpool + testcontainers-go with postgres:16-alpine — consistent with the contract.
- Service tests all appear to follow the consumer-side interface + in-memory mock pattern (per AGENTS.md §3).

No issues to report on existing test infrastructure. The gap is purely coverage.

────────────────────────────────────────────────────────────────────────────────

Summary table

┌──────────────────────────┬─────────────────┬─────────────────┬──────────────┬──────────┐
│ Package │ service_test │ repository_test │ handler_test │ Severity │
├──────────────────────────┼─────────────────┼─────────────────┼──────────────┼──────────┤
│ reports │ ✅ │ ❌ │ ❌ │ CRITICAL │
├──────────────────────────┼─────────────────┼─────────────────┼──────────────┼──────────┤
│ auth │ ✅ │ ❌ │ ✅ │ CRITICAL │
├──────────────────────────┼─────────────────┼─────────────────┼──────────────┼──────────┤
│ imports │ ✅ │ ❌ │ ❌ │ CRITICAL │
├──────────────────────────┼─────────────────┼─────────────────┼──────────────┼──────────┤
│ resources │ n/a (empty dir) │ n/a │ n/a │ CRITICAL │
├──────────────────────────┼─────────────────┼─────────────────┼──────────────┼──────────┤
│ attendance │ ✅ │ ✅ │ ❌ │ advisory │
├──────────────────────────┼─────────────────┼─────────────────┼──────────────┼──────────┤
│ behavior │ ✅ │ ✅ │ ❌ │ advisory │
├──────────────────────────┼─────────────────┼─────────────────┼──────────────┼──────────┤
│ cbcstreams │ ✅ │ ✅ │ ❌ │ advisory │
├──────────────────────────┼─────────────────┼─────────────────┼──────────────┼──────────┤
│ cbctimetableslots │ ✅ │ ✅ │ ❌ │ advisory │
├──────────────────────────┼─────────────────┼─────────────────┼──────────────┼──────────┤
│ cohortpositions │ ✅ │ ✅ │ ❌ │ advisory │
├──────────────────────────┼─────────────────┼─────────────────┼──────────────┼──────────┤
│ health │ ✅ │ ✅ │ ❌ │ advisory │
├──────────────────────────┼─────────────────┼─────────────────┼──────────────┼──────────┤
│ parents │ ✅ │ ✅ │ ❌ │ advisory │
├──────────────────────────┼─────────────────┼─────────────────┼──────────────┼──────────┤
│ teacherdeliverysummaries │ ✅ │ ✅ │ ❌ │ advisory │
├──────────────────────────┼─────────────────┼─────────────────┼──────────────┼──────────┤
│ teacherperformance │ ✅ │ ✅ │ ❌ │ advisory │
├──────────────────────────┼─────────────────┼─────────────────┼──────────────┼──────────┤
│ teachers │ ✅ │ ✅ │ ❌ │ advisory │
├──────────────────────────┼─────────────────┼─────────────────┼──────────────┼──────────┤
│ teacherworkloadsummaries │ ✅ │ ✅ │ ❌ │ advisory │
├──────────────────────────┼─────────────────┼─────────────────┼──────────────┼──────────┤
│ timetablestructure │ ✅ │ ✅ │ ❌ │ advisory │
├──────────────────────────┼─────────────────┼─────────────────┼──────────────┼──────────┤
│ all others (12 pkgs) │ ✅ │ ✅ │ ✅ │ ok │
└──────────────────────────┴─────────────────┴─────────────────┴──────────────┴──────────┘

────────────────────────────────────────────────────────────────────────────────

Recommended next steps

1.  Immediate — add repository_test.go to reports, auth, imports. Three packages. The pattern is fully established (pgxpool + testcontainers + the shared
    startPG(t) helper that already exists in members/repository_test.go and could be lifted into a testutil package to avoid ~21-line copy-paste).
2.  Immediately after — decide on internal/resources (delete or scaffold).
3.  Decide policy — handler tests and worker tests: required or advisory?
4.  Optional
