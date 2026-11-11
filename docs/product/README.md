# Phase 1 — Product Knowledge

Vision, users, capabilities, and terminology for Somotracker.

## Status

draft — bootstrapped from existing code and AGENTS.md files. No human sign-off yet.

## Vision

Somotracker is a school management system for East African schools (Kenyan school-management domain). It is a B2B multi-tenant application: each school (or school group / tenant) manages its own users, students, academics, attendance, health, behavior, and fees.

Sources: root `AGENTS.md` ("Project: School management system for East African schools"), `README.md` ("School management system").

## Users

User roles (verified from `backend/migrations/00000000000003_core_domain.sql`, `user_role` enum):

| Role | Notes |
|------|-------|
| OWNER | Tenant-level role |
| ADMIN | |
| HEADTEACHER | |
| TEACHER | |
| GUARDIAN | In MVP scope (decision: include Guardian, not deferred to Phase 2) |
| FINANCE | |
| PLATFORMOPERATOR | |

Permissions are checked per permission (`authz.Require(perm)`), never via role lists (`BE-020`). The permission matrix is data in `platform/authz` and must be implemented (BE-020/BE-021); it is not a placeholder to be left unimplemented (see `backend/internal/platform/authz/authz.go` and `docs/specifications/registry.md`).

## Capabilities

### Implemented (backend handlers + frontend feature)

| Capability | Endpoints | Backend | Frontend |
|------------|-----------|---------|----------|
| Request magic link (Stytch Discovery) | `POST /api/v1/auth/request-magic-link` | `internal/identity/handler` | `features/auth` |
| Authenticate magic-link token | `POST /api/v1/auth/authenticate-token` | `internal/identity/handler` | `features/auth` |
| Logout / revoke session | `POST /api/v1/auth/logout` | `internal/identity/handler` | `features/auth` |
| Onboarding state (get) | `GET /api/v1/onboarding/current` | `internal/identity/handler` | `features/auth` |
| Onboarding state (update) | `PUT /api/v1/onboarding/current` | `internal/identity/handler` | `features/auth` |
| Complete onboarding / provision tenant | `POST /api/v1/onboarding/complete` | `internal/identity/handler` | `features/auth` |

Source: `docs/api/openapi.yaml` (OpenAPI 3.1, version 0.1.0), `backend/internal/identity/handler/handler.go`, `frontend/src/features/auth/`.

### Schema exists, no handlers yet (not MVP scope unless specified)

Tables exist for students, guardian links, and academic data (subjects, topics, subtopics, streams, classrooms, academic years, terms, enrollments), but no domain handlers/services exist for them outside identity. Source: `backend/migrations/0000000000000{4,5}.sql`, `backend/internal/` directory listing.

## Terminology

Domain terminology to use consistently (verified from root `AGENTS.md` security/domain principles and `frontend/AGENTS.md` FE-024):

| Term | Meaning |
|------|---------|
| Education system | Country-specific curriculum framework (e.g. Kenya CBC, Tanzania, Uganda); one or more per country |
| Subject | A learning area within an education system + class level (e.g. Mathematics, English) |
| Topic | A strand within a subject; has a `sequence_index` |
| Sub-topic | A sub-strand within a topic; has a `sequence_index` |
| Class level | A grade level within an education system |
| Stream | A class group (color/name) scoped to a school |
| KES | Kenyan shillings; money stored as integer minor units |
| Minor units | Smallest currency unit; e.g. `amountMinor` in JSON |

**Decision adopted:** the academic hierarchy is **subject → topic → sub-topic** (migrations `00000000000005_academic_domain.sql`), not "learning area / strand / sub-strand". Subjects are scoped to an `education_system_id` and `class_level_id`, so multiple countries' curricula (Kenya CBC, Tanzania, Uganda) can be added later under their own `countries` → `education_systems` rows. Each `school` references one `education_system_id`, fixing its curriculum.

Source: `backend/migrations/00000000000003_core_domain.sql` (countries, education_systems) and `00000000000005_academic_domain.sql` (subjects/topics/subtopics).


## Non-goals (MVP)

- Generic `PATCH` / `Idempotency-Key` header (`backend/AGENTS.md` naming conventions)
- Page numbers, total counts, OFFSET pagination (`BE-014`, `FE-009`)
