# Student Module — Focused Gap Analysis

**Created:** 2026-10-15
**Scope:** Students only. Excludes auth, bulk imports.
**Status:** Draft

## Current State

**Frontend**
- Routes:
  - `/students` → `StudentsTable` working, uses `listStudents`, `deleteStudents`, `getStudentSummary`
  - `/students/[id]` → `StudentDetail` component exists but is placeholder. State initialized to null, no fetch, TODO comments for `GET /students/:id` and `PATCH /students/:id`
  - `/students/add` → page exists, implementation unknown
- Feature module: `frontend/src/features/students/` with components `students-list.tsx`, `student-detail.tsx`

**Backend**
- Router registered:
  - `GET /api/students`
  - `GET /api/students/summary`
  - `DELETE /api/students`
  - `GET /api/students/unassigned`
  - Import jobs excluded
- Router NOT registered:
  - `GET /api/students/:id` → handler `StudentsHandler.GetStudent` exists in code but not wired
  - `PATCH /api/students/:id` → handler `StudentsHandler.UpdateStudent` exists in code but not wired
- Service interface has `GetStudentByID` and `UpdateStudent` methods implied
- Schema: `students` has `full_name, date_of_birth, gender, admission_number, metadata JSONB`. No structured address/phone/email/nationality/emergency contact

**Docs**
- Feature spec: `docs/features/FEATURE-enhanced-student-profile.md` — Proposed
- Feature spec: `docs/features/FEATURE-enhanced-student-listing.md` — Proposed
- Gaps:
  - `GAP-student-schema-missing-fields.md` — Open High
  - `GAP-student-detail-placeholder.md` — Open High
  - `GAP-student-list-missing-fields.md` — Open High

## Mismatches

1. **Detail UI missing data**
   - Frontend expects data but never fetches
   - Backend handler exists but not routed

2. **Schema insufficient for profile**
   - Profile UI designs fields: address, phone, email, nationality, emergency contact
   - DB only has unstructured `metadata`

3. **List enrichment missing**
   - `StudentListItem` type includes `date_of_birth` but backend `ListStudents` response may not expose it consistently
   - Expected columns: DOB, guardian badge, attendance % — not in current API

4. **Edit capability not wired**
   - UI has Edit button state but no mutation

## Lean Targeted Plan

### Priority 1 — Make detail page usable
**Goal:** Replace placeholder with read-only profile

Steps:
- Wire backend routes:
  - `protected.Get("/students/:id", r.Students.GetStudent)`
  - `protected.Patch("/students/:id", r.Students.UpdateStudent)`
- Implement `StudentsService.GetStudentByID` to return enriched profile from current schema + metadata
- Frontend: replace placeholder with TanStack Query fetch using `api.get('/api/students/${id}')`
- Update `StudentDetail` to render profile fields, no edit yet

Deliverable: `/students/[id]` loads data. Gap `GAP-student-detail-placeholder.md` partially closed.

### Priority 2 — Schema for structured profile
**Goal:** Add structured fields for profile

Steps:
- Create migration adding `address, phone, email, nationality, emergency_contact_name, emergency_contact_phone, special_needs`
- Update `StudentsService` queries and API response DTO
- Backfill from `metadata` where present
- Update RLS policies

Deliverable: `GAP-student-schema-missing-fields.md` closed.

### Priority 3 — Edit and list enhancements
**Goal:** Enable edit and richer list

- Implement `PATCH /students/:id` validation and service update
- Frontend edit form with zod validation, `form.setError` on 400
- Extend `ListStudents` response to include `date_of_birth, guardian_linked, attendance_rate`
- Update `StudentsTable` columns accordingly

Deliverable: `FEATURE-enhanced-student-profile.md` and `FEATURE-enhanced-student-listing.md` moved to In Progress.

## Dependencies
- Academic period resolver for attendance stats — defer stats tab for now, show read-only attendance rate when available
- Guardian linking UI — defer to next iteration

## Risks
- Computing attendance on read may be slow → defer KPI cards, show basic rate first
- Metadata backfill may be inconsistent → make fields nullable

## Next Action
Approve Priority 1. I can wire routes and stub the service response to unblock frontend detail page immediately, keeping changes unstaged.
