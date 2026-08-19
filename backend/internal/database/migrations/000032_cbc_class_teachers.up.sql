-- Migration: 000032_cbc_class_teachers
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: cbc_class_teachers

CREATE TABLE IF NOT EXISTS cbc_class_teachers (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID         NOT NULL,
    class_id         UUID         NOT NULL,
    user_id          UUID         NOT NULL,
    learning_area_id UUID         NULL,
    teacher_role     teacher_role NOT NULL DEFAULT 'SUBJECT_TEACHER',
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_cbc_class_teachers_tenant_user
        FOREIGN KEY (tenant_id, user_id)
        REFERENCES users(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_cbc_class_teachers_tenant_class
        FOREIGN KEY (tenant_id, class_id)
        REFERENCES cbc_classes(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_cbc_class_teachers_tenant_learning_area
        FOREIGN KEY (tenant_id, learning_area_id)
        REFERENCES cbc_learning_areas(tenant_id, id) ON DELETE SET NULL (learning_area_id),
    CONSTRAINT chk_cct_primary_no_area CHECK (
        teacher_role != 'PRIMARY_CLASS_TEACHER' OR learning_area_id IS NULL
    ),
    CONSTRAINT chk_cct_subject_area_required CHECK (
        teacher_role != 'SUBJECT_TEACHER' OR learning_area_id IS NOT NULL
    ),
    CONSTRAINT unique_cbc_class_teacher UNIQUE (class_id, user_id, learning_area_id)
);

CREATE INDEX IF NOT EXISTS idx_cbc_class_teachers_tenant   ON cbc_class_teachers (tenant_id);
CREATE INDEX IF NOT EXISTS idx_cbc_class_teachers_class_id ON cbc_class_teachers (class_id);
CREATE INDEX IF NOT EXISTS idx_cbc_class_teachers_user_id  ON cbc_class_teachers (user_id);
CREATE INDEX IF NOT EXISTS idx_cbc_class_teachers_role     ON cbc_class_teachers (teacher_role);

-- Only one PRIMARY_CLASS_TEACHER per class
CREATE UNIQUE INDEX IF NOT EXISTS idx_cbc_one_primary_per_class
    ON cbc_class_teachers (class_id)
    WHERE teacher_role = 'PRIMARY_CLASS_TEACHER';





CREATE TRIGGER trg_cbc_class_teachers_updated_at
    BEFORE UPDATE ON cbc_class_teachers
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN cbc_class_teachers.updated_at IS
    'Tracks teacher assignment changes mid-term.';
