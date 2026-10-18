# Feature: Enhanced Student Profile

**Status:** Done
**Owner:** Product / Platform
**Created:** 2026-10-15
**Last Updated:** 2026-10-15
**Related Issues:** #
**Affected Layers:** backend | frontend

## Summary
Expand the current student detail view from a placeholder to a comprehensive profile with editable personal info, guardian links, and student stats/KPIs for attendance, academic progress and engagement.

## Rationale
**Why now:** Admins and teachers currently bounce between list, enrollment, guardian and attendance screens to assemble a student’s story. The placeholder profile blocks core workflows and creates data entry friction.
**Business outcome:** Faster decisions, fewer support tickets for missing student details, and a foundation for guardian communications and reporting.
**Success metrics:**
- Profile page load < 500ms p95
- Edit success rate > 98%
- Time to view full student context reduced from 3 screens to 1

## Current State
**Backend:** `StudentsHandler.GetStudent` and `StudentsHandler.UpdateStudent` exist in `backend/internal/api/students_handler.go` with Swagger annotations for `GET /students/{id}` and `PATCH /students/{id}`. Handlers call `StudentsService.GetStudentByID` / `UpdateStudent`. Routes are **not wired** in `backend/internal/api/router.go` — only `GET /students`, `GET /students/summary`, `DELETE /students` are registered.
**Frontend:** `frontend/src/features/students/components/student-detail.tsx` renders placeholder UI with `profile` state initialized to null and TODO comments for `GET /students/:id` and `PATCH /students/:id`. Tabs are scaffolded but empty. List page works.
**Schema:** `students` table has basic fields + `metadata JSONB`. Structured fields for address/phone/email/nationality/emergency contact missing.

## Problem Statement
Today student detail at `/students/[id]` is a placeholder:
```tsx
<h1>Student</h1><p>Student ID: {id}</p>
```
List view only shows name, admission number, class, grade level, gender. No DOB, no guardian, no contact info, no edit capability. Admins must go to multiple screens to get a complete picture.

## Solution Overview
Build a full profile page with tabs/sections:
* Overview – key bio
* Academics – current class, enrollment history, grade level
* Guardians – linked guardians with relationship type
* Stats & KPIs – attendance rate, punctuality, engagement metrics
* Notes – internal comments

### User Stories
- As a School Admin, I want to view a student’s full profile on one page so I can make decisions quickly
- As a Data Officer, I want to edit personal details and contact info inline
- As a Teacher, I want to see guardian contacts from the profile

### Acceptance Criteria
- [ ] Student detail page loads full profile data from API
- [ ] Display: full_name, admission_number, date_of_birth, gender, address, phone, email, nationality
- [ ] Show current class + enrollment history
- [ ] Show linked guardians with contact info
- [ ] Show Stats & KPIs: attendance rate, tardy count, days present/absent for current term
- [ ] Show academic progress summary where data exists
- [ ] Edit form with validation, persists via API
- [ ] Uses Shadcn components, no dumb back buttons
- [ ] Error contract compliant

## Scope

### Backend
- API changes:
  - `GET /students/:id` → return enriched profile, not just list item
  - `PATCH /students/:id` → update personal fields
- DB changes:
  - Extend `students` with structured fields or use `metadata JSONB` for extensibility
  - Ensure RLS enforced
- Consider new table `student_contacts` or `student_documents` if needed

### Frontend
- Replace placeholder `StudentDetail` component with full profile UI
- Tabs: Overview, Academics, Guardians, Stats & KPIs, Notes
- Edit mode with form validation
- Link to guardian profile

#### Stats & KPIs Tab Wireframe
```
[Student Detail Header]
Name | Admission No. | Class | Actions [Edit] [Print]

Tabs: Overview | Academics | Guardians | Stats & KPIs | Notes

Stats & KPIs Content
┌─────────────────────────────────────────────────────────┐
│ Term Selector: [Term 1 2026 ▼]   Last Updated: today   │
├─────────────────────────────────────────────────────────┤
│ KPI Cards                                               │
│ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐         │
│ │ Attendance  │ │ Tardy       │ │ Present Days│         │
│ │ 92%         │ │ 3           │ │ 85 / 92     │         │
│ └─────────────┘ └─────────────┘ └─────────────┘         │
│                                                         │
│ Trend Mini Chart                                        │
│ Attendance over weeks in term                           │
│ [sparkline]                                             │
│                                                         │
│ Recent Absences Table                                   │
│ Date | Slot | Subject | Reason | Marked By               │
│ ...                                                     │
└─────────────────────────────────────────────────────────┘
```
Components: shadcn Card, Tabs, Select, Badge, Table, Chart via Recharts

### Public / Marketing
- N/A

### Non-Goals
- Full health records system v1
- Messaging to guardians
- Document and photo uploads – deferred

## Design Notes
Current `students` schema has limited columns. Use `metadata JSONB` for flexible extension now, migrate to columns later if needed. Keep compatibility with existing `ListStudents`. Stats & KPIs read from summary tables per ADR-2026-10-15-stats-architecture.

## Open Gaps / Risks
- GAP: Guardian contact API not exposed in profile payload
- GAP: No academic results tables yet for grade KPIs
- Stats use summary tables updated by background workers per ADR-2026-10-15-stats-architecture

## Test Plan
- API contract tests for GET/PATCH
- Frontend E2E for edit flow
- RLS tests for tenant isolation

### Test Matrix
| ID | Scope | Method | Data | Expected Result |
|----|-------|--------|------|-----------------|
| T-01 | GET /students/:id | API contract | Valid student ID, correct tenant | 200 with student_id, admission_number, full_name, date_of_birth, gender, nemis_upn |
| T-02 | GET /students/:id | API contract | Wrong tenant | 404 / 403 per RLS |
| T-03 | PATCH /students/:id | API contract | Valid partial payload | 200, fields persisted |
| T-04 | PATCH /students/:id | Validation | Invalid date format | 400 with error contract |
| T-05 | UI Load | Playwright | Navigate to /students/[id] | Profile renders, tabs visible, no dumb back button |
| T-06 | UI Edit | Playwright | Edit name & save | Optimistic update, toast success, query invalidated |
| T-07 | RLS | Integration | Cross-tenant request | Data not leaked |


## Rollout
- Feature flag `enhanced_student_profile`
- Backend first, then frontend

## Implementation Notes
**2026-10-15 – Done**
- Wired `GET /students/:id` and `PATCH /students/:id` in `backend/internal/api/router.go`; handlers already existed in `students_handler.go`
- Added `getStudent` / `updateStudent` helpers to `frontend/src/lib/api/students.ts`
- Replaced placeholder `StudentDetail` with TanStack Query data fetching + mutation, Shadcn UI tabs, edit mode
**2026-10-15 – Updated**
- Structured columns added via migration 000023_add_student_structured_fields: address, phone, email, nationality, emergency_contact_name, emergency_contact_phone, special_needs
- `StudentDetail` Overview now renders and inline-edits those fields via `PATCH /students/:id`
- Backfill script `backend/scripts/backfill_student_metadata.go` added for metadata → columns hydration
- Stats & KPIs, Guardians, Academics tabs scaffolded for future work per open gaps

## References
- Current placeholder: `frontend/src/features/students/components/student-detail.tsx`
- Current list: `frontend/src/features/students/components/students-list.tsx`
- Service: `backend/internal/services/students_service.go`
- Schema: `docs/reference/backend_schema.md` → `students`
