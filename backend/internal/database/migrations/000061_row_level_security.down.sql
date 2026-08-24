-- Migration: 000061_row_level_security (rollback)
-- SomoTracker — rollback for 000061_row_level_security.

DO $$ DECLARE tbl TEXT; BEGIN
    FOR tbl IN SELECT unnest(ARRAY[
        'academic_terms',
        'academic_years',
        'cbc_classes',
        'cbc_class_teachers',
        'cbc_learning_areas',
        'cbc_parents',
        'cbc_student_enrollments',
        'cbc_student_parents',
        'cbc_students',
        'cbc_schools',
        'cbc_streams',
        'timetable_allocations',
        'fee_categories',
        'fee_templates',
        'import_jobs',
        'import_job_staging',
        'invitations',
        'invoices',
        'invoice_items',
        'medical_incidents',
        'member_active_school',
        'memberships',
        'payments',
        'users',
        'school_member_counts',
        'student_health_profiles',
        'timetable_blocks',
        'attendance_records',
        'behavior_categories',
        'behavior_notes',
        'attendance_term_summaries',
        'cbc_attendance_sessions',
        'cbc_strands',
        'cbc_sub_strands',
        'performance_indicators',
        'grading_scale_profiles',
        'grading_scale_ranges',
        'assessment_sessions',
        'student_assessment_scores',
        'student_assessment_outcome_grades',
        'student_term_subject_summaries',
        'student_term_overall_summaries',
        'student_cohort_position_summaries',
        'student_subject_strand_summaries',
        'student_performance_projections',
        'student_behavior_term_summaries',
        'teacher_subject_performance_summaries',
        'teacher_delivery_summaries',
        'teacher_workload_summaries',
        'class_learning_area_term_summaries',
        'class_term_attendance_summaries'
    ]) LOOP
        EXECUTE format('DROP POLICY IF EXISTS tenant_isolation_policy ON %I', tbl);
        EXECUTE format('ALTER TABLE IF EXISTS %I DISABLE ROW LEVEL SECURITY', tbl);
    END LOOP;
END $$;

DROP FUNCTION IF EXISTS fn_pending_invitation_by_email CASCADE;
DROP FUNCTION IF EXISTS fn_resolve_session CASCADE;
DROP FUNCTION IF EXISTS fn_rls_tenant_policy CASCADE;
