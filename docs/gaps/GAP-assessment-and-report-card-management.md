# Gap: Assessment and Report Card Management

**Area:** backend | frontend
**Severity:** High
**Status:** Open
**Owner:** 
**Created:** 2026-10-15
**Related Issues:** #

## Description
The system has curriculum, timetable, enrollment and events foundations but lacks a complete Assessment domain. There is no way to create assessments per class/subject/term, capture student marks, convert percentages to grades/CBE descriptors, generate term report cards, or publish results to students/guardians. No audit trail for mark changes and no batch publishing.

## Current Behavior
* Teachers cannot record formative/summative assessment results.
* No report card generation or publishing workflow exists.
* No grade scale configuration or conversion from percentage to grade.
* Assessment data capture is not linked to `student_class_enrollments`, `subjects`, `academic_years`/`academic_terms`.
* No audit for corrections after approval/publish.

## Expected Behavior
* Teachers can create assessments per class/subject/term with type Formative/Summative and sub-type CAT/Midterm/End Term etc.
* Marks entry is percentage-first, applied to a school-wide frozen grade scale snapshot at assessment creation.
* Report cards aggregate approved assessments + attendance per student per term.
* Batch publish of assessment results and report cards to selected student IDs with server-side PDF generation.
* Corrections after publish require reason, audit trail and re-approval.

## Impact
Core school management flow broken. Teachers cannot record academic performance, schools cannot issue report cards, guardians have no visibility into progress. Blocks adoption for academic reporting.

## Reproduction Steps
1. Log in as Teacher.
2. Navigate to Class > Subject.
3. No Assessment or Marks entry option exists.
4. Navigate to Student profile.
5. No report card or results history visible.

## Proposed Solutions
- Option A: Full assessment domain with grade scales, weighting config, report card generation and batch publishing.
- Option B: Minimal marks sheet only, defer report cards.
Chosen: Option A with scope constraints: data capture only, no live exam room management.

## Decision
Lock scope to data capture + reporting:
* Domain named Assessments with Type Formative/Summative and sub-type CAT/Midterm/End Term.
* No room/invigilation/seating management now.
* Marks entered as percentage, converted using school-wide grade scale snapshotted at assessment creation. No re-grading.
* Topic/Sub-topic linkage optional, assessments created at Subject level.
* Assessment weighting configurable per school and snapshotted.
* Batch publish of selected student IDs. PDF server side.
* Corrections require reason and re-approval.

## Acceptance Criteria
- [ ] Assessment creation per class/subject/term with type/sub-type
- [ ] Marks entry grid with percentage input and frozen grade conversion
- [ ] School-wide grade scale configuration with versioning/snapshot
- [ ] Report card generation aggregates approved assessments + attendance
- [ ] Batch publish of results/report cards with PDF generation
- [ ] Audit trail for mark changes and corrections with reason and re-approval
- [ ] Guardian/Student view of published results only via RLS

## Links
References planning doc `docs/planning/assessment-management-strategy.md`
