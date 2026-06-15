-- Migration: 000001_initial_schema (rollback)
-- ============================================================================
-- Drops every object created by the initial schema migration,
-- in strict reverse-dependency order.
-- ============================================================================

-- VIEWS (no table dependencies)
DROP VIEW IF EXISTS v_cbc_final_term_scores;
DROP VIEW IF EXISTS v_igcse_final_term_scores;
DROP VIEW IF EXISTS v_invoice_balances;

-- FINANCIAL
DROP TABLE IF EXISTS payments        CASCADE;
DROP TABLE IF EXISTS invoice_items   CASCADE;
DROP TABLE IF EXISTS invoices        CASCADE;
DROP TABLE IF EXISTS fee_templates   CASCADE;
DROP TABLE IF EXISTS fee_categories  CASCADE;

-- HEALTH
DROP TABLE IF EXISTS medical_incidents       CASCADE;
DROP TABLE IF EXISTS student_health_profiles  CASCADE;

-- TIMETABLE
DROP TABLE IF EXISTS ib_timetable_slots    CASCADE;
DROP TABLE IF EXISTS igcse_timetable_slots CASCADE;
DROP TABLE IF EXISTS cbc_timetable_slots   CASCADE;

-- ATTENDANCE
DROP TABLE IF EXISTS ib_attendance_logs      CASCADE;
DROP TABLE IF EXISTS ib_attendance_periods   CASCADE;
DROP TABLE IF EXISTS igcse_attendance_logs   CASCADE;
DROP TABLE IF EXISTS igcse_attendance_periods CASCADE;
DROP TABLE IF EXISTS cbc_attendance_logs     CASCADE;
DROP TABLE IF EXISTS cbc_attendance_periods  CASCADE;

-- ASSESSMENT WEIGHTS
DROP TABLE IF EXISTS assessment_weights CASCADE;

-- IB MODULE
DROP TABLE IF EXISTS ib_class_teachers          CASCADE;
DROP TABLE IF EXISTS ib_task_criterion_scores   CASCADE;
DROP TABLE IF EXISTS ib_tasks                   CASCADE;
DROP TABLE IF EXISTS ib_disciplines             CASCADE;
DROP TABLE IF EXISTS ib_subject_groups          CASCADE;

-- IGCSE MODULE
DROP TABLE IF EXISTS igcse_class_teachers    CASCADE;
DROP TABLE IF EXISTS igcse_assessment_marks  CASCADE;
DROP TABLE IF EXISTS igcse_class_assessments CASCADE;
DROP TABLE IF EXISTS igcse_papers            CASCADE;
DROP TABLE IF EXISTS igcse_subjects          CASCADE;

-- CBC MODULE
DROP TABLE IF EXISTS cbc_class_teachers       CASCADE;
DROP TABLE IF EXISTS cbc_task_evaluations     CASCADE;
DROP TABLE IF EXISTS cbc_formative_tasks      CASCADE;
DROP TABLE IF EXISTS cbc_learning_outcomes    CASCADE;
DROP TABLE IF EXISTS cbc_sub_strands          CASCADE;
DROP TABLE IF EXISTS cbc_strands              CASCADE;
DROP TABLE IF EXISTS cbc_learning_areas       CASCADE;

-- CORE TABLES
DROP TABLE IF EXISTS student_enrollments  CASCADE;
DROP TABLE IF EXISTS students             CASCADE;
DROP TABLE IF EXISTS classes              CASCADE;
DROP TABLE IF EXISTS academic_terms       CASCADE;
DROP TABLE IF EXISTS academic_years       CASCADE;
DROP TABLE IF EXISTS grades               CASCADE;
DROP TABLE IF EXISTS memberships          CASCADE;
DROP TABLE IF EXISTS schools              CASCADE;
DROP TABLE IF EXISTS education_systems    CASCADE;
DROP TABLE IF EXISTS sessions             CASCADE;
DROP TABLE IF EXISTS users                CASCADE;
DROP TABLE IF EXISTS tenants              CASCADE;

-- ENUM TYPES
DROP TYPE IF EXISTS assessment_type;
DROP TYPE IF EXISTS ib_criterion_type;
DROP TYPE IF EXISTS cbc_score_level;
DROP TYPE IF EXISTS gender_type;
DROP TYPE IF EXISTS attendance_status;
DROP TYPE IF EXISTS enrollment_status;
DROP TYPE IF EXISTS user_role;

-- FUNCTIONS
DROP FUNCTION IF EXISTS fn_timerange;
DROP FUNCTION IF EXISTS fn_set_updated_at;

-- ============================================================================
-- END OF ROLLBACK
-- ============================================================================
