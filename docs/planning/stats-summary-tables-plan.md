# Stats Summary Tables Plan

**Created:** 2026-10-15
**Status:** Draft

## Overview
Inventory of stat queries from documented features and mapping to summary tables updated by background workers.

## Summary Tables

### student_term_attendance_stats
**Used by:** Student listing attendance %, Student profile KPIs, Attendance Summary Stats
**Key:** school_id, academic_term_id, student_id
**Columns:** present_count, absent_count, late_count, attendance_rate, last_updated_at
**Updated by:** attendance.marked events

### guardian_link_stats
**Used by:** Student listing guardian badge, Guardian Summary Stats
**Key:** school_id, student_id / guardian_id
**Columns:** guardian_count, is_linked, primary_guardian_id, last_updated_at
**Updated by:** guardian_student_links create/delete

### teacher_workload_stats
**Used by:** Teacher Summary Stats, Teacher Profile Enhancement
**Key:** school_id, academic_term_id, teacher_membership_id
**Columns:** classes_taught_count, total_slots_count, last_updated_at
**Updated by:** class_timetable_slot create/delete, teacher assignment changes

### class_enrollment_stats
**Used by:** Class Summary Stats, Class Profile Enhancement, Classes Enrollment Polish
**Key:** school_id, academic_term_id, class_id
**Columns:** enrolled_count, unassigned_count, last_updated_at
**Updated by:** enrollments create/delete

### student_profile_stats
**Used by:** Enhanced Student Profile KPIs
**Key:** school_id, student_id
**Columns:** attendance_rate, guardian_linked, etc.

## Event Flow
Service updates primary data → publishes event → Asynq worker processes → upserts summary table

## Migration Steps
1. Create summary tables
2. Backfill from existing data
3. Add event emission in services
4. Deploy workers
5. Switch reads to summary tables

## References
* ADR-2026-10-15-stats-architecture.md
