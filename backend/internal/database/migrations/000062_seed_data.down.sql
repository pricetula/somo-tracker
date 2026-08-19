-- Migration: 000062_seed_data (rollback)
-- SomoTracker — rollback for 000062_seed_data.

-- ================================================================================
-- 🌱 SEED DATA ROLLBACK
-- ================================================================================
-- Order matters: delete tables whose triggers sync into school_member_counts
-- BEFORE deleting cbc_schools themselves, to avoid FK violations when the
-- AFTER DELETE triggers (trg_cbc_students_counts_delete,
-- trg_memberships_counts_delete) try to INSERT/UPDATE school_member_counts
-- after the corresponding cbc_schools row is already gone.

DELETE FROM assessment_weight_configs;
DELETE FROM member_active_school;
DELETE FROM cbc_student_enrollments;
DELETE FROM cbc_student_parents;
DELETE FROM cbc_students;
DELETE FROM memberships;
DELETE FROM school_member_counts;
DELETE FROM cbc_schools;
DELETE FROM tenants;
