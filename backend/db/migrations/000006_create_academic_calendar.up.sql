-- Migration: 000006_create_academic_calendar
-- Purpose: Create academic years, terms, public holidays, and school events
--   to support the multi-country SIS curriculum and operations tracking.
--
-- Dependencies:
--   - 000001_init_extensions (pgcrypto for gen_random_uuid()).
--   - 000004_create_sis_schema (provides schools, countries tables).
--   - 000005_create_subject_hierarchy (provides education_systems table, though not directly used here).
--
-- Usage:
--   migrate -database "$DATABASE_URL" -path backend/db/migrations up

-- ============================================================================
-- Section 1: academic_years
-- ============================================================================

-- Academic years scoped to schools. A school may have multiple academic years
-- over time (e.g., 2026, 2027). The name field provides a human-readable
-- identifier; start_date/end_date define the calendar boundaries.
CREATE TABLE academic_years (
    id            UUID        NOT NULL    DEFAULT gen_random_uuid()
                                       PRIMARY KEY,
    school_id     UUID        NOT NULL    REFERENCES schools(id)
                                       ON DELETE CASCADE,
    name          VARCHAR(64) NOT NULL,
    start_date    DATE        NOT NULL,
    end_date      DATE        NOT NULL,
    created_at    TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL    DEFAULT NOW()
);

-- Index: academic_years_school_id_idx covers school-scoped queries.
CREATE INDEX academic_years_school_id_idx ON academic_years (school_id);

-- Index: academic_years_name_idx supports name lookups (e.g., finding a specific year).
CREATE INDEX academic_years_name_idx ON academic_years (name);

-- Constraint: academic_years_school_name_unique ensures a school cannot have
-- duplicate academic year names.
ALTER TABLE academic_years ADD CONSTRAINT academic_years_school_name_unique
    UNIQUE (school_id, name);

-- ============================================================================
-- Section 2: academic_terms
-- ============================================================================

-- Academic terms (semester, quarter, term 1/2/3) scoped to an academic year.
-- Each term has its own date range for scheduling and grading purposes.
CREATE TABLE academic_terms (
    id              UUID        NOT NULL    DEFAULT gen_random_uuid()
                                         PRIMARY KEY,
    academic_year_id UUID     NOT NULL    REFERENCES academic_years(id)
                                         ON DELETE CASCADE,
    name            VARCHAR(64) NOT NULL,
    start_date      DATE        NOT NULL,
    end_date        DATE        NOT NULL,
    created_at      TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL    DEFAULT NOW()
);

-- Index: academic_terms_academic_year_id_idx covers academic-year-scoped queries.
CREATE INDEX academic_terms_academic_year_id_idx ON academic_terms (academic_year_id);

-- Index: academic_terms_name_idx supports term name lookups within an academic year.
CREATE INDEX academic_terms_name_idx ON academic_terms (name);

-- Constraint: academic_terms_year_name_unique ensures a term name is unique
-- within its academic year (e.g., "Term 1" within a year, but "Term 1" can
-- exist in different academic years).
ALTER TABLE academic_terms ADD CONSTRAINT academic_terms_year_name_unique
    UNIQUE (academic_year_id, name);

-- ============================================================================
-- Section 3: public_holidays
-- ============================================================================

-- Public holidays declared at the country level. These are fixed-date
-- holidays that apply nationwide (e.g., Madaraka Day, Jamhuri Day, Labour Day).
CREATE TABLE public_holidays (
    id         UUID        NOT NULL    DEFAULT gen_random_uuid()
                                   PRIMARY KEY,
    country_id UUID        NOT NULL    REFERENCES countries(id)
                                   ON DELETE CASCADE,
    name       VARCHAR(255) NOT NULL,
    date       DATE        NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL    DEFAULT NOW()
);

-- Index: public_holidays_country_id_idx covers country-scoped holiday queries.
CREATE INDEX public_holidays_country_id_idx ON public_holidays (country_id);

-- Index: public_holidays_date_idx supports date-based lookups (e.g., "what holidays
-- are on this date?").
CREATE INDEX public_holidays_date_idx ON public_holidays (date);

-- Index: public_holidays_country_date_idx is a composite index for efficient
-- country+date queries.
CREATE INDEX public_holidays_country_date_idx ON public_holidays (country_id, date);

-- ============================================================================
-- Section 4: school_events
-- ============================================================================

-- School-specific events such as sports days, admission days, exams, etc.
-- Events may span multiple days and may optionally require student attendance tracking.
CREATE TABLE school_events (
    id              UUID        NOT NULL    DEFAULT gen_random_uuid()
                                       PRIMARY KEY,
    school_id       UUID        NOT NULL    REFERENCES schools(id)
                                         ON DELETE CASCADE,
    title           VARCHAR(255) NOT NULL,
    event_type      VARCHAR(64)  NOT NULL,
    start_date      DATE        NOT NULL,
    end_date        DATE        NOT NULL,
    requires_attendance BOOLEAN  NOT NULL  DEFAULT FALSE,
    created_at      TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL    DEFAULT NOW()
);

-- Index: school_events_school_id_idx covers school-scoped event queries.
CREATE INDEX school_events_school_id_idx ON school_events (school_id);

-- Index: school_events_event_type_idx supports filtering by event type
-- (e.g., show all SPORTS events).
CREATE INDEX school_events_event_type_idx ON school_events (event_type);

-- Index: school_events_date_range_idx supports date range queries
-- (e.g., events happening within a given period).
CREATE INDEX school_events_date_range_idx ON school_events (start_date, end_date);

-- ============================================================================
-- Section 5: updated_at triggers
-- ============================================================================

-- Reuse the set_updated_at() helper from migration 000004.
CREATE TRIGGER academic_years_updated_at_trg
    BEFORE UPDATE ON academic_years
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER academic_terms_updated_at_trg
    BEFORE UPDATE ON academic_terms
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER public_holidays_updated_at_trg
    BEFORE UPDATE ON public_holidays
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER school_events_updated_at_trg
    BEFORE UPDATE ON school_events
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- ============================================================================
-- Section 6: Comments (documentation)
-- ============================================================================

COMMENT ON TABLE academic_years IS 'Academic years scoped to a school (e.g., 2026, 2026-2027). Used for scheduling, grading periods.';
COMMENT ON COLUMN academic_years.id IS 'Auto-generated UUID primary key.';
COMMENT ON COLUMN academic_years.school_id IS 'FK to schools(id). Cascades on school delete.';
COMMENT ON COLUMN academic_years.name IS 'Human-readable year identifier (e.g., "2026", "2026-2027"). Unique per school.';
COMMENT ON COLUMN academic_years.start_date IS 'First day of the academic year.';
COMMENT ON COLUMN academic_years.end_date IS 'Last day of the academic year.';
COMMENT ON COLUMN academic_years.created_at IS 'UTC timestamp of row creation.';
COMMENT ON COLUMN academic_years.updated_at IS 'UTC timestamp of last modification.';

COMMENT ON TABLE academic_terms IS 'Academic terms (semester, quarter, term 1/2/3) within an academic year.';
COMMENT ON COLUMN academic_terms.id IS 'Auto-generated UUID primary key.';
COMMENT ON COLUMN academic_terms.academic_year_id IS 'FK to academic_years(id). Cascades on academic year delete.';
COMMENT ON COLUMN academic_terms.name IS 'Term name (e.g., "Term 1", "Semester 1"). Unique within an academic year.';
COMMENT ON COLUMN academic_terms.start_date IS 'First day of the term.';
COMMENT ON COLUMN academic_terms.end_date IS 'Last day of the term.';
COMMENT ON COLUMN academic_terms.created_at IS 'UTC timestamp of row creation.';
COMMENT ON COLUMN academic_terms.updated_at IS 'UTC timestamp of last modification.';

COMMENT ON TABLE public_holidays IS 'National public holidays at the country level (e.g., Madaraka Day, Labour Day).';
COMMENT ON COLUMN public_holidays.id IS 'Auto-generated UUID primary key.';
COMMENT ON COLUMN public_holidays.country_id IS 'FK to countries(id). Cascades on country delete.';
COMMENT ON COLUMN public_holidays.name IS 'Holiday name (e.g., "Madaraka Day", "Labour Day").';
COMMENT ON COLUMN public_holidays.date IS 'The calendar date of the holiday.';
COMMENT ON COLUMN public_holidays.created_at IS 'UTC timestamp of row creation.';
COMMENT ON COLUMN public_holidays.updated_at IS 'UTC timestamp of last modification.';

COMMENT ON TABLE school_events IS 'School-specific events (sports, exams, admission days, etc.) with optional attendance tracking.';
COMMENT ON COLUMN school_events.id IS 'Auto-generated UUID primary key.';
COMMENT ON COLUMN school_events.school_id IS 'FK to schools(id). Cascades on school delete.';
COMMENT ON COLUMN school_events.title IS 'Event title (e.g., "Inter-House Sports Day").';
COMMENT ON COLUMN school_events.event_type IS 'Event type: SPORTS, ADMISSION, EXAM, etc.';
COMMENT ON COLUMN school_events.start_date IS 'First day of the event.';
COMMENT ON COLUMN school_events.end_date IS 'Last day of the event.';
COMMENT ON COLUMN school_events.requires_attendance IS 'Whether student attendance must be tracked for this event.';
COMMENT ON COLUMN school_events.created_at IS 'UTC timestamp of row creation.';
COMMENT ON COLUMN school_events.updated_at IS 'UTC timestamp of last modification.';