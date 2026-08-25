# Backend Error Handling — Remediation Plan

**Companion to:** `error-handling-audit.md`
**Status:** Draft — Phase 0 not yet started
**Owner:** (assign)

---

## How to use this document

This is the canonical source of truth for the remediation work. Each phase
below is written to be handed to a coding agent (or engineer) as a
standalone task — reference **this file plus `error-handling-audit.md`**
in every prompt, since the agent will not have memory of prior sessions.

Suggested prompt template for any phase:

> Read `error-handling-audit.md` and `docs/error-handling-remediation-plan.md`
> in this repo. Implement **Phase N** as scoped below. Do not start work
> belonging to any other phase.

Mark each phase's checkbox and status as it lands so the plan stays accurate.

---

## Guiding principle

Every issue in the audit traces back to one root cause: there is no single,
hard-to-misuse path for (a) returning an HTTP error and (b) closing a
database transaction. Fix those two primitives once as additive code, then
migrate call sites in small, independently-shippable slices. Never mix
"fix a bug" and "change the pattern" in the same PR.

---

## Phase 0 — Foundation (additive only, no existing files touched except wiring)

**Status:** ☐ Not started

**Goal:** Ship the new shared primitives. Zero behavior change to any
currently-passing route, except Fiber's built-in 4xx errors will start
returning canonical JSON instead of HTML/text (audit issue #4) — that's
the one intended side effect.

**In scope — new files:**

- `internal/xerrors/xerrors.go` — `Error` type (Code, Message, Fields,
  Status, wrapped cause), constructors per the canonical code table in the
  audit's Appendix (`not_found`, `invalid_input`, `unauthorized`,
  `forbidden`, `conflict`, `already_exists`, `unprocessable_entity`,
  `device_fingerprint_mismatch`, `request_canceled`, `timeout`,
  `internal_error`), `WithField()`, `Wrap()`.
- `internal/middleware/errors.go` — rewritten `HTTPError(c, err)`:
  `errors.As` into `*xerrors.Error`, fallback to Internal/500 + server-side
  log if not, log all 5xx regardless of origin, write
  `{code, message, errors, request_id}`, pull `request_id` from whatever
  correlation pattern already exists in the repo (inspect first, don't
  assume a package name).
- `internal/database/tx.go` — `WithTx(ctx, pool, fn)` per the audit's
  proposed implementation: recover-safe deferred rollback-or-commit,
  `context.WithoutCancel(ctx)` for cleanup, rollback failures logged at
  Warn (except `pgx.ErrTxClosed`), commit failures wrapped via
  `xerrors.Wrap` and returned.
- `internal/httpctx/httpctx.go` — `TenantID(c)`, `SchoolID(c)` safe
  accessors returning `xerrors.Unauthorized` / `xerrors.InvalidInput`
  instead of panicking or silently zero-valuing.

**In scope — existing file modification (only this one):**

- `cmd/api/main.go` — wire `middleware.HTTPError` as the sole
  `fiber.Config.ErrorHandler`; delete the branch that falls through to
  `fiber.DefaultErrorHandler` for status codes under 500.

**Explicitly out of scope:** `internal/auth`, `internal/students`,
`internal/attendance`, `internal/behavior`, any repository file.

**Acceptance criteria:**

- [ ] `go build ./...` passes
- [ ] Existing test suite passes unchanged
- [ ] New table-driven tests for `xerrors.Wrap` (double-wrap preserves
      original code/status) and `HTTPError`'s fallback-to-Internal path
- [ ] Manually confirm a route hitting Fiber's built-in 404 now returns
      JSON, not HTML

---

## Phase 1 — Stop the silent transaction failures

**Status:** ☐ Not started
**Depends on:** Phase 0

**Goal:** Fix audit's Critical Issue #1 — `auth/service.go`'s 5 occurrences
of `defer func() { _ = finish() }()`.

**In scope:**

- `internal/auth/service.go` only (lines ~332, 407, 661, 698, 845 per audit)
- Replace the manual `finish()` closure + swallowed defer with
  `database.WithTx`, preserving all existing query logic inside the
  closures passed to it.

**Explicitly out of scope:** any other file. This is a single-file,
high-severity, low-blast-radius change — keep it that way.

**Acceptance criteria:**

- [ ] No `_ = finish()` or equivalent swallowed transaction error remains
      in `auth/service.go`
- [ ] All existing auth tests pass
- [ ] Commit/rollback failures are now logged (verify with a forced-failure
      test if the test harness supports it, e.g. killing the connection
      mid-transaction)

---

## Phase 2 — Repository rollback logging

**Status:** ☐ Not started
**Depends on:** Phase 0

**Goal:** Fix audit's Critical Issue #2 across repositories. One PR per
package to keep reviews small.

**In scope (one sub-phase per file, can be parallelized across PRs):**

- [ ] `internal/attendance/worker.go` (line ~196)
- [ ] `internal/attendance/repository.go`
- [ ] `internal/behavior/repository.go`
- [ ] `internal/database/tenant.go`
- [ ] `internal/academicyears/repository.go`
- [ ] `internal/assessments/worker.go`

**Approach:** Prefer migrating fully to `database.WithTx` where the
transaction shape allows it. Where a full refactor isn't practical in one
pass, at minimum replace `_ = tx.Rollback(...)` with logged rollback per
the audit's suggested pattern (`rbErr != pgx.ErrTxClosed` check).

**Acceptance criteria (per file):**

- [ ] No bare `_ = tx.Rollback(...)` remains
- [ ] Existing tests for that package pass

---

## Phase 3 — Collapse the three error-response formats into one

**Status:** ☐ Not started
**Depends on:** Phase 0

**Goal:** Fix audit's High Severity Issues #3, #5, #6, #7. Every handler
returns `error` (ideally `*xerrors.Error`) and calls `middleware.HTTPError`
— no handler calls `c.Status(...).JSON(...)` directly anymore.

Order (smallest/most contained first, largest last):

### 3a. `internal/students/handler.go`

**Status:** ☐ Not started

- Remove the custom `writeError` function and `errorResponse` type
  (audit lines 124–131).
- Replace all ~30 call sites with returns of `*xerrors.Error`, letting
  the caller's `middleware.HTTPError` handle serialization.
- Fixes the `Errors` vs `errors` field-casing bug as a side effect.

### 3b. `internal/auth/handler.go`

**Status:** ☐ Not started

- Fix the 2 direct-JSON switch-school endpoints (audit lines ~419/425).
- Fix the 401→400 status bug for missing `active_school_id` in the same
  pass (audit issue #7) — same lines, same PR, since separating them adds
  no safety and doubles review overhead.

### 3c. `internal/behavior/handler.go`

**Status:** ☐ Not started

- Replace ~10 direct-JSON occurrences with `*xerrors.Error` returns.

### 3d. `internal/attendance/handler.go`

**Status:** ☐ Not started

- Largest surface (~15 occurrences). Do last, once the pattern is proven
  on 3a–3c.
- Includes the second `active_school_id` 401→400 fix (audit line 237).

**Acceptance criteria (per sub-phase):**

- [ ] No direct `c.Status(...).JSON(...)` calls remain in the file
- [ ] All error codes match the canonical table (`invalid_input`, not
      `VALIDATION_ERROR`, etc.)
- [ ] `request_id` present in responses (verify via integration test or
      manual curl)
- [ ] Existing tests pass; add a test for the 401→400 fix where applicable

---

## Phase 4 — Guardrails

**Status:** ☐ Not started
**Depends on:** Phase 3 complete (all sub-phases)

**Goal:** Prevent regression back to the old patterns now that "correct"
is well established across the codebase.

**In scope:**

- [ ] Add `wrapcheck` to `.golangci.yml`, enforcing that returned errors
      are wrapped (audit issue #8). Fix any newly-surfaced violations in
      repositories that weren't touched in Phase 2.
- [ ] Add a CI check (grep-based is fine to start) that fails the build
      on any `c.Status(...).JSON(...)` outside `internal/middleware/`.
- [ ] Sweep unsafe type assertions (audit issue #9) — replace remaining
      `c.Locals(...).(string)` call sites with `httpctx.TenantID` /
      `httpctx.SchoolID` from Phase 0.

**Acceptance criteria:**

- [ ] `golangci-lint run` passes with `wrapcheck` enabled
- [ ] CI fails on a deliberately introduced direct-JSON test violation
      (verify the check actually works before merging it)
- [ ] No unchecked type assertions on `c.Locals` remain in handler code

---

## Final verification (after Phase 4)

Re-run the audit's original checklist:

- [ ] All 4xx/5xx responses return canonical JSON format
- [ ] All responses include `request_id` when available
- [ ] Error codes match xerrors sentinels
- [ ] Transaction commit/rollback errors are logged
- [ ] Fiber 404/405 return JSON, not HTML
- [ ] Frontend error handling works without special cases
