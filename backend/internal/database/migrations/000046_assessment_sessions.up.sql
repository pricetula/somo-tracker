-- Migration: 000045_assessment_sessions
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: assessment_sessions

CREATE TABLE IF NOT EXISTS assessment_sessions (
    id                      UUID                        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID                        NOT NULL,
    school_id               UUID                        NOT NULL,
    class_id                UUID                        NOT NULL,
    learning_area_id        UUID                        NOT NULL,
    academic_term_id        UUID                        NOT NULL,
    academic_year_id        UUID                        NOT NULL,
    name                    VARCHAR(255)                NOT NULL,
    evaluation_method       assessment_evaluation_method NOT NULL,
    max_points              NUMERIC(10,2)               NULL,
    grading_scale_profile_id UUID                       NULL,
    status                  assessment_session_status   NOT NULL DEFAULT 'DRAFT',
    rejection_comment       TEXT                        NULL,
    submitted_by            UUID                        NULL,
    approved_by             UUID                        NULL,
    scheduled_date          DATE                        NULL,
    created_at              TIMESTAMPTZ                 NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ                 NOT NULL DEFAULT NOW(),
    created_by              UUID                        NOT NULL,

    CONSTRAINT fk_assessment_sessions_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_assessment_sessions_tenant_class
        FOREIGN KEY (tenant_id, class_id)
        REFERENCES cbc_classes(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_assessment_sessions_tenant_term
        FOREIGN KEY (tenant_id, school_id, academic_term_id)
        REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_assessment_sessions_learning_area
        FOREIGN KEY (tenant_id, learning_area_id)
        REFERENCES cbc_learning_areas(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_assessment_sessions_tenant_scale_profile
        FOREIGN KEY (tenant_id, grading_scale_profile_id)
        REFERENCES grading_scale_profiles(tenant_id, id) ON DELETE SET NULL (grading_scale_profile_id),
    CONSTRAINT fk_assessment_sessions_tenant_submitted_by
        FOREIGN KEY (tenant_id, submitted_by)
        REFERENCES users(tenant_id, id) ON DELETE SET NULL (submitted_by),
    CONSTRAINT fk_assessment_sessions_tenant_approved_by
        FOREIGN KEY (tenant_id, approved_by)
        REFERENCES users(tenant_id, id) ON DELETE SET NULL (approved_by),
    CONSTRAINT fk_assessment_sessions_tenant_created_by
        FOREIGN KEY (tenant_id, created_by)
        REFERENCES users(tenant_id, id),
    CONSTRAINT uq_assessment_sessions_tenant UNIQUE (tenant_id, id),
    CONSTRAINT chk_quantitative_has_points CHECK (
        evaluation_method != 'QUANTITATIVE' OR max_points IS NOT NULL
    ),
    CONSTRAINT chk_quantitative_has_scale CHECK (
        evaluation_method != 'QUANTITATIVE' OR grading_scale_profile_id IS NOT NULL
    ),
    CONSTRAINT chk_rubric_no_points CHECK (
        evaluation_method != 'RUBRIC' OR max_points IS NULL
    ),
    CONSTRAINT chk_rubric_no_scale CHECK (
        evaluation_method != 'RUBRIC' OR grading_scale_profile_id IS NULL
    )
);

CREATE INDEX IF NOT EXISTS idx_assessment_sessions_tenant
    ON assessment_sessions (tenant_id);
CREATE INDEX IF NOT EXISTS idx_assessment_sessions_school
    ON assessment_sessions (school_id);
CREATE INDEX IF NOT EXISTS idx_assessment_sessions_class
    ON assessment_sessions (class_id);
CREATE INDEX IF NOT EXISTS idx_assessment_sessions_term
    ON assessment_sessions (academic_term_id);
CREATE INDEX IF NOT EXISTS idx_assessment_sessions_status
    ON assessment_sessions (status);
CREATE INDEX IF NOT EXISTS idx_assessment_sessions_learning_area
    ON assessment_sessions (learning_area_id);

DROP TRIGGER IF EXISTS trg_assessment_sessions_updated_at ON assessment_sessions;
CREATE TRIGGER trg_assessment_sessions_updated_at
    BEFORE UPDATE ON assessment_sessions
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

COMMENT ON TABLE assessment_sessions IS
    'Tracks CBC assessment sessions through their lifecycle:
     DRAFT (teacher creating/grading) → PENDING_APPROVAL (submitted to admin)
     → PUBLISHED (approved, visible to parents). Rejection returns to DRAFT.
     Supports two evaluation methods: QUANTITATIVE (total marks converted via
     grading scale) and RUBRIC (direct indicator-level grading).';

COMMENT ON COLUMN assessment_sessions.max_points IS
    'Total possible marks for QUANTITATIVE sessions. NULL for RUBRIC sessions.
     Cannot be updated once any student score rows exist.';

COMMENT ON COLUMN assessment_sessions.rejection_comment IS
    'Admin feedback when rejecting a session. Cleared on re-submission.';

-- max_points write-once enforcement (000003 item 6b)
CREATE OR REPLACE FUNCTION fn_block_assessment_max_points_update()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.max_points IS DISTINCT FROM OLD.max_points THEN
        IF EXISTS (
            SELECT 1 FROM student_assessment_scores
            WHERE session_id = OLD.id
            LIMIT 1
        ) THEN
            RAISE EXCEPTION 'Cannot update max_points for session (id: %) — student assessment scores already exist for this session', OLD.id
                USING ERRCODE = 'P0002';  -- assigned application-level error code
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_assessment_sessions_max_points_immutable ON assessment_sessions;
CREATE TRIGGER trg_assessment_sessions_max_points_immutable
    BEFORE UPDATE ON assessment_sessions
    FOR EACH ROW
    WHEN (OLD.max_points IS DISTINCT FROM NEW.max_points)
    EXECUTE FUNCTION fn_block_assessment_max_points_update();

COMMENT ON TRIGGER trg_assessment_sessions_max_points_immutable ON assessment_sessions IS
    'Enforces that max_points cannot be changed after any student assessment
     score rows reference this session. Throws error code P0002 which the
     application can catch specifically.';
