-- Migration: 000001_initial_schema (rollback)
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only, v5)
-- Drops every object created by the initial schema migration,
-- in strict reverse FK dependency order.

BEGIN;

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
DROP TRIGGER IF EXISTS trg_auto_register_subject_teacher          ON cbc_timetable_slots;

-- ============================================================================
-- FUNCTIONS (dropped after their triggers, before dependent views/tables)
-- ============================================================================

DROP FUNCTION IF EXISTS fn_set_updated_at            CASCADE;
DROP FUNCTION IF EXISTS fn_timerange                 CASCADE;
DROP FUNCTION IF EXISTS fn_sync_invoice_payment_status CASCADE;
DROP FUNCTION IF EXISTS fn_sync_school_staff_counts_insert  CASCADE;
DROP FUNCTION IF EXISTS fn_sync_school_staff_counts_delete  CASCADE;
DROP FUNCTION IF EXISTS fn_sync_school_staff_counts_update  CASCADE;
DROP FUNCTION IF EXISTS fn_sync_school_student_counts_insert CASCADE;
DROP FUNCTION IF EXISTS fn_sync_school_student_counts_delete CASCADE;
DROP FUNCTION IF EXISTS fn_sync_school_student_counts_update CASCADE;
DROP FUNCTION IF EXISTS fn_auto_register_subject_teacher     CASCADE;

-- ============================================================================
-- LAYER 10 — USER ACTIVE SCHOOL CONTEXT
-- ============================================================================

DROP TABLE IF EXISTS member_active_school CASCADE;

-- ============================================================================
-- LAYER 9 — REPORTING
-- ============================================================================

DROP TABLE IF EXISTS cbc_term_competency_summaries CASCADE;

-- ============================================================================
-- LAYER 8 — RESULTS
-- ============================================================================

DROP TABLE IF EXISTS learner_portfolios CASCADE;
DROP TABLE IF EXISTS learner_rubric_results CASCADE;
DROP TABLE IF EXISTS assessment_sessions CASCADE;

-- ============================================================================
-- LAYER 7 — ASSESSMENT ARCHITECTURE
-- ============================================================================

DROP TABLE IF EXISTS assessment_blueprint_indicators CASCADE;
DROP TABLE IF EXISTS assessment_blueprints CASCADE;
DROP TABLE IF EXISTS assessment_weight_configs CASCADE;

-- ============================================================================
-- LAYER 6 — OPERATIONS
-- ============================================================================

DROP TABLE IF EXISTS cbc_timetable_slots CASCADE;
DROP TABLE IF EXISTS cbc_attendance_logs CASCADE;
DROP TABLE IF EXISTS cbc_attendance_periods CASCADE;
DROP TABLE IF EXISTS cbc_class_teachers CASCADE;

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
DROP TABLE IF EXISTS import_jobs CASCADE;
DROP TABLE IF EXISTS invitations CASCADE;
DROP TABLE IF EXISTS memberships CASCADE;
DROP TABLE IF EXISTS sessions CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS tenants CASCADE;

-- ============================================================================
-- ENUMS
-- ============================================================================

DROP TYPE IF EXISTS cbc_enrollment_status CASCADE;
DROP TYPE IF EXISTS invitation_status CASCADE;
DROP TYPE IF EXISTS attendance_status CASCADE;
DROP TYPE IF EXISTS user_role CASCADE;
DROP TYPE IF EXISTS gender_type CASCADE;
DROP TYPE IF EXISTS cbc_grade_level CASCADE;
DROP TYPE IF EXISTS cbc_education_level CASCADE;
DROP TYPE IF EXISTS cbc_school_type CASCADE;
DROP TYPE IF EXISTS cbc_learning_pathway CASCADE;
DROP TYPE IF EXISTS cbc_assessment_type CASCADE;
DROP TYPE IF EXISTS knec_target_exam CASCADE;
DROP TYPE IF EXISTS cbc_rubric_level CASCADE;
DROP TYPE IF EXISTS cbc_rubric_level_with_sub_levels CASCADE;
DROP TYPE IF EXISTS lrr_score_type CASCADE;
DROP TYPE IF EXISTS portfolio_evidence_type CASCADE;
DROP TYPE IF EXISTS knec_sync_status CASCADE;
DROP TYPE IF EXISTS invoice_payment_status CASCADE;

-- ============================================================================
-- EXTENSIONS (optional — only drop if no other objects depend on it)
-- ============================================================================

DROP EXTENSION IF EXISTS btree_gist CASCADE;

COMMIT;
