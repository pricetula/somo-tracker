-- ================================================================================
-- 🌱 SEED DATA: EDUCATION SYSTEMS AND GRADES
-- ================================================================================

-- -----------------------------------------------------------------------------
-- 1. Insert Education Systems
-- -----------------------------------------------------------------------------
INSERT INTO education_systems (id, name, country_code) VALUES
    ('11111111-1111-1111-1111-111111111111', 'Kenya CBC', 'KE'),
    ('22222222-2222-2222-2222-222222222222', 'Cambridge IGCSE', 'GB'),
    ('33333333-3333-3333-3333-333333333333', 'IB MYP', 'CH')
ON CONFLICT (id) DO UPDATE 
SET name = EXCLUDED.name, country_code = EXCLUDED.country_code;

-- -----------------------------------------------------------------------------
-- 2. Insert Grades for Kenya CBC (Competency-Based Curriculum)
--    Focusing primarily on Junior Secondary (JSS) and Senior Secondary phases.
-- -----------------------------------------------------------------------------
INSERT INTO grades (id, education_system_id, name, sequence_order) VALUES
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', 'PP1', 1),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', 'PP2', 2),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', 'Grade 1', 3),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', 'Grade 2', 4),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', 'Grade 3', 5),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', 'Grade 4', 6),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', 'Grade 5', 7),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', 'Grade 6', 8),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', 'Grade 7 (JSS 1)', 9),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', 'Grade 8 (JSS 2)', 10),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', 'Grade 9 (JSS 3)', 11),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', 'Grade 10 (SSS 1)', 12),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', 'Grade 11 (SSS 2)', 13),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', 'Grade 12 (SSS 3)', 14);

-- -----------------------------------------------------------------------------
-- 3. Insert Grades for Cambridge IGCSE / British Curriculum
--    Standard progression covering Primary, Lower Secondary, and IGCSE years.
-- -----------------------------------------------------------------------------
INSERT INTO grades (id, education_system_id, name, sequence_order) VALUES
    (gen_random_uuid(), '22222222-2222-2222-2222-222222222222', 'Year 1', 1),
    (gen_random_uuid(), '22222222-2222-2222-2222-222222222222', 'Year 2', 2),
    (gen_random_uuid(), '22222222-2222-2222-2222-222222222222', 'Year 3', 3),
    (gen_random_uuid(), '22222222-2222-2222-2222-222222222222', 'Year 4', 4),
    (gen_random_uuid(), '22222222-2222-2222-2222-222222222222', 'Year 5', 5),
    (gen_random_uuid(), '22222222-2222-2222-2222-222222222222', 'Year 6', 6),
    (gen_random_uuid(), '22222222-2222-2222-2222-222222222222', 'Year 7', 7),
    (gen_random_uuid(), '22222222-2222-2222-2222-222222222222', 'Year 8', 8),
    (gen_random_uuid(), '22222222-2222-2222-2222-222222222222', 'Year 9', 9),
    (gen_random_uuid(), '22222222-2222-2222-2222-222222222222', 'Year 10 (IGCSE 1)', 10),
    (gen_random_uuid(), '22222222-2222-2222-2222-222222222222', 'Year 11 (IGCSE 2)', 11);

-- -----------------------------------------------------------------------------
-- 4. Insert Grades for International Baccalaureate Middle Years Programme (IB MYP)
--    The MYP spans a 5-year curriculum framework typically for ages 11 to 16.
-- -----------------------------------------------------------------------------
INSERT INTO grades (id, education_system_id, name, sequence_order) VALUES
    (gen_random_uuid(), '33333333-3333-3333-3333-333333333333', 'MYP 1', 1),
    (gen_random_uuid(), '33333333-3333-3333-3333-333333333333', 'MYP 2', 2),
    (gen_random_uuid(), '33333333-3333-3333-3333-333333333333', 'MYP 3', 3),
    (gen_random_uuid(), '33333333-3333-3333-3333-333333333333', 'MYP 4', 4),
    (gen_random_uuid(), '33333333-3333-3333-3333-333333333333', 'MYP 5', 5);