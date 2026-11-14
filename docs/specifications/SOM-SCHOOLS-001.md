---
id: SOM-SCHOOLS-001
title: School Management — Create, View, Edit, Deactivate Schools
status: approved
author: agent
date: 2026-11-14
updated: 2026-11-14
fr_ids: [BE-006, BE-007, BE-009, BE-010, BE-012, BE-013, BE-014, BE-015, BE-016, BE-017, BE-020, BE-021, BE-025, BE-026, BE-036, FE-007, FE-008, FE-011, FE-013, FE-014, FE-019, FE-020, FE-021, FE-026, FE-027, FE-028, FE-029, FE-030, FE-031, FE-034, FE-035, FE-036, FE-037, FE-038, FE-039, FE-040, FE-041, FE-042, FE-043, FE-044, FE-045, FE-046, FE-047]
nf_ids: [BE-031, BE-032]
api_contract: yes
migrations: no
generated_files: []
---

## 1. Summary

School management lets a tenant **Owner** create, view, edit, and deactivate schools within their tenant. Routes live at `/schools/*` (no separate `/settings` page). Only the `OWNER` role has `school:write` and can access these routes; all other roles are blocked.

- `GET /schools` — list schools in tenant (paginated data table: name, code, country, education system).
- `GET /schools/add` — open a dialog form (intercepted route) to create a school; full-page form on hard load.
- `GET /schools/:id` — view school detail (sheet side panel on client nav; full page on hard load). The detail page contains an inline edit form for the school's details; no separate `/schools/:id/edit` route.
- `DELETE /api/v1/schools` — deactivate schools (soft delete, sets `deleted_at`) via confirmation dialog; follows bulk `ids` body pattern `{ "ids": ["uuid1", ...] }`.

This supersedes the `SOM-SIDEBAR-001` note that the school switcher "may move to a Settings page later" — there is no Settings page; school management lives at `/schools/*`.

## 2. Requirements (FR/NFR IDs with rationale)

| FR/NFR ID | Rationale |
|-----------|-----------|
| BE-006 … BE-015 | Standard CRUD pattern: UUIDv7 PKs, tenant-scoped queries, cursor pagination, Zod validation, error contract, soft delete via `deleted_at` |
| BE-016 | Every query tenant-scoped; cross-tenant returns identical 404 |
| BE-017 | Session context (tenant, role) loaded once; never accept school/tenant ID from client |
| BE-020 | Routes register with `authz.Require("school:write")` for mutating routes; `school:read` for list/view |
| BE-021 | Services obtain `Scope` from `authz.ScopeFor` and pass to store |
| BE-025 | DB enum values SCREAMING_SNAKE (`OWNER`); middleware role strings lowercase (`owner`) |
| BE-026 | Given/When/Then acceptance criteria; unstated behavior = Open Question |
| BE-036 | Backend tests with real PostgreSQL (testcontainers); cross-tenant 404 |
| FE-007 | Native fetch via `src/lib/api/client.ts`; base `/api/v1`; never send school ID header; no `[school]` URL segments |
| FE-008 | Generated API types from OpenAPI; snake_case wire → camelCase at adapter layer |
| FE-011 | Active school is backend state; `useActiveSchool()` |
| FE-013 | Shell sidebar-07; `<PageHeader>` for title/actions |
| FE-014 | Add/edit/view via intercepting + parallel routes in Dialog/Sheet; `@modal` default returns `null` |
| FE-019 | Cursor pagination; limit default 50 max 100; `q`/filters; no page numbers/OFFSET |
| FE-020 | `ApiError` shape `{code,message,errors?,request_id}`; global toast; 404 identical missing/cross-tenant |
| FE-021 | Shared composites never fetch; sidebar receives data via props |
| FE-026 | Reference FR/NFR IDs in PRs/commits/tests; GWT AC |
| FE-027 | Role-gated screen test per role; modal route soft-nav + hard-load tests |
| FE-028 | Create routes use `xxx/add` via intercepting + parallel routes; Dialog on client nav, full-page on hard load |
| FE-029 | View routes use `xxx/id`; Sheet side panel on client redirect; full-page on hard load |
| FE-030 | Update resource per-section on the detail page; no dedicated edit routes |
| FE-031 | Create forms: react-hook-form + zodResolver + shadcn Form; uniform styling |
| FE-034 | Delete confirmation via Dialog/AlertDialog; accept/cancel; cannot dismiss by overlay click |
| FE-035 | `<Link>` for all internal navigation; never native `<a href>` |
| FE-036 | Submit buttons disable on click; re-enable on response |
| FE-037 | Forms reset to pristine after successful submission |
| FE-038 | Empty state when lists return no data |
| FE-039 | Skeleton screens during data fetching |
| FE-040 | Search inputs debounce 300ms before API call |
| FE-041 | Active filters shown as removable chips above lists |
| FE-042 | Route-level error boundaries |
| FE-043 | Toast notifications for success/error/info/warning feedback |
| FE-045 | Accessibility: ARIA labels, keyboard navigation, focus management |
| FE-046 | Page titles and meta tags via TanStack Router `head` for SEO |
| FE-047 | Print styles: hide nav/sidebar when printing |
| BE-031 | Log IDs (request, tenant, school, user), route, status, latency, error code |
| BE-032 | Sentry: panics and 5xx only; scrub PII |

## 3. Domain model affected

**Entity: `schools`** (existing, migration `00000000000003_core_domain.sql` — no new migration)

| Column | Type | Notes |
|--------|------|-------|
| `id` | UUID PK (uuidv7) | |
| `tenant_id` | UUID → `tenants.id` | Tenant-scoped; ON DELETE CASCADE |
| `name` | TEXT NOT NULL | School display name |
| `code` | TEXT NOT NULL | Unique within tenant (`UNIQUE (tenant_id, code)`) |
| `country_id` | UUID → `countries.id` | ON DELETE RESTRICT |
| `education_system_id` | UUID → `education_systems.id` | ON DELETE RESTRICT |
| `created_at` | TIMESTAMPTZ | Default NOW() |
| `updated_at` | TIMESTAMPTZ | Default NOW() |
| `deleted_at` | TIMESTAMPTZ | Nullable; soft delete (deactivate) |

**Business rules**

- A school belongs to exactly one tenant (`tenant_id` from session, never client-supplied).
- `code` unique per tenant; create/edit must validate uniqueness (DB + API).
- Deactivate = soft delete (`deleted_at = NOW()`), not hard delete. Deactivated schools excluded from list and switcher.
- When a school is deactivated, only `deleted_at` is set; no cascading offboarding of users/memberships in MVP.
- Only `OWNER` may create/edit/deactivate. `school:read` granted to all authenticated roles (per SOM-AUTHZ-001), but `/schools/*` routes are owner-only per settled decision.

## 4. API contract

Changes to `docs/api/openapi.yaml` (backend-owned). New endpoints:

### `POST /api/v1/schools` — Create school

Auth: `authz.Require("school:write")` (owner only).

Request body:
```json
{
  "name": "string",
  "code": "string",
  "country_id": "uuid",
  "education_system_id": "uuid"
}
```

Response `201`: full school object (see `School` schema below).
Response `400`: validation error (duplicate `code`, missing fields).
Response `403`: FORBIDDEN (non-owner).
Response `409`: `code` already exists in tenant.
Response `401`/`500`: standard.

### `GET /api/v1/schools` — List schools

Existing `GET /api/v1/schools` returns `SchoolRef[] {id, name}`. The `/schools` list page calls a paginated list endpoint returning `School[]` with cursor pagination (FE-019) including fields: name, code, country, education system. Data table component uses `SOM-DATATABLE-001`. Filter out deactivated schools (`deleted_at IS NULL`).

### `GET /api/v1/schools/:id` — Get school

Auth: `authz.Require("school:read")`.

Response `200`: full school object.
Response `404`: not found or cross-tenant (identical).

### `PATCH /api/v1/schools/:id` — Update school

Auth: `authz.Require("school:write")` (owner only).

Request body (partial):
```json
{
  "name": "string",
  "code": "string",
  "country_id": "uuid",
  "education_system_id": "uuid"
}
```

Response `200`: updated school object.
Response `403`/`404`/`409`: standard.

### `DELETE /api/v1/schools` — Deactivate schools (soft delete, bulk `ids` body)

Auth: `authz.Require("school:write")` (owner only).

Follows the pending API contract pattern: `DELETE /api/v1/schools` with body `{ "ids": ["uuid1", "uuid2", ...] }`. The UI deactivates one school at a time (sends `{ "ids": [":id"] }`).

Sets `deleted_at = NOW()` on matched schools (soft delete). Deactivated schools are excluded from the list and switcher.

Response `200`: `{ "success": true }`.
Response `403`/`404`: standard.

### Schemas (add to `docs/api/openapi.yaml` `components/schemas`)

```yaml
School:
  type: object
  properties:
    id: { type: string, format: uuid }
    tenant_id: { type: string, format: uuid }
    name: { type: string }
    code: { type: string }
    country_id: { type: string, format: uuid }
    education_system_id: { type: string, format: uuid }
    created_at: { type: string, format: date-time }
    updated_at: { type: string, format: date-time }
    deleted_at: { type: string, format: date-time, nullable: true }

SchoolCreateRequest:
  type: object
  required: [name, code, country_id, education_system_id]
  properties:
    name: { type: string }
    code: { type: string }
    country_id: { type: string, format: uuid }
    education_system_id: { type: string, format: uuid }

SchoolUpdateRequest:
  type: object
  properties:
    name: { type: string }
    code: { type: string }
    country_id: { type: string, format: uuid }
    education_system_id: { type: string, format: uuid }
```

Frontend: regenerate types via `pnpm generate:api`; map snake_case → camelCase at `src/features/schools/api/` adapter layer (FE-008).

## 5. Workflow (Given/When/Then acceptance criteria)

### A. List schools

**Given** an authenticated `owner` with schools in their tenant
**When** they navigate to `/schools`
**Then** they see a table of schools (name, code, country, education system) with skeleton loading (FE-039), empty state if none (FE-038), search debounced 300ms (FE-040), active filters as chips (FE-041)

**Given** a non-owner user (headteacher, teacher, etc.)
**When** they navigate to `/schools`
**Then** they are redirected to `/unauthorized` (route hidden in sidebar per role)

### B. Create school (dialog)

**Given** an owner on `/schools`
**When** they click "Add school" (`<Link>` to `/schools/add`, FE-035)
**Then** a dialog opens (intercepted route, FE-028) with a form (react-hook-form + zodResolver, FE-031) for name, code, country, education system

**When** they submit
**Then** submit button disables (FE-036), on success form resets (FE-037), toast shows success (FE-043), list refreshes, dialog closes

**On hard load of `/schools/add`**
**Then** a full-page form renders the same component (FE-028); `@modal` default returns `null`

### C. View school detail

**Given** an owner on `/schools`
**When** they click a school row
**Then** a sheet side panel opens at `/schools/:id` (intercepted route, FE-029) showing school details

**On hard load of `/schools/:id`**
**Then** full page renders the same component (FE-029)

### D. Edit school (inline on detail page)

**Given** an owner viewing `/schools/:id`
**When** they click "Edit"
**Then** an inline edit form appears on the detail page, pre-filled with school data

**When** they submit
**Then** submit button disables (FE-036), on success toast (FE-043), list/detail refreshes, form resets (FE-037)

### E. Deactivate school

**Given** an owner on `/schools`
**When** they click "Deactivate" on a school row
**Then** a confirmation Dialog/AlertDialog opens (FE-034) with accept/cancel; cannot dismiss by clicking overlay

**When** they accept
**Then** `DELETE /api/v1/schools` is called with body `{ "ids": [":id"] }`; school is soft-deleted (`deleted_at` set); removed from list and switcher; toast confirms (FE-043)

**Given** a school is deactivated
**When** the school switcher lists schools
**Then** the deactivated school is not offered for switching

### F. Authorization

**Given** a request to `POST/PATCH/DELETE /api/v1/schools/*` without `school:write`
**Then** backend returns `403 FORBIDDEN` (BE-020)

**Given** a request for a school in another tenant
**Then** backend returns identical `404` (BE-016)

## 6. Implementation plan (files to touch, by application)

### Backend

1. `docs/api/openapi.yaml` — add `POST`, `GET /:id`, `PATCH /:id`, `DELETE /:id` under `/api/v1/schools`; add `School`, `SchoolCreateRequest`, `SchoolUpdateRequest` schemas
2. `make generate-swag` — regenerate `backend/docs/swagger.*`
3. `backend/internal/schools/` — new domain module:
   - `handler/schools.go` — handlers with `authz.Require("school:write")`/`authz.Require("school:read")`
   - `service/service.go` — business logic, tenant-scoped, transaction management via `TxManager.Run`
   - `store/store.go` — store methods mapping sqlc rows → domain types
   - `store/queries/schools.sql` — sqlc queries: `CreateSchool`, `GetSchool`, `ListSchools`, `UpdateSchool`, `DeactivateSchools`
4. `backend/internal/app/app.go` — register routes under `/api/v1/schools` with authz
5. `backend/internal/platform/authz/` — confirm `school:read`/`school:write` in permission matrix (already in SOM-AUTHZ-001)
6. `backend/internal/platform/cursor/` — use existing cursor engine for keyset pagination
7. Tests per BE-036 (testcontainers PostgreSQL): owner can CRUD; non-owner gets 403; cross-tenant gets identical 404; duplicate code → 409

### Frontend

1. `src/features/schools/` — new feature module (FE-001): `index.ts`, `api/`, `query-keys.ts`, components
2. Routes (FE-002):
   - `routes/schools.tsx` — list page (`<DataTable>` from `SOM-DATATABLE-001`, FE-019/FE-033)
   - `routes/schools/add.tsx` — create (intercepting `@modal/(.)add`, Dialog; full-page fallback, FE-028)
   - `routes/$schoolId.tsx` — view + inline edit form (intercepting `(.)[id]`, Sheet; full-page fallback, FE-029/FE-030)
3. `src/features/schools/api/` — verb-first API functions (`createSchool`, `getSchool`, `updateSchool`, `deactivateSchools`), snake→camel adapter (FE-008)
4. Sidebar `SchoolSwitcher` — filter out deactivated schools (`deleted_at IS NULL`) via `/me` schools list (FE-021, no fetching in sidebar)
5. Tests per FE-026/FE-027: role-gated per role; soft-nav + hard-load for each modal route
6. `pnpm generate:api` — regenerate `src/features/auth/api/generated/types.ts`
7. `make generate-check` — verify no drift

### Generated files

- `backend/docs/swagger.*` via `make generate-swag`
- `frontend/src/features/schools/api/generated/types.ts` via `pnpm generate:api`
- `backend/internal/schools/store/gen/` via `sqlc generate`

## 7. Tests required

### Backend (per BE-036)

- `schools_handler_test.go`:
  - Owner creates school → 201, fields match, `tenant_id` from session
  - Owner creates duplicate `code` in same tenant → 409
  - Non-owner creates → 403
  - Owner gets school by id → 200; cross-tenant id → identical 404
  - Owner updates school → 200; non-owner → 403
  - Owner deactivates school → 200, `deleted_at` set; excluded from list
  - List returns only non-deactivated, tenant-scoped schools
- Store/RLS tests with real PostgreSQL (testcontainers)

### Frontend (per FE-026 / FE-027)

- `schools.list.test.tsx` — owner sees table; non-owner redirected to `/unauthorized`
- `schools.add.test.tsx` — soft-nav opens dialog; hard-load shows full page; submit success/failure
- `schools.view.test.tsx` — sheet on soft-nav; full page on hard load; inline edit form present on detail page
- `schools.edit.inline.test.tsx` — submit inline edit form on `/schools/:id`; updates persist; toast confirms
- `schools.deactivate.test.tsx` — confirm dialog; cannot dismiss by overlay; school removed

## 8. Open questions

1. **Code uniqueness error:** Should duplicate `code` return `409` or `400` with field error? Decide in handler implementation.

## 9. Links

- [SOM-SIDEBAR-001](./SOM-SIDEBAR-001.md) — approved sidebar spec; school switcher at `/schools/*` supersedes "Settings page later" note
- [SOM-AUTHZ-001](./SOM-AUTHZ-001.md) — implemented; `school:read`/`school:write` permission matrix; role naming alignment
- [SOM-DATATABLE-001](./SOM-DATATABLE-001.md) — approved frontend data table core; list page uses `<DataTable>`
- [SOM-PAGINATION-001](./SOM-PAGINATION-001.md) — approved backend cursor engine; defines cursor wire format
- [docs/requirements/backend.md](../requirements/backend.md) — BE-016, BE-017, BE-020, BE-021, BE-025, BE-036
- [docs/requirements/frontend.md](../requirements/frontend.md) — FE-007, FE-008, FE-011, FE-013, FE-014, FE-019, FE-020, FE-021, FE-026, FE-027, FE-028, FE-029, FE-030, FE-031, FE-034, FE-035, FE-036, FE-037, FE-038, FE-039, FE-040, FE-041, FE-042, FE-043, FE-045, FE-046, FE-047
- [docs/api/openapi.yaml](../api/openapi.yaml) — source of truth for API contract
- [backend/migrations/00000000000003_core_domain.sql](../../../backend/migrations/00000000000003_core_domain.sql) — existing `schools` table (no new migration)
- [docs/specifications/registry.md](./registry.md) — living index; update on status change
