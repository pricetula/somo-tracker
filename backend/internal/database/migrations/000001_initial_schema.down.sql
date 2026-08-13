-- Migration: 000001_initial_schema (rollback)
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only, v5)
-- Drops every object created by the squashed initial schema migration,
-- in strict reverse FK dependency order.

-- ============================================================================
-- TRIGGERS (dropped before their tables / functions)
-- ============================================================================

DROP TRIGGER IF EXISTS trg_users_updated_at                       ON users;
DROP TRIGGER IF EXISTS trg_cbc_schools_updated_at                 ON cbc_schools;
DROP TRIGGER IF EXISTS trg_cbc_parents_updated_at                 ON cbc_parents;
DROP TRIGGER IF EXISTS trg_cbc_students_updated_at                ON cbc_students;
DROP TRIGGER IF EXISTS trg_cbc_students_counts_update             ON cbc_students;
DROP TRIGGER IF EXISTS trg_cbc_students_counts_insert             ON cbc_students;
DROP TRIGGER IF EXISTS trg_cbc_students_counts_delete             ON cbc_students;
DROP TRIGGER IF EXISTS trg_memberships_counts_update              ON memberships;
DROP TRIGGER IF EXISTS trg_memberships_counts_insert              ON memberships;
DROP TRIGGER IF EXISTS trg_memberships_counts_delete              ON memberships;
DROP TRIGGER IF EXISTS trg_sync_invoice_payment_status_update     ON payments;
DROP TRIGGER IF EXISTS trg_sync_invoice_payment_status_insert     ON payments;
DROP TRIGGER IF EXISTS trg_sync_invoice_payment_status_delete     ON payments;
DROP TRIGGER IF EXISTS trg_cbc_streams_updated_at                 ON cbc_streams;
DROP TRIGGER IF EXISTS trg_assessment_sessions_refresh_summary    ON assessment_sessions;
DROP TRIGGER IF EXISTS trg_assessment_sessions_max_points_immutable ON assessment_sessions;
DROP TRIGGER IF EXISTS trg_grading_scale_ranges_immutable         ON grading_scale_ranges;
DROP TRIGGER IF EXISTS trg_behavior_notes_refresh_term_summary    ON behavior_notes;
DROP TRIGGER IF EXISTS trg_class_daily_attendance_summaries_updated_at   ON class_daily_attendance_summaries;
DROP TRIGGER IF EXISTS trg_student_term_subject_summaries_updated_at     ON student_term_subject_summaries;
DROP TRIGGER IF EXISTS trg_student_term_overall_summaries_updated_at     ON student_term_overall_summaries;
DROP TRIGGER IF EXISTS trg_student_cohort_position_summaries_updated_at  ON student_cohort_position_summaries;
DROP TRIGGER IF EXISTS trg_student_subject_strand_summaries_updated_at   ON student_subject_strand_summaries;
DROP TRIGGER IF EXISTS trg_student_performance_projections_updated_at    ON student_performance_projections;
DROP TRIGGER IF EXISTS trg_student_behavior_term_summaries_updated_at    ON student_behavior_term_summaries;
DROP TRIGGER IF EXISTS trg_teacher_subject_performance_summaries_updated_at ON teacher_subject_performance_summaries;
DROP TRIGGER IF EXISTS trg_teacher_delivery_summaries_updated_at         ON teacher_delivery_summaries;
DROP TRIGGER IF EXISTS trg_teacher_workload_summaries_updated_at         ON teacher_workload_summaries;
DROP TRIGGER IF EXISTS trg_class_learning_area_term_summaries_updated_at ON class_learning_area_term_summaries;
DROP TRIGGER IF EXISTS trg_class_term_attendance_summaries_updated_at    ON class_term_attendance_summaries;

-- ============================================================================
-- FUNCTIONS (dropped after their triggers, before dependent views/tables)
-- ============================================================================

DROP FUNCTION IF EXISTS fn_set_updated_at            CASCADE;
DROP FUNCTION IF EXISTS fn_timerange                 CASCADE;
DROP FUNCTION IF EXISTS fn_sync_invoice_payment_status_insert  CASCADE;
DROP FUNCTION IF EXISTS fn_sync_invoice_payment_status_delete  CASCADE;
DROP FUNCTION IF EXISTS fn_sync_invoice_payment_status_update CASCADE;
DROP FUNCTION IF EXISTS fn_sync_school_staff_counts_insert  CASCADE;
DROP FUNCTION IF EXISTS fn_sync_school_staff_counts_delete  CASCADE;
DROP FUNCTION IF EXISTS fn_sync_school_staff_counts_update  CASCADE;
DROP FUNCTION IF EXISTS fn_sync_school_student_counts_insert CASCADE;
DROP FUNCTION IF EXISTS fn_sync_school_student_counts_delete CASCADE;
DROP FUNCTION IF EXISTS fn_sync_school_student_counts_update CASCADE;
DROP FUNCTION IF EXISTS fn_current_tenant_id CASCADE;
DROP FUNCTION IF EXISTS fn_rls_tenant_policy CASCADE;
DROP FUNCTION IF EXISTS fn_check_non_break_slot CASCADE;
DROP FUNCTION IF EXISTS max_points_check CASCADE;
DROP FUNCTION IF EXISTS fn_assessment_sessions_after_publish CASCADE;
DROP FUNCTION IF EXISTS fn_block_assessment_max_points_update CASCADE;
DROP FUNCTION IF EXISTS fn_block_grading_scale_range_modification CASCADE;
DROP FUNCTION IF EXISTS fn_refresh_term_subject_summary_for_session CASCADE;
DROP FUNCTION IF EXISTS fn_compute_term_overall_summaries_for_term CASCADE;
DROP FUNCTION IF EXISTS fn_compute_single_student_term_overall_summary CASCADE;
DROP FUNCTION IF EXISTS fn_compute_cohort_positions_for_term CASCADE;
DROP FUNCTION IF EXISTS fn_refresh_subject_strand_summary_for_session CASCADE;
DROP FUNCTION IF EXISTS fn_compute_performance_projections_for_term CASCADE;
DROP FUNCTION IF EXISTS fn_refresh_student_behavior_term_summary CASCADE;
DROP FUNCTION IF EXISTS fn_refresh_student_behavior_term_summary_for_note CASCADE;
DROP FUNCTION IF EXISTS fn_compute_teacher_subject_performance_summaries CASCADE;
DROP FUNCTION IF EXISTS fn_compute_teacher_delivery_summaries CASCADE;
DROP FUNCTION IF EXISTS fn_compute_teacher_workload_summaries CASCADE;

-- ============================================================================
-- LAYER 11 — ATTENDANCE & BEHAVIOR
-- ============================================================================

DROP TABLE IF EXISTS class_term_attendance_summaries CASCADE;
DROP TABLE IF EXISTS class_learning_area_term_summaries CASCADE;
DROP TABLE IF EXISTS class_daily_attendance_summaries CASCADE;
DROP TABLE IF EXISTS cbc_attendance_sessions CASCADE;
DROP TABLE IF EXISTS attendance_term_summaries CASCADE;
DROP TABLE IF EXISTS behavior_notes CASCADE;
DROP TABLE IF EXISTS behavior_categories CASCADE;
DROP TABLE IF EXISTS attendance_records CASCADE;

-- ============================================================================
-- LAYER 10 — USER ACTIVE SCHOOL CONTEXT
-- ============================================================================

DROP TABLE IF EXISTS member_active_school CASCADE;

-- ============================================================================
-- LAYER 7 — ASSESSMENT ARCHITECTURE
-- ============================================================================

DROP TABLE IF EXISTS assessment_weight_configs CASCADE;

-- ============================================================================
-- LAYER 6 — OPERATIONS & TIMETABLE
-- ============================================================================

DROP TABLE IF EXISTS cbc_timetable_slots CASCADE;
DROP TABLE IF EXISTS timetable_structures CASCADE;
DROP TABLE IF EXISTS cbc_class_teachers CASCADE;

-- ============================================================================
-- LAYER 12 — ASSESSMENT & GRADING ENGINE
-- ============================================================================

DROP TABLE IF EXISTS student_assessment_outcome_grades CASCADE;
DROP TABLE IF EXISTS student_assessment_scores CASCADE;
DROP TABLE IF EXISTS assessment_sessions CASCADE;
DROP TABLE IF EXISTS grading_scale_ranges CASCADE;
DROP TABLE IF EXISTS grading_scale_profiles CASCADE;

-- ============================================================================
-- MATERIALISED SUMMARY TABLES (squashed from 000005–000016)
-- ============================================================================

DROP TABLE IF EXISTS teacher_workload_summaries CASCADE;
DROP TABLE IF EXISTS teacher_delivery_summaries CASCADE;
DROP TABLE IF EXISTS teacher_subject_performance_summaries CASCADE;
DROP TABLE IF EXISTS student_behavior_term_summaries CASCADE;
DROP TABLE IF EXISTS student_performance_projections CASCADE;
DROP TABLE IF EXISTS student_subject_strand_summaries CASCADE;
DROP TABLE IF EXISTS student_cohort_position_summaries CASCADE;
DROP TABLE IF EXISTS student_term_overall_summaries CASCADE;
DROP TABLE IF EXISTS student_term_subject_summaries CASCADE;

-- ============================================================================
-- LAYER 5 — CURRICULUM
-- ============================================================================

DROP TABLE IF EXISTS performance_indicators CASCADE;
DROP TABLE IF EXISTS cbc_sub_strands CASCADE;
DROP TABLE IF EXISTS cbc_strands CASCADE;
DROP TABLE IF EXISTS cbc_learning_areas CASCADE;

-- ============================================================================
-- LAYER 4 — FINANCE & HEALTH
-- ============================================================================

DROP TABLE IF EXISTS payments CASCADE;
DROP TABLE IF EXISTS invoice_items CASCADE;
DROP TABLE IF EXISTS invoices CASCADE;
DROP TABLE IF EXISTS fee_templates CASCADE;
DROP TABLE IF EXISTS fee_categories CASCADE;
DROP TABLE IF EXISTS medical_incidents CASCADE;
DROP TABLE IF EXISTS student_health_profiles CASCADE;

-- ============================================================================
-- LAYER 3 — CALENDAR
-- ============================================================================

DROP TABLE IF EXISTS academic_terms CASCADE;
DROP TABLE IF EXISTS academic_years CASCADE;

-- ============================================================================
-- LAYER 2 — CBC ACTORS
-- ============================================================================

DROP TABLE IF EXISTS cbc_student_enrollments CASCADE;
DROP TABLE IF EXISTS cbc_student_parents CASCADE;
DROP TABLE IF EXISTS cbc_students CASCADE;
DROP TABLE IF EXISTS cbc_parents CASCADE;
DROP TABLE IF EXISTS cbc_streams CASCADE;
DROP TABLE IF EXISTS cbc_classes CASCADE;
DROP TABLE IF EXISTS cbc_schools CASCADE;

-- ============================================================================
-- COUNTS TABLE
-- ============================================================================

DROP TABLE IF EXISTS school_member_counts CASCADE;

-- ============================================================================
-- LAYER 1 — PLATFORM INFRASTRUCTURE
-- ============================================================================

DROP TABLE IF EXISTS import_job_staging CASCADE;
DROP TABLE IF EXISTS import_job_failures CASCADE;
DROP TABLE IF EXISTS import_job_chunks CASCADE;
DROP TABLE IF EXISTS import_jobs CASCADE;
DROP TABLE IF EXISTS invitations CASCADE;
DROP TABLE IF EXISTS memberships CASCADE;
DROP TABLE IF EXISTS sessions CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS tenants CASCADE;

-- ============================================================================
-- ENUMS
-- ============================================================================

DROP TYPE IF EXISTS behavior_category_type CASCADE;
DROP TYPE IF EXISTS parent_relationship_type CASCADE;
DROP TYPE IF EXISTS payment_method_type CASCADE;
DROP TYPE IF EXISTS import_failure_type CASCADE;
DROP TYPE IF EXISTS import_staging_status CASCADE;
DROP TYPE IF EXISTS import_job_type CASCADE;
DROP TYPE IF EXISTS cbc_enrollment_status CASCADE;
DROP TYPE IF EXISTS invitation_status CASCADE;
DROP TYPE IF EXISTS user_role CASCADE;
DROP TYPE IF EXISTS gender_type CASCADE;
DROP TYPE IF EXISTS cbc_grade_level CASCADE;
DROP TYPE IF EXISTS cbc_education_level CASCADE;
DROP TYPE IF EXISTS cbc_school_type CASCADE;
DROP TYPE IF EXISTS cbc_learning_pathway CASCADE;
DROP TYPE IF EXISTS teacher_role CASCADE;
DROP TYPE IF EXISTS import_chunk_status CASCADE;
DROP TYPE IF EXISTS block_type CASCADE;
DROP TYPE IF EXISTS invoice_payment_status CASCADE;
DROP TYPE IF EXISTS attendance_status CASCADE;
DROP TYPE IF EXISTS behavior_note_status CASCADE;
DROP TYPE IF EXISTS behavior_severity CASCADE;
DROP TYPE IF EXISTS cbc_performance_level CASCADE;
DROP TYPE IF EXISTS assessment_session_status CASCADE;
DROP TYPE IF EXISTS assessment_evaluation_method CASCADE;

-- ============================================================================
-- EXTENSIONS (optional — only drop if no other objects depend on it)
-- ============================================================================

DROP EXTENSION IF EXISTS btree_gist CASCADE;
DROP EXTENSION IF EXISTS pgcrypto CASCADE;
