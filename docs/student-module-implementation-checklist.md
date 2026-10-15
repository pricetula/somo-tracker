# Student Module Implementation Checklist

Derived from `docs/student-module-roadmap.md`. For implementation starting next week.

## Phase 1 – Core Profile & Schema
- [ ] Migration: Add structured columns to `students`
  - address, phone, email, nationality, emergency_contact_name, emergency_contact_phone, special_needs
  - RLS policies updated
  - Backfill script for existing metadata if needed

- [ ] Backend API: Enriched student profile
  - [ ] `GET /students/:id` returns enriched payload with personal fields, current class, enrollment history, guardians, stats_kpis, notes
  - [ ] `PATCH /students/:id` validates and updates editable fields
  - [ ] Error contract compliant

- [ ] Frontend: Student Detail Page
  - [ ] Replace placeholder `StudentDetail` component
  - [ ] Tabs: Overview, Academics, Guardians, Stats & KPIs, Notes
  - [ ] Stats & KPIs tab shows attendance rate, present/absent/tardy counts for current term
  - [ ] Edit mode with validation

## Phase 2 – Listing Enhancements
- [ ] Backend API: Extend ListStudents
  - [ ] Return date_of_birth, guardian_linked boolean, attendance_rate
  - [ ] Keep existing filters and pagination

- [ ] Frontend: Enhanced Student Listing
  - [ ] Add DOB, Guardian badge, Attendance % columns
  - [ ] Inline badges for without class / without guardian
  - [ ] Bulk assign class action
  - [ ] Bulk delete action
  - [ ] Attendance threshold filter
  - [ ] Column visibility toggle

## Phase 3 – Relationships & Notes
- [ ] Guardian linking UI
  - [ ] View linked guardians in detail page
  - [ ] Link/unlink flow

- [ ] Student Notes
  - [ ] `student_notes` table
  - [ ] Notes tab in profile with create/edit/delete

## Deferred
- [ ] Document / photo uploads
- [ ] Background summary tables for stats
- [ ] Bulk export CSV

## Definition of Done
- All acceptance criteria in feature specs met
- RLS enforced
- Error contract compliant
- No dumb back buttons
- Docs updated

## References
- `docs/student-module-roadmap.md`
- `docs/features/FEATURE-enhanced-student-profile.md`
- `docs/features/FEATURE-enhanced-student-listing.md`
