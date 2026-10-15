# Student Module – Roadmap & Gap Inventory

**Purpose:** Single source of truth for all student-related features, gaps, and dependencies for documentation and implementation planning.

**Last Updated:** 2026-10-15
**Status:** Draft

## Overview
Student management is core to school operations. This roadmap consolidates feature specs, gaps, and schema needs for student profile, listing, stats/KPIs, and guardian linking.

## Current State
* Backend schema: `students` with basic fields + `metadata JSONB`
* API: List, Delete, Summary. No enriched `GET /students/:id`
* Frontend: List table functional. Detail page is placeholder

## Prioritized Workstream

### Phase 1 – Core Profile & Schema
**Goal:** Make student detail usable with structured data

1. **Student Schema Missing Structured Fields**
   Doc: `docs/gaps/GAP-student-schema-missing-fields.md`
   Action: Add address, phone, email, nationality, emergency_contact_name, emergency_contact_phone, special_needs to `students`
   Dependency for: Profile, Listing

2. **Enhanced Student Profile**
   Doc: `docs/features/FEATURE-enhanced-student-profile.md`
   Action: Enriched `GET /students/:id` + `PATCH`, detail page with Overview, Academics, Guardians, Stats & KPIs, Notes tabs
   Depends on: 1

3. **Student Detail Page Placeholder**
   Doc: `docs/gaps/GAP-student-detail-placeholder.md`
   Action: Replace placeholder component with full profile UI
   Depends on: 2

### Phase 2 – Listing Enhancements
**Goal:** Faster triage in list view

4. **Student List Missing Fields**
   Doc: `docs/gaps/GAP-student-list-missing-fields.md`
   Action: Extend `ListStudents` to return date_of_birth, guardian_linked, attendance_rate
   Depends on: 1

5. **Enhanced Student Listing**
   Doc: `docs/features/FEATURE-enhanced-student-listing.md`
   Action: Add DOB, Guardian badge, Attendance % columns, bulk assign/delete, attendance filter, column visibility
   Depends on: 4

### Phase 3 – Relationships & Notes
**Goal:** Complete profile ecosystem

6. **Guardian Linking UI**
   Status: Gap not yet documented
   Action: Document UI flow for linking/unlinking guardians, viewing relationship type and contacts in detail page

7. **Student Notes**
   Status: Proposed
   Action: `student_notes` table + UI tab in profile

## Deferred
* Document and photo uploads – deferred per decision
* Background summary tables for stats – deferred, note in listing feature
* Bulk export CSV – dropped

## Dependencies Map
Schema columns → Enriched API → Detail page → Listing enhancements
Guardian links → Profile + Listing badges
Attendance data → Stats & KPIs tab + Listing attendance %

## Open Questions
* Compute attendance on read vs materialized view? Deferred
* Should contact info be normalized to `student_contacts` table now or later?
* Who can edit medical info / special needs? Role rules not defined

## References
* `docs/features/FEATURE-enhanced-student-profile.md`
* `docs/features/FEATURE-enhanced-student-listing.md`
* `docs/student-detail-gap-analysis.md`
* `docs/gaps/GAP-student-schema-missing-fields.md`
* `docs/gaps/GAP-student-list-missing-fields.md`
* `docs/gaps/GAP-student-detail-placeholder.md`
