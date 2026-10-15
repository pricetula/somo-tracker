# Gap: Router Handlers Nil – Students / Enrollments / Invitations

**Area:** backend
**Severity:** Critical
**Status:** Open
**Owner:** 
**Created:** 2026-10-15
**Related Issues:** #

## Description
`backend/internal/api/router.go` registers protected routes for Students, StudentsImport, Enrollments and Invitation handlers, but `NewRouter` never initialises `Students`, `StudentsImport`, `Enrollments` and sets `AdminInvitation`, `TeacherInvitation`, `FinanceInvitation`, `GuardianInvitation` to `nil`. Any request to those routes will panic with a nil pointer dereference.

## Current Behavior
* `NewRouter` returns a `Router` with `Students` = nil, `StudentsImport` = nil, `Enrollments` = nil, invitations = nil.
* Routes are registered:
  * `GET /api/students`, `GET /api/students/summary`, `DELETE /api/students`
  * `POST /api/students/add`, `GET /api/students/jobs/:job_id`
  * `POST /api/classes/:id/enrollments`, `GET /api/classes/:id/enrollments`, `GET /api/students/unassigned`, `PATCH /api/enrollments/:id`, `DELETE /api/enrollments/:id`
  * `POST /api/admins/invitations`, `GET /api/admins/invitations/jobs/:job_id`, etc.
* First hit to any of these endpoints panics.

## Expected Behavior
All handlers are instantiated with their service dependencies in `NewRouter` and routes are safe to call.

## Impact
Core admin flows are broken: student listing/import, class enrollments, and all invitation bulk jobs are unusable in production. Blocks release.

## Reproduction Steps
1. Start API with current `NewRouter`
2. Authenticate and obtain session cookie
3. `GET /api/students`
4. Server panics: nil pointer dereference on `r.Students.ListStudents`

## Proposed Solutions
* Initialise `StudentsHandler`, `StudentsImportHandler`, `EnrollmentsHandler` in `NewRouter` using existing services, similar to `ClassesHandler` / `TeachersHandler`.
* Initialise invitation handlers with required Stytch/Asynq dependencies, or remove routes if not ready and document as deferred.

## Decision
Initialize handlers. Invitation handlers require additional dependencies – create constructors and wire via Fx provides.

## Acceptance Criteria
- [ ] `NewRouter` sets `Students`, `StudentsImport`, `Enrollments` non-nil
- [ ] Invitation handlers are either wired or routes removed with a clear TODO
- [ ] `go test ./...` passes, no nil panic on route registration
- [ ] Integration smoke test for `GET /api/students` returns 200

## Links
* Router: `backend/internal/api/router.go`
* NewRouter: `backend/internal/api/router.go` `NewRouter`
* Related gaps: `docs/gaps/GAP-student-detail-placeholder.md`
