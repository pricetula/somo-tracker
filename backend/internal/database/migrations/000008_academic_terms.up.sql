-- Migration: 000008_academic_terms
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: academic_terms

CREATE TABLE IF NOT EXISTS academic_terms (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID         NOT NULL,
    school_id        UUID         NOT NULL,
    academic_year_id UUID         NOT NULL,
    name             VARCHAR(100) NOT NULL,
    term_number      SMALLINT     NOT NULL,
    start_date       DATE         NOT NULL,
    end_date         DATE         NOT NULL,
    is_current       BOOLEAN      NOT NULL DEFAULT false,
    is_final         BOOLEAN      NOT NULL DEFAULT false,
    version          INTEGER      NOT NULL DEFAULT 1,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_by       UUID         NOT NULL,
    updated_by       UUID         NOT NULL,

    CONSTRAINT chk_term_dates   CHECK (start_date < end_date),
    CONSTRAINT chk_term_number  CHECK (term_number BETWEEN 1 AND 3),
    CONSTRAINT uq_academic_terms_tenant        UNIQUE (tenant_id, id),
    CONSTRAINT fk_academic_terms_tenant_created_by
        FOREIGN KEY (tenant_id, created_by)
        REFERENCES users(tenant_id, id),
    CONSTRAINT fk_academic_terms_tenant_updated_by
        FOREIGN KEY (tenant_id, updated_by)
        REFERENCES users(tenant_id, id),
    CONSTRAINT uq_academic_terms_tenant_school UNIQUE (tenant_id, school_id, id),

    CONSTRAINT fk_academic_terms_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,

    CONSTRAINT fk_academic_terms_tenant_year
        FOREIGN KEY (tenant_id, school_id, academic_year_id)
        REFERENCES academic_years(tenant_id, school_id, id) ON DELETE CASCADE,

    CONSTRAINT EXCL_academic_terms_no_overlap EXCLUDE USING gist (
        school_id WITH =,
        daterange(start_date, end_date, '[]') WITH &&
    )
);

CREATE INDEX IF NOT EXISTS idx_academic_terms_tenant_id ON academic_terms (tenant_id);
-- BUG FIX: was incorrectly targeting academic_years; fixed to academic_terms
CREATE INDEX IF NOT EXISTS idx_academic_terms_school_id ON academic_terms (school_id);
CREATE INDEX IF NOT EXISTS idx_academic_terms_year_id   ON academic_terms (academic_year_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_one_current_term_per_school
    ON academic_terms (school_id) WHERE is_current = TRUE;

CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_term_number_per_year
    ON academic_terms (academic_year_id, term_number);

DROP TRIGGER IF EXISTS trg_academic_terms_updated_at ON academic_terms;
CREATE TRIGGER trg_academic_terms_updated_at
    BEFORE UPDATE ON academic_terms
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

-- ---------------------------------------------------------------------------
-- TERM BOUNDS VALIDATION TRIGGER
-- ---------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION fn_validate_term_dates_within_year()
RETURNS TRIGGER AS $$
DECLARE
    v_year_start DATE;
    v_year_end   DATE;
BEGIN
    SELECT start_date, end_date INTO v_year_start, v_year_end
    FROM academic_years
    WHERE id = NEW.academic_year_id;

    IF (NEW.start_date < v_year_start OR NEW.end_date > v_year_end) THEN
        RAISE EXCEPTION 'Term dates (% to %) must fall within parent Academic Year bounds (% to %)',
            NEW.start_date, NEW.end_date, v_year_start, v_year_end;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_validate_term_dates ON academic_terms;
CREATE TRIGGER trg_validate_term_dates
    BEFORE INSERT OR UPDATE ON academic_terms
    FOR EACH ROW EXECUTE FUNCTION fn_validate_term_dates_within_year();

COMMENT ON COLUMN academic_terms.term_number IS
    'Kenya CBC operates a 3-term academic year. term_number enforces this:
     1 = Term 1, 2 = Term 2, 3 = Term 3.';

COMMENT ON COLUMN academic_terms.is_final IS
    'Marks the last term of the academic year before a national KNEC exam cycle
     (KPSEA at end of G6, KJSEA at end of G9, KSSEA at end of G12). The
     application uses this flag to lock SBA submissions and trigger KNEC sync
     workflows. Set to TRUE only on Term 3 of an exam year.';
