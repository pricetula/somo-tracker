-- ================================================================================
-- 🌱 SEED DATA: EDUCATION SYSTEMS AND GRADES (DYNAMIC CTE APPROACH)
-- ================================================================================

-- -----------------------------------------------------------------------------
-- 1. KENYA CBC
-- -----------------------------------------------------------------------------
WITH seeded_cbc AS (
    INSERT INTO education_systems (id, name, country_code) 
    VALUES (gen_random_uuid(), 'Kenya CBC', 'KE')
    -- Assumes a UNIQUE constraint on 'name' to handle re-runs safely
    ON CONFLICT (name) DO UPDATE SET country_code = EXCLUDED.country_code
    RETURNING id
)
INSERT INTO grades (id, education_system_id, name, sequence_order) VALUES
    (gen_random_uuid(), (SELECT id FROM seeded_cbc), 'PP1', 1),
    (gen_random_uuid(), (SELECT id FROM seeded_cbc), 'PP2', 2),
    (gen_random_uuid(), (SELECT id FROM seeded_cbc), 'Grade 1', 3),
    (gen_random_uuid(), (SELECT id FROM seeded_cbc), 'Grade 2', 4),
    (gen_random_uuid(), (SELECT id FROM seeded_cbc), 'Grade 3', 5),
    (gen_random_uuid(), (SELECT id FROM seeded_cbc), 'Grade 4', 6),
    (gen_random_uuid(), (SELECT id FROM seeded_cbc), 'Grade 5', 7),
    (gen_random_uuid(), (SELECT id FROM seeded_cbc), 'Grade 6', 8),
    (gen_random_uuid(), (SELECT id FROM seeded_cbc), 'Grade 7 (JSS 1)', 9),
    (gen_random_uuid(), (SELECT id FROM seeded_cbc), 'Grade 8 (JSS 2)', 10),
    (gen_random_uuid(), (SELECT id FROM seeded_cbc), 'Grade 9 (JSS 3)', 11),
    (gen_random_uuid(), (SELECT id FROM seeded_cbc), 'Grade 10 (SSS 1)', 12),
    (gen_random_uuid(), (SELECT id FROM seeded_cbc), 'Grade 11 (SSS 2)', 13),
    (gen_random_uuid(), (SELECT id FROM seeded_cbc), 'Grade 12 (SSS 3)', 14);


-- -----------------------------------------------------------------------------
-- 2. CAMBRIDGE IGCSE
-- -----------------------------------------------------------------------------
WITH seeded_cambridge AS (
    INSERT INTO education_systems (id, name, country_code) 
    VALUES (gen_random_uuid(), 'Cambridge IGCSE', 'GB')
    ON CONFLICT (name) DO UPDATE SET country_code = EXCLUDED.country_code
    RETURNING id
)
INSERT INTO grades (id, education_system_id, name, sequence_order) VALUES
    (gen_random_uuid(), (SELECT id FROM seeded_cambridge), 'Year 1', 1),
    (gen_random_uuid(), (SELECT id FROM seeded_cambridge), 'Year 2', 2),
    (gen_random_uuid(), (SELECT id FROM seeded_cambridge), 'Year 3', 3),
    (gen_random_uuid(), (SELECT id FROM seeded_cambridge), 'Year 4', 4),
    (gen_random_uuid(), (SELECT id FROM seeded_cambridge), 'Year 5', 5),
    (gen_random_uuid(), (SELECT id FROM seeded_cambridge), 'Year 6', 6),
    (gen_random_uuid(), (SELECT id FROM seeded_cambridge), 'Year 7', 7),
    (gen_random_uuid(), (SELECT id FROM seeded_cambridge), 'Year 8', 8),
    (gen_random_uuid(), (SELECT id FROM seeded_cambridge), 'Year 9', 9),
    (gen_random_uuid(), (SELECT id FROM seeded_cambridge), 'Year 10 (IGCSE 1)', 10),
    (gen_random_uuid(), (SELECT id FROM seeded_cambridge), 'Year 11 (IGCSE 2)', 11);


-- -----------------------------------------------------------------------------
-- 3. IB MYP
-- -----------------------------------------------------------------------------
WITH seeded_ib AS (
    INSERT INTO education_systems (id, name, country_code) 
    VALUES (gen_random_uuid(), 'IB MYP', 'CH')
    ON CONFLICT (name) DO UPDATE SET country_code = EXCLUDED.country_code
    RETURNING id
)
INSERT INTO grades (id, education_system_id, name, sequence_order) VALUES
    (gen_random_uuid(), (SELECT id FROM seeded_ib), 'MYP 1', 1),
    (gen_random_uuid(), (SELECT id FROM seeded_ib), 'MYP 2', 2),
    (gen_random_uuid(), (SELECT id FROM seeded_ib), 'MYP 3', 3),
    (gen_random_uuid(), (SELECT id FROM seeded_ib), 'MYP 4', 4),
    (gen_random_uuid(), (SELECT id FROM seeded_ib), 'MYP 5', 5);