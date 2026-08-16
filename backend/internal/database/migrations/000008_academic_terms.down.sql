-- Migration: 000008_academic_terms (rollback)
-- SomoTracker — rollback for 000008_academic_terms.

DROP FUNCTION IF EXISTS fn_validate_term_dates_within_year CASCADE;

DROP TABLE IF EXISTS academic_terms CASCADE;
