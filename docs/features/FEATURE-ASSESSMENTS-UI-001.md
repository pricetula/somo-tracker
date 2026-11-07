---
id: FEATURE-ASSESSMENTS-UI-001
title: Assessment UI and API Behavior
status: draft
depends_on:
  - FEATURE-ASSESSMENTS-001
touches:
  - backend
  - frontend
  - api
---

# Assessment UI and API Behavior

## Goal
Translate the authoritative assessment spec (`FEATURE-ASSESSMENTS-001`) into implementable frontend screens, backend API behavior, error-code usage, and per-layer test coverage for the MVP.

## Out of scope
- Database migrations and implementation work authorized by the spec (design spec only; no migrations authorized by the spec alone).
- Offline queue implementation (deferred; idempotency contract remains).
- Swahili launch, guardian notifications, official KNEC/NECTA data exchange.
- Strand/sub-strand breakdowns, quizzes, custom formulas.

## Domain references
- docs/features/FEATURE-ASSESSMENTS-001.md — authoritative spec (§11 permissions, §13 API, §16 frontend, §19 acceptance criteria).
- docs/domain/glossary.md — Assessment, Assessment Result, Computation Policy, Term Result, Report Card, Working Scale, Missing-Data Rule, Rounding Rule.
- docs/domain/model.md — assessment entity relationships.
- docs/api/openapi.yaml — generated API contract authority; backend writes OpenAPI via swag from handler annotations.
- docs/domain/rules/RULE-001-permissions.md — permission matrix reference.
- backend/AGENTS.md — BE-001 … BE-045.
- frontend/AGENTS.md — FE-001 … FE-027, auth addendum.

## Acceptance criteria

- AC-1: Result-entry grid distinguishes `not_marked`, entered zero (`entered` + raw 0), `absent`, and `excused`; absent/excused and not-marked never display as zero and never silently become zero.
- AC-2: After a save, the grid shows the server-validated `normalized_percentage`, `working_value`, and `derived_grade_label`; no client-side conversion or display that bypasses server validation.
- AC-3: Partial saves supported with row-level errors; bulk upsert idempotent via `(tenant_id, actor_user_id, client_request_id)` and rejects key reuse with a different payload; stale writes return `RESULT_CONFLICT` instead of overwriting.
- AC-4: Term-result preview renders category aggregates, effective weights, selection (`all`/`last_n`), per-input include/exclude + reason, rounding outcome, and grade mapping; repeated deterministic computation yields identical values.
- AC-5: Report-card draft clearly shows incomplete subjects; lines render from immutable snapshot data, never live grading labels.
- AC-6: Guardian view serves only the latest published report-card version for a linked learner with `can_view_results = true`; drafts/reviewed/approved/superseded/audit data are unreachable.
- AC-7: Lifecycle transitions are separate authorized actions with required reasons/audit: assessment `draft → open → finalized`, `finalized → reopened/cancelled` (admin/head teacher); report card `draft → reviewed → approved → published`, plus revision creating a new version.
- AC-8: Every new endpoint uses the canonical error envelope `{ code, message, errors?, requestId }` with mapped status codes (400/401/403/404/409/422/429/500) and stable SCREAMING_SNAKE codes (e.g. `ASSESSMENT_NOT_OPEN`, `RESULT_CONFLICT`, `REVISION_REQUIRED`, `REPORT_CARD_NOT_READY`).
- AC-9: List endpoints use keyset pagination `{ items, nextCursor }` with `limit` (default 50, max 100), whitelisted `sort`, opaque cursor tied to sort+filters; single reads return the bare object with no envelope.
- AC-10: Every new endpoint is covered by tenant/school isolation tests and per-role permission tests per BE-036 and FE-027.

## API changes

- Reference: `FEATURE-ASSESSMENTS-001` §13 (API contract) and §14/§15/§19 (integrity, RLS, acceptance criteria).
- Proposed routes to add to `docs/api/openapi.yaml` (swag annotations in Go handlers per BE-022 … BE-023):
  - Grading schemes: `GET /api/v1/schools/{schoolId}/grading-scheme-families`, `POST .../families`, `GET /api/v1/grading-scheme-families/{familyId}/versions`, `POST .../families/{familyId}/versions`, `POST /api/v1/grading-schemes/{schemeId}/activate`, `POST /api/v1/grading-schemes/{schemeId}/retire`.
  - Computation policies: `GET /api/v1/schools/{schoolId}/computation-policy-families`, `POST .../families`, `POST /api/v1/computation-policy-families/{familyId}/versions`, `POST /api/v1/computation-policies/{policyId}/activate`, `PATCH /api/v1/schools/{schoolId}/assessment-settings`.
  - Subject offerings & assessments: `GET|POST /api/v1/subject-offerings/{offeringId}/assessments`, `GET /api/v1/assessments/{assessmentId}`, `PATCH /api/v1/assessments/{assessmentId}`, `POST .../open`, `POST .../finalize`, `POST .../reopen`, `POST .../cancel`.
  - Results: `GET /api/v1/assessments/{assessmentId}/results`, `PUT /api/v1/assessments/{assessmentId}/results/{enrollmentId}`, `POST /api/v1/assessments/{assessmentId}/results:bulk-upsert`.
  - Term results: `GET /api/v1/enrollments/{enrollmentId}/terms/{termId}/subject-results`, `GET /api/v1/term-results/{termResultId}`, `POST /api/v1/term-result-families/{familyId}/recompute`.
  - Report cards: `GET /api/v1/enrollments/{enrollmentId}/terms/{termId}/report-card`, `POST /api/v1/enrollments/{enrollmentId}/terms/{termId}/report-card`, `PATCH /api/v1/report-cards/{reportCardId}`, `POST .../review`, `POST .../approve`, `POST .../publish`, `POST .../revise`, `GET /api/v1/guardians/me/report-cards`.
  - Guardian endpoints: `GET /api/v1/guardians/me/report-cards`.
- Error codes (stable SCREAMING_SNAKE per BE-025, used in `errors[].code`):
  - `FORBIDDEN`, `NOT_FOUND`, `VALIDATION_ERROR`, `ASSESSMENT_NOT_OPEN`, `ASSESSMENT_ALREADY_FINALIZED`, `REVISION_REQUIRED`, `SCHEME_NOT_ACTIVE`, `SCHEME_SCALE_MISMATCH`, `POLICY_NOT_ACTIVE`, `POLICY_INVALID_WEIGHTS`, `ENROLLMENT_NOT_ELIGIBLE`, `RESULT_CONFLICT`, `IDEMPOTENCY_KEY_REUSED`, `TERM_RESULT_INCOMPLETE`, `REPORT_CARD_NOT_READY`, `REPORT_CARD_ALREADY_PUBLISHED`.
- Request/response conventions (per BE-024/BE-025):
  - JSON/camelCase field names; ids are UUID strings; timestamps RFC3339 UTC with `Z`; date-only `yyyy-MM-dd` strings; money as integer in minor units (KES); enum values SCREAMING_SNAKE; single reads return the bare object with no envelope; lists return `{ items, nextCursor }`; request DTOs validated via `go-playground/validator/v10`; business rules validated in services → `apperr`.
- Publication snapshot flow (BE-027): term-result computation wrapped in transaction; new version + explanation rows atomically persisted; freeze of referenced versions on publish; job enqueued after commit.

## Backend notes

- BE-006 handlers: thin — bind input, read session, call service, map DTO response; no SQL/svc/pgx/Fiber imports below handler.
- BE-012 transactions: one per service call via `TxManager.Run`; stores take connection from context; jobs enqueued after commit, not held across network.
- BE-016/BE-017: every query tenant-scoped; school scoped from session; cross-tenant + nonexistent → identical 404; never accept school/tenant ID from client.
- BE-020 authz: `authz.Require(perm)`; permission matrix from `platform/authz`; scope from `authz.ScopeFor(ctx, perm)` passed to store.
- BE-027 asynq jobs: idempotent payloads with `tenant_id` + `school_id`; recompute/finalize jobs after commit; every job has `client_request_id` for dedup.
- BE-030 rate limits: keyed by user, not IP; before-auth IP keys apply; imports/PDFs/exports have per-tenant concurrency caps.
- BE-035: reference FR/NFR IDs in PRs/commits/tests; when silent → open question, not assumption.
- BE-039: `updated_at` maintained via DB trigger (`trigger_set_timestamp`), not only app code.
- BE-041: soft delete via `deleted_at` + partial index; prefer hard delete only when domain allows.
- BE-043: status/check fields use ENUM, not TEXT CHECK.
- BE-045: no JSON schema validation unless specified; `'{}'` is sufficient.
- Result model→API mapping (camelCase):
  - `%` score: `rawScore`, `normalizedPercentage`, `workingValue`; `derivedGradeLabel`; `status` enum; `gradingSchemeId`; `enteredBy`, `enteredAt`.
  - Rubric: `rawBandLabel` required; `rawScore` NULL; computed values nullable on display, required after validation.
  - Result state: `not_marked` → all numeric null; `entered` → `rawScore` non-null; `absent`/`excused` → all numeric null, non-zero reason text.
  - Bulk upsert response: `saved[]` with persisted canonical values + `errors[]` + `conflicts[]`; each row has stable enrollment ID + machine-readable error code + updated-at timestamp.
  - Term result inputs: per-assessment row with `workingValue` nullable, `included` boolean, `drop_reason` enum/text; unique `(term_result_id, assessment_id)`.
  - Report card line: immutable snapshot — `finalGrade`, `workingValue`, `roundedWorkingValue`, `finalGradeLabel`, `subjectNameSnapshot`, `gradingLabelsSnapshot jsonb`.

## Frontend notes

- FE-001 feature layout: `features/assessments/{index.ts, types.ts, schemas.ts, api/, hooks/, components/, pages/}`; named exports only; no `export *`; features never import other features' internals.
- FE-007 active school: never contain school segment in URL (`/students`, never `/[school]/students`); `useActiveSchool()` prefetches `schoolId` from backend; query keys are `[schoolId, 'assessments', scope, params]` for cache isolation; cache cleared when school changes.
- FE-008 API types: generated from OpenAPI 3.1 into `src/features/assessments/api/generated/` via `pnpm generate:api`; Zod schemas also generated; frontend MUST use generated types (snake_case mapped to camelCase); backend is contract authority.
- FE-009 pagination: lists use `useInfiniteQuery` with `getNextPageParam` reading `nextCursor`; null means last page; `@tanstack/react-virtual`; query keys `[schoolId, 'assessments', scope, params]`; limit default 50 max 100; sort whitelisted (e.g. `name_asc`); `maxPages` set; row heights fixed; staleTime 0 gcTime 0 for sensitive lists (results/behavior).
- FE-013 app shell: shadcn `sidebar-07`; header contains only `SidebarTrigger`, a vertical `Separator` and a dynamic `Breadcrumb`; page title and primary actions go in shared `<PageHeader>` in the content area.
- FE-014 intercepting + parallel routes: add/edit/view use `@modal/(.)add`, `(.)[id]`, `(.)[id]/edit` routes; each also has a full-page route rendering the same page component; every `@modal` slot has `default.tsx` returning `null`.
- FE-015 controls: no native `select/input/textarea/button/dialog/table/input[type=date]`; use shadcn counterparts; dates via `Popover` + `Calendar`.
- FE-016 semantic Tailwind: `bg-background`, `text-foreground`, `bg-muted`, `text-destructive`, `border-border` only; no hex/palette colors; status colors (EE/ME/AE/BE/attendance) from `globals.css` tokens via one style map; colors never alone; separate sections with spacing.
- FE-017: no wrapper elements without layout/semantic purpose; use fragments.
- FE-019 DataTable: `<DataTable>` (`@tanstack/react-table` + shadcn `Table`) with server-side pagination, sorting and filtering; only exemption: `RollCallList`. Row actions and bulk operations use confirmations + conflict handling per AC-3.
- FE-020 forms: `react-hook-form` + `zodResolver` + shadcn `Form` parts (`FormField`, `FormItem`, `FormLabel`, `FormControl`, `FormMessage`); no `useState` form bindings.
- FE-023 dates: `date-fns` + `@date-fns/tz` Africa/Nairobi; date-only fields `yyyy-MM-dd` strings, never `Date` objects.
- FE-024 performance levels: generated enum with one label map; CBC terms exactly (learning area, strand, sub-strand); money integer minor units formatted via `formatKes()`.
- FE-026 FR/NFR IDs in PRs/commits/tests; unstated behavior becomes an Open Question, not an assumption.

## Verification

- AC-1..AC-10 unit tests per role + tenant isolation; run `make compliance` and `pnpm test`; new endpoints have contract tests vs generated spec; RLS + permission matrix tests use testcontainers Postgres.

## Open questions

- What existing platform route pattern covers school-scoped resources? (BE-016 implies school scoping from session; need to reconcile `/api/v1/schools/{schoolId}/...` convention with the codebase's `/api/v1/...` base and resource paths.)
- Offline queue: deferred or included in MVP? (spec §0.2 excludes; but FE: offline writes via client-generated `id` + `client_request_id` idempotency; if deferred, UI must not treat locally queued unsynced marks as finalized.)
- Number of assessment categories beyond `formative`/`summative` in v1? (spec §0.2 excludes quizzes/question banks; category enum in assessment schema currently `formative|summative` — confirm if additional categories needed.)
- Rounding display: nearest vs down differences shown to teacher vs guardian? (spec §8.6 defines exact behavior; UI must differentiate school-level rounding preference.)
- How does existing `openapi.yaml` convention handle `@BasePath /api/v1` with Go handlers? (swag annotations per BE-022; ensure new routes' annotations generate the proposed paths; regenerate via `make generate`.)