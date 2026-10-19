# Student Module Implementation Checklist

Derived from `docs/planning/student-module-roadmap.md`. For implementation starting next week.

## Phase 1 – Core Profile & Schema
- [x] Migration: Add structured columns to `students`
  - address, phone, email, nationality, emergency_contact_name, emergency_contact_phone, special_needs
  - RLS policies updated
  - Backfill script for existing metadata if needed

- [x] Backend API: Enriched student profile
  - [x] `GET /students/:id` returns enriched payload with personal fields, current class, enrollment history, guardians, stats_kpis, notes
  - [x] `PATCH /students/:id` validates and updates editable fields
  - [x] Error contract compliant

- [x] Frontend: Student Detail Page
  - [x] Replace placeholder `StudentDetail` component
  - [x] Tabs: Overview, Academics, Guardians, Stats & KPIs, Notes
  - [x] Stats & KPIs tab shows attendance rate, present/absent/tardy counts for current term
  - [x] Edit mode with validation

## Phase 2 – Listing Enhancements
- [x] Backend API: Extend ListStudents
  - [x] Return date_of_birth, guardian_linked boolean, attendance_rate
  - [x] Keep existing filters and pagination

- [x] Frontend: Enhanced Student Listing
  - [x] Add DOB, Guardian badge, Attendance % columns
  - [x] Inline badges for without class / without guardian
  - [x] Bulk assign class action
  - [x] Bulk delete action

## Phase 3 – Relationships & Notes
- [x] Guardian linking UI
  - [x] View linked guardians in detail page
  - [x] Link/unlink flow

- [x] Student Notes
  - [x] `student_notes` table
  - [x] Notes tab in profile with create/edit/delete

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
- `docs/planning/student-module-roadmap.md`
- `docs/features/FEATURE-enhanced-student-profile.md`
- `docs/features/FEATURE-enhanced-student-listing.md`
