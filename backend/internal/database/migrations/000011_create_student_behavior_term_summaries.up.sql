-- Migration: 000011_create_student_behavior_term_summaries
-- Creates the student_behavior_term_summaries materialised table, the
-- category_type classification for behavior_categories, and the trigger
-- that keeps the summary in sync when behavior notes are inserted/updated.
--
-- Grain: (student_id, academic_term_id)
--
-- This table is an incremental materialised summary of APPROVED and
-- INCLUDED_IN_REPORT behavior notes per student per term. When a behavior
-- note transitions to APPROVED or INCLUDED_IN_REPORT, the summary is
-- refreshed for that student+term. PENDING_REVIEW and REJECTED notes are
-- excluded from the main counts but included in pending_review_count and
-- resolved_count for admin visibility.
--
-- primary_category_id is the behavior category with the highest count
-- among notes in this term (APPROVED + INCLUDED_IN_REPORT only). Ties
-- are resolved by the most recent note's category_id.

-- ============================================================================
-- ENRICHMENT: Add category_type to behavior_categories
-- Allows the system to distinguish commendations from disciplinary incidents.
-- ============================================================================

DO $$ BEGIN
    CREATE TYPE behavior_category_type AS ENUM ('COMMENDATION', 'DISCIPLINARY', 'OTHER');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

ALTER TABLE IF EXISTS behavior_categories
    ADD COLUMN IF NOT EXISTS category_type behavior_category_type NOT NULL DEFAULT 'DISCIPLINARY';

COMMENT ON COLUMN behavior_categories.category_type IS
    'Classification of the behavior category: COMMENDATION (positive/laudable
     behaviour), DISCIPLINARY (negative behaviour / infraction), or OTHER.
     Used by student_behavior_term_summaries to compute commendations_count
     and disciplinary_count. Defaults to DISCIPLINARY for existing categories.';

-- ============================================================================
-- TABLE: student_behavior_term_summaries
-- ============================================================================

CREATE TABLE IF NOT EXISTS student_behavior_term_summaries (
    id                    UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             UUID          NOT NULL,
    school_id             UUID          NOT NULL,
    student_id            UUID          NOT NULL,
    academic_term_id      UUID          NOT NULL,
    total_incidents       INT           NOT NULL DEFAULT 0,
    urgent_count          INT           NOT NULL DEFAULT 0,
    commendations_count   INT           NOT NULL DEFAULT 0,
    disciplinary_count    INT           NOT NULL DEFAULT 0,
    pending_review_count  INT           NOT NULL DEFAULT 0,
    resolved_count        INT           NOT NULL DEFAULT 0,
    primary_category_id   UUID,                   -- category with highest count (or NULL if none)
    last_refreshed_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    created_at            TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_student_behavior_term UNIQUE (student_id, academic_term_id),
    CONSTRAINT fk_behavior_summaries_tenant_student
        FOREIGN KEY (tenant_id, student_id)
        REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_behavior_summaries_tenant_term
        FOREIGN KEY (tenant_id, school_id, academic_term_id)
        REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_behavior_summaries_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_behavior_summaries_category
        FOREIGN KEY (primary_category_id)
        REFERENCES behavior_categories(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_behavior_summaries_tenant
    ON student_behavior_term_summaries (tenant_id);
CREATE INDEX IF NOT EXISTS idx_behavior_summaries_school
    ON student_behavior_term_summaries (school_id);
CREATE INDEX IF NOT EXISTS idx_behavior_summaries_student_term
    ON student_behavior_term_summaries (student_id, academic_term_id);
CREATE INDEX IF NOT EXISTS idx_behavior_summaries_term
    ON student_behavior_term_summaries (academic_term_id);

DROP TRIGGER IF EXISTS trg_student_behavior_term_summaries_updated_at
    ON student_behavior_term_summaries;
CREATE TRIGGER trg_student_behavior_term_summaries_updated_at
    BEFORE UPDATE ON student_behavior_term_summaries
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

COMMENT ON TABLE student_behavior_term_summaries IS
    'Incremental materialised summary of behavior notes per student per term.
     Only counts APPROVED and INCLUDED_IN_REPORT notes in the main totals
     (total_incidents, urgent_count, commendations_count, disciplinary_count,
     primary_category_id). pending_review_count counts all PENDING_REVIEW
     notes for admin visibility. Refreshed via trigger on behavior_notes
     insert/update.';

COMMENT ON COLUMN student_behavior_term_summaries.total_incidents IS
    'Total count of APPROVED + INCLUDED_IN_REPORT behavior notes for this
     student+term. Excludes PENDING_REVIEW and REJECTED.';

COMMENT ON COLUMN student_behavior_term_summaries.urgent_count IS
    'Count of APPROVED + INCLUDED_IN_REPORT notes where is_urgent = true.';

COMMENT ON COLUMN student_behavior_term_summaries.commendations_count IS
    'Count of APPROVED + INCLUDED_IN_REPORT notes whose category has
     category_type = COMMENDATION.';

COMMENT ON COLUMN student_behavior_term_summaries.disciplinary_count IS
    'Count of APPROVED + INCLUDED_IN_REPORT notes whose category has
     category_type = DISCIPLINARY.';

COMMENT ON COLUMN student_behavior_term_summaries.pending_review_count IS
    'Count of PENDING_REVIEW notes for this student+term (regardless of
     approval status). Provides admin visibility into backlog.';

COMMENT ON COLUMN student_behavior_term_summaries.resolved_count IS
    'Count of notes with status in (APPROVED, INCLUDED_IN_REPORT, REJECTED)
     — any note that has been acted upon. total_incidents + pending_review_count
     + rejected notes = all notes for the term.';

COMMENT ON COLUMN student_behavior_term_summaries.primary_category_id IS
    'The behavior category with the highest count among APPROVED +
     INCLUDED_IN_REPORT notes for this student+term. Ties are resolved
     by the most recent note''s category. NULL when no applicable notes exist.';

-- ============================================================================
-- FUNCTION: fn_refresh_student_behavior_term_summary(target_student_id UUID,
--                                                     target_term_id UUID)
--
-- Recomputes the student_behavior_term_summary row for the given student+term
-- from scratch (idempotent upsert).
-- ============================================================================

CREATE OR REPLACE FUNCTION fn_refresh_student_behavior_term_summary(
    target_student_id UUID,
    target_term_id UUID
)
RETURNS void
LANGUAGE plpgsql
AS $$
DECLARE
    v_tenant_id UUID;
    v_school_id UUID;
BEGIN
    -- Resolve tenant_id and school_id from the student's enrollment in this term
    SELECT enr.tenant_id, enr.school_id
    INTO v_tenant_id, v_school_id
    FROM cbc_student_enrollments enr
    WHERE enr.student_id = target_student_id
      AND enr.academic_term_id = target_term_id
    LIMIT 1;

    -- If the student is not enrolled in this term, nothing to do
    IF NOT FOUND THEN
        RETURN;
    END IF;

    -- Upsert the summary row
    INSERT INTO student_behavior_term_summaries (
        tenant_id, school_id, student_id, academic_term_id,
        total_incidents, urgent_count,
        commendations_count, disciplinary_count,
        pending_review_count, resolved_count,
        primary_category_id, last_refreshed_at
    )
    WITH approved_notes AS (
        -- Only APPROVED and INCLUDED_IN_REPORT notes count toward main totals
        SELECT *
        FROM behavior_notes
        WHERE student_id = target_student_id
          AND tenant_id = v_tenant_id
          AND status IN ('APPROVED', 'INCLUDED_IN_REPORT')
    ),
    all_term_notes AS (
        -- All notes for this student+term (for pending/resolved counts)
        SELECT *
        FROM behavior_notes
        WHERE student_id = target_student_id
          AND tenant_id = v_tenant_id
    ),
    category_counts AS (
        SELECT
            category_id,
            COUNT(*) AS cnt,
            MAX(created_at) AS most_recent
        FROM approved_notes
        GROUP BY category_id
    ),
    ranked_categories AS (
        SELECT category_id
        FROM category_counts
        ORDER BY cnt DESC, most_recent DESC
        LIMIT 1
    )
    SELECT
        v_tenant_id,
        v_school_id,
        target_student_id,
        target_term_id,

        -- total_incidents: count of APPROVED + INCLUDED_IN_REPORT
        (SELECT COUNT(*) FROM approved_notes)::INT,

        -- urgent_count: those flagged urgent among approved
        (SELECT COUNT(*) FROM approved_notes WHERE is_urgent = true)::INT,

        -- commendations_count: approved notes in COMMENDATION categories
        (SELECT COUNT(*)
         FROM approved_notes an
         JOIN behavior_categories bc ON bc.id = an.category_id
         WHERE bc.category_type = 'COMMENDATION')::INT,

        -- disciplinary_count: approved notes in DISCIPLINARY categories
        (SELECT COUNT(*)
         FROM approved_notes an
         JOIN behavior_categories bc ON bc.id = an.category_id
         WHERE bc.category_type = 'DISCIPLINARY')::INT,

        -- pending_review_count: all PENDING_REVIEW notes (not just approved)
        (SELECT COUNT(*)
         FROM all_term_notes
         WHERE status = 'PENDING_REVIEW')::INT,

        -- resolved_count: notes with status APPROVED, INCLUDED_IN_REPORT, or REJECTED
        (SELECT COUNT(*)
         FROM all_term_notes
         WHERE status IN ('APPROVED', 'INCLUDED_IN_REPORT', 'REJECTED'))::INT,

        -- primary_category_id: highest-count category (tie-break by most recent)
        (SELECT category_id FROM ranked_categories),

        NOW()

    ON CONFLICT (student_id, academic_term_id)
    DO UPDATE SET
        tenant_id            = EXCLUDED.tenant_id,
        school_id            = EXCLUDED.school_id,
        total_incidents      = EXCLUDED.total_incidents,
        urgent_count         = EXCLUDED.urgent_count,
        commendations_count  = EXCLUDED.commendations_count,
        disciplinary_count   = EXCLUDED.disciplinary_count,
        pending_review_count = EXCLUDED.pending_review_count,
        resolved_count       = EXCLUDED.resolved_count,
        primary_category_id  = EXCLUDED.primary_category_id,
        last_refreshed_at    = NOW(),
        updated_at           = NOW();
END;
$$;

COMMENT ON FUNCTION fn_refresh_student_behavior_term_summary IS
    'Refreshes student_behavior_term_summaries for a single student+term.
     Idempotent — safe to call on INSERT or UPDATE of any behavior note.';

-- ============================================================================
-- FUNCTION: fn_refresh_student_behavior_term_summary_for_note()
-- Trigger function that resolves the student+term from the affected note
-- and calls fn_refresh_student_behavior_term_summary.
-- ============================================================================

CREATE OR REPLACE FUNCTION fn_refresh_student_behavior_term_summary_for_note()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    v_term_id UUID;
BEGIN
    -- Find the academic term that contains this note's date.
    -- We join the note's timetable_slot → class → enrollments → academic_term.
    SELECT enr.academic_term_id INTO v_term_id
    FROM cbc_student_enrollments enr
    WHERE enr.student_id = COALESCE(NEW.student_id, OLD.student_id)
      AND enr.status = 'ACTIVE'
      -- Use the note's date to find which term it falls in
      AND enr.academic_term_id IN (
          SELECT at.id
          FROM academic_terms at
          WHERE at.tenant_id = COALESCE(NEW.tenant_id, OLD.tenant_id)
            AND at.school_id = COALESCE(NEW.school_id, OLD.school_id)
            AND COALESCE(NEW.date, OLD.date) BETWEEN at.start_date AND at.end_date
      )
    LIMIT 1;

    IF FOUND THEN
        PERFORM fn_refresh_student_behavior_term_summary(
            COALESCE(NEW.student_id, OLD.student_id),
            v_term_id
        );
    END IF;

    RETURN COALESCE(NEW, OLD);
END;
$$;

COMMENT ON FUNCTION fn_refresh_student_behavior_term_summary_for_note IS
    'Trigger function: on INSERT or UPDATE of behavior_notes, resolves
     the student+enrollment term from the note''s date and refreshes the
     student_behavior_term_summary.';

-- ============================================================================
-- TRIGGER: trg_behavior_notes_refresh_term_summary
-- Fires AFTER INSERT OR UPDATE on behavior_notes.
-- Calls fn_refresh_student_behavior_term_summary_for_note for the affected note.
-- ============================================================================

DROP TRIGGER IF EXISTS trg_behavior_notes_refresh_term_summary
    ON behavior_notes;
CREATE TRIGGER trg_behavior_notes_refresh_term_summary
    AFTER INSERT OR UPDATE ON behavior_notes
    FOR EACH ROW
    EXECUTE FUNCTION fn_refresh_student_behavior_term_summary_for_note();

COMMENT ON TRIGGER trg_behavior_notes_refresh_term_summary ON behavior_notes IS
    'After a behavior note is inserted or updated, refresh the
     student_behavior_term_summary for the affected student+term.';

-- ============================================================================
-- RLS POLICY
-- ============================================================================

ALTER TABLE IF EXISTS student_behavior_term_summaries ENABLE ROW LEVEL SECURITY;

DO $$ BEGIN
    DROP POLICY IF EXISTS tenant_isolation_policy ON student_behavior_term_summaries;
    CREATE POLICY tenant_isolation_policy ON student_behavior_term_summaries
        FOR ALL
        USING (tenant_id = fn_current_tenant_id())
        WITH CHECK (tenant_id = fn_current_tenant_id());
END $$;

COMMENT ON TABLE student_behavior_term_summaries IS
    'Incremental materialised summary of behavior notes per student per term.
     RLS-enabled — tenant-scoped.';
