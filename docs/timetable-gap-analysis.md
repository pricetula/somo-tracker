# Timetable Module – Gap Analysis

**Scope:** Timetable features touching Students, Classes, Teachers
**Last Updated:** 2026-10-15
**Status:** Draft

## Current Implementation

### Backend
* `timetable_templates` + `time_slots` – template creation and management
* `class_timetable_slots` – mapping of class_room + day_of_week + time_slot + subject + teacher_membership + room
* `timetable_substitutions` – substitution requests
* `timetable_attendance` – attendance per student per slot per date
* API endpoints:
  * Create/Update/List/Get Template
  * List Time Slots
  * SetupClassTimetableSlot
  * GetClassSlotsByTemplate
  * DeleteClassTimetableSlot
  * Substitutions CRUD

### Frontend
* Template list page
* Template creation wizard with time slot matrix
* `TimetableGrid` with class selection and Assign/Edit/Delete per slot
* `AssignSlotForm` page exists for subject/teacher/room assignment
* Substitutions table UI exists

## Identified Gaps

### 1. Class Timetable Assignment Polish & Validation
**Gap:** Assignment UI exists but lacks conflict detection, bulk assignment, and clear validation feedback.
Backend `SetupClassTimetableSlot` exists and frontend `AssignSlotForm` works, but no guardrails.
**Impact:** Invalid timetables can be created, UX is fragmented.
**Related:** `frontend/src/features/timetable/SPEC.md` Future Iterations

### 2. Teacher Timetable View
**Gap:** No view for teachers to see their own weekly schedule.
**Impact:** Teachers cannot see classes they are assigned to.
**Needed:** `GET /timetable/teacher/:membershipId/schedule` + UI

### 3. Class Timetable View
**Gap:** No class-specific timetable view for students/teachers.
**Impact:** Students/class admins cannot see weekly schedule.
**Needed:** Class timetable grid view

### 4. Student Timetable View
**Gap:** No personal weekly timetable for students based on enrollment.
**Impact:** Students cannot see their own schedule.
**Needed:** Derive from class timetable slots + student enrollment

### 5. Conflict Detection
**Gap:** No validation for teacher double-booking, room double-booking, class overlapping slots.
**Impact:** Invalid timetables can be created.
**Needed:** Backend validation on `SetupClassTimetableSlot`

### 6. Timetable Publishing / Versioning
**Gap:** No concept of active timetable per academic term/year.
**Impact:** Cannot switch between draft and published timetables.
**Needed:** Link timetable templates to academic terms and publish flag

### 7. Attendance Marking Integration
**Gap:** Attendance marking UI is not tied to timetable slots context.
**Impact:** Manual attendance entry is disconnected from schedule.
**Needed:** Attendance UI filtered by class timetable slot

### 8. Substitution Workflow Completeness
**Gap:** Substitution creation UI exists but approval flow, notifications, and auto-update of attendance records missing.
**Impact:** Incomplete substitution management.

### 9. Mobile / Print Views
**Gap:** No printable weekly grid for classes/teachers.
**Impact:** Offline use limited.

## Proposed Feature Specs

* `FEATURE-timetable-class-assignment-polish.md` – Conflict detection and bulk assignment
* `FEATURE-timetable-teacher-view.md`
* `FEATURE-timetable-class-view.md`
* `FEATURE-student-timetable-view.md`
* `FEATURE-timetable-conflict-detection.md`
* `FEATURE-timetable-publishing.md`

## Prioritization Recommendation
1. Conflict Detection
2. Class Timetable View
3. Teacher Timetable View
4. Student Timetable View
5. Timetable Publishing / Versioning
6. Attendance marking integration
7. Substitution workflow completion
8. Mobile / Print Views

## References
* `frontend/src/features/timetable/SPEC.md`
* `backend/internal/api/timetable_handler.go`
* `docs/backend_schema.md` – class_timetable_slots, timetable_substitutions, timetable_attendance
