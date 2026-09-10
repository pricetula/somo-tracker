-- Migration: 000006_create_academic_calendar (down)
-- Purpose: Rollback academic years, terms, public holidays, and school events.
-- Usage:
--   migrate -database "$DATABASE_URL" -path backend/db/migrations down

-- Drop triggers first.
DROP TRIGGER IF EXISTS school_events_updated_at_trg ON school_events;
DROP TRIGGER IF EXISTS public_holidays_updated_at_trg ON public_holidays;
DROP TRIGGER IF EXISTS academic_terms_updated_at_trg ON academic_terms;
DROP TRIGGER IF EXISTS academic_years_updated_at_trg ON academic_years;

-- Drop tables in reverse dependency order.
DROP TABLE IF EXISTS school_events;
DROP TABLE IF EXISTS public_holidays;
DROP TABLE IF EXISTS academic_terms;
DROP TABLE IF EXISTS academic_years;
