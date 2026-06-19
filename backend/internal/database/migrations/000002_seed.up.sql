-- ================================================================================
-- 🌱 SEED DATA: EDUCATION SYSTEMS AND GRADES (DYNAMIC CTE APPROACH)
-- ================================================================================

-- -----------------------------------------------------------------------------
-- 1. KENYA CBC
-- -----------------------------------------------------------------------------
WITH seeded_cbc AS (
    INSERT INTO education_systems (id, name, country_code) 
    VALUES (gen_random_uuid(), 'Kenya CBC', 'KE')
    ON CONFLICT (name) DO UPDATE SET country_code = EXCLUDED.country_code
    RETURNING id
),

-- Step 1: Seed the Curriculum Stages linked to Kenya CBC
seeded_stages AS (
    INSERT INTO curriculum_stages (id, education_system_id, name, code) VALUES
        (gen_random_uuid(), (SELECT id FROM seeded_cbc), 'Early Years Education', 'CBC_EARLY_YEARS'),
        (gen_random_uuid(), (SELECT id FROM seeded_cbc), 'Lower Primary', 'CBC_LOWER_PRIMARY'),
        (gen_random_uuid(), (SELECT id FROM seeded_cbc), 'Upper Primary', 'CBC_UPPER_PRIMARY'),
        (gen_random_uuid(), (SELECT id FROM seeded_cbc), 'Junior Secondary', 'CBC_JUNIOR_SECONDARY'),
        (gen_random_uuid(), (SELECT id FROM seeded_cbc), 'Senior School', 'CBC_SENIOR_SCHOOL')
    ON CONFLICT (education_system_id, code) DO UPDATE SET name = EXCLUDED.name
    RETURNING id, code
),

-- Step 2: Seed the Assessment Types (how a term is broken down) per Stage
seeded_assessment_types AS (
    INSERT INTO assessment_types (id, curriculum_stage_id, name, code, weight_contribution) VALUES
        -- Early Years & Lower Primary (100% pure classroom formative)
        (gen_random_uuid(), (SELECT id FROM seeded_stages WHERE code = 'CBC_EARLY_YEARS'), 'Continuous School-Based Assessment', 'CBA_FORMATIVE', 100.00),
        (gen_random_uuid(), (SELECT id FROM seeded_stages WHERE code = 'CBC_LOWER_PRIMARY'), 'Continuous School-Based Assessment', 'CBA_FORMATIVE', 100.00),
        
        -- Upper Primary (60% School CBA / 40% National Exam Split)
        (gen_random_uuid(), (SELECT id FROM seeded_stages WHERE code = 'CBC_UPPER_PRIMARY'), 'Classroom Portfolios & Practicals', 'CBA_SCHOOL_ACCUMULATED', 60.00),
        (gen_random_uuid(), (SELECT id FROM seeded_stages WHERE code = 'CBC_UPPER_PRIMARY'), 'Kenya Primary School Education Assessment (KPSEA)', 'KPSEA_NATIONAL_SUMMATIVE', 40.00),
        
        -- Junior Secondary (60% Formative + Terminal Blend)
        (gen_random_uuid(), (SELECT id FROM seeded_stages WHERE code = 'CBC_JUNIOR_SECONDARY'), 'Continuous Assessment Tests (CATs) & Projects', 'JSS_FORMATIVE', 60.00),
        (gen_random_uuid(), (SELECT id FROM seeded_stages WHERE code = 'CBC_JUNIOR_SECONDARY'), 'Kenya Junior Secondary Education Assessment (KJSEA)', 'KJSEA_NATIONAL_SUMMATIVE', 40.00),
        
        -- Senior School (Shifts to heavy track testing and terminal mocks)
        (gen_random_uuid(), (SELECT id FROM seeded_stages WHERE code = 'CBC_SENIOR_SCHOOL'), 'School Performance & Specialized Pathway Practicals', 'SSS_FORMATIVE', 40.00),
        (gen_random_uuid(), (SELECT id FROM seeded_stages WHERE code = 'CBC_SENIOR_SCHOOL'), 'Senior School Certificate Examination', 'SSCE_NATIONAL_SUMMATIVE', 60.00)
    ON CONFLICT (curriculum_stage_id, code) DO NOTHING
),

-- Step 3: Seed the Grade Scale Rubrics & Percentage Bands per Stage
seeded_scales AS (
    INSERT INTO assessment_grade_scales (id, curriculum_stage_id, grade_key, description, min_percentage, max_percentage, points) VALUES
        -- Early Years & Lower Primary: Pure Qualitative Rubrics (No percentages or numeric points)
        (gen_random_uuid(), (SELECT id FROM seeded_stages WHERE code = 'CBC_EARLY_YEARS'), 'EE', 'Exceeding Expectations', NULL, NULL, NULL),
        (gen_random_uuid(), (SELECT id FROM seeded_stages WHERE code = 'CBC_EARLY_YEARS'), 'ME', 'Meeting Expectations', NULL, NULL, NULL),
        (gen_random_uuid(), (SELECT id FROM seeded_stages WHERE code = 'CBC_EARLY_YEARS'), 'AE', 'Approaching Expectations', NULL, NULL, NULL),
        (gen_random_uuid(), (SELECT id FROM seeded_stages WHERE code = 'CBC_EARLY_YEARS'), 'BE', 'Below Expectations', NULL, NULL, NULL),
        
        (gen_random_uuid(), (SELECT id FROM seeded_stages WHERE code = 'CBC_LOWER_PRIMARY'), 'EE', 'Exceeding Expectations', NULL, NULL, NULL),
        (gen_random_uuid(), (SELECT id FROM seeded_stages WHERE code = 'CBC_LOWER_PRIMARY'), 'ME', 'Meeting Expectations', NULL, NULL, NULL),
        (gen_random_uuid(), (SELECT id FROM seeded_stages WHERE code = 'CBC_LOWER_PRIMARY'), 'AE', 'Approaching Expectations', NULL, NULL, NULL),
        (gen_random_uuid(), (SELECT id FROM seeded_stages WHERE code = 'CBC_LOWER_PRIMARY'), 'BE', 'Below Expectations', NULL, NULL, NULL),

        -- Upper Primary: Rubrics map to numerical percentage thresholds
        (gen_random_uuid(), (SELECT id FROM seeded_stages WHERE code = 'CBC_UPPER_PRIMARY'), 'EE', 'Exceeding Expectations', 80.00, 100.00, NULL),
        (gen_random_uuid(), (SELECT id FROM seeded_stages WHERE code = 'CBC_UPPER_PRIMARY'), 'ME', 'Meeting Expectations', 60.00, 79.99, NULL),
        (gen_random_uuid(), (SELECT id FROM seeded_stages WHERE code = 'CBC_UPPER_PRIMARY'), 'AE', 'Approaching Expectations', 40.00, 59.99, NULL),
        (gen_random_uuid(), (SELECT id FROM seeded_stages WHERE code = 'CBC_UPPER_PRIMARY'), 'BE', 'Below Expectations', 0.00, 39.99, NULL),

        -- Junior Secondary: 8-Level Advanced Granular Scale
        (gen_random_uuid(), (SELECT id FROM seeded_stages WHERE code = 'CBC_JUNIOR_SECONDARY'), 'EE1', 'Exceeding Expectations Tier 1', 90.00, 100.00, 8),
        (gen_random_uuid(), (SELECT id FROM seeded_stages WHERE code = 'CBC_JUNIOR_SECONDARY'), 'EE2', 'Exceeding Expectations Tier 2', 75.00, 89.99, 7),
        (gen_random_uuid(), (SELECT id FROM seeded_stages WHERE code = 'CBC_JUNIOR_SECONDARY'), 'ME1', 'Meeting Expectations Tier 1', 58.00, 74.99, 6),
        (gen_random_uuid(), (SELECT id FROM seeded_stages WHERE code = 'CBC_JUNIOR_SECONDARY'), 'ME2', 'Meeting Expectations Tier 2', 41.00, 57.99, 5),
        (gen_random_uuid(), (SELECT id FROM seeded_stages WHERE code = 'CBC_JUNIOR_SECONDARY'), 'AE1', 'Approaching Expectations Tier 1', 31.00, 40.99, 4),
        (gen_random_uuid(), (SELECT id FROM seeded_stages WHERE code = 'CBC_JUNIOR_SECONDARY'), 'AE2', 'Approaching Expectations Tier 2', 21.00, 30.99, 3),
        (gen_random_uuid(), (SELECT id FROM seeded_stages WHERE code = 'CBC_JUNIOR_SECONDARY'), 'BE1', 'Below Expectations Tier 1', 11.00, 20.99, 2),
        (gen_random_uuid(), (SELECT id FROM seeded_stages WHERE code = 'CBC_JUNIOR_SECONDARY'), 'BE2', 'Below Expectations Tier 2', 0.00, 10.99, 1),

        -- Senior School: Traditional 12-point system mapping letter grades to raw weights
        (gen_random_uuid(), (SELECT id FROM seeded_stages WHERE code = 'CBC_SENIOR_SCHOOL'), 'A', 'Plain', 80.00, 100.00, 12),
        (gen_random_uuid(), (SELECT id FROM seeded_stages WHERE code = 'CBC_SENIOR_SCHOOL'), 'A-', 'Minus', 75.00, 79.99, 11),
        (gen_random_uuid(), (SELECT id FROM seeded_stages WHERE code = 'CBC_SENIOR_SCHOOL'), 'B+', 'Plus', 70.00, 74.99, 10),
        (gen_random_uuid(), (SELECT id FROM seeded_stages WHERE code = 'CBC_SENIOR_SCHOOL'), 'B', 'Plain', 65.00, 69.99, 9),
        (gen_random_uuid(), (SELECT id FROM seeded_stages WHERE code = 'CBC_SENIOR_SCHOOL'), 'B-', 'Minus', 60.00, 64.99, 8),
        (gen_random_uuid(), (SELECT id FROM seeded_stages WHERE code = 'CBC_SENIOR_SCHOOL'), 'C+', 'Plus', 55.00, 59.99, 7),
        (gen_random_uuid(), (SELECT id FROM seeded_stages WHERE code = 'CBC_SENIOR_SCHOOL'), 'C', 'Plain', 50.00, 54.99, 6),
        (gen_random_uuid(), (SELECT id FROM seeded_stages WHERE code = 'CBC_SENIOR_SCHOOL'), 'C-', 'Minus', 45.00, 49.99, 5),
        (gen_random_uuid(), (SELECT id FROM seeded_stages WHERE code = 'CBC_SENIOR_SCHOOL'), 'D+', 'Plus', 40.00, 44.99, 4),
        (gen_random_uuid(), (SELECT id FROM seeded_stages WHERE code = 'CBC_SENIOR_SCHOOL'), 'D', 'Plain', 35.00, 39.99, 3),
        (gen_random_uuid(), (SELECT id FROM seeded_stages WHERE code = 'CBC_SENIOR_SCHOOL'), 'D-', 'Minus', 30.00, 34.99, 2),
        (gen_random_uuid(), (SELECT id FROM seeded_stages WHERE code = 'CBC_SENIOR_SCHOOL'), 'E', 'Failing', 0.00, 29.99, 1)
    ON CONFLICT (curriculum_stage_id, grade_key) DO NOTHING
)

-- Step 4: Final query inserts the individual grades with their direct stage pointers
INSERT INTO grades (id, education_system_id, curriculum_stage_id, name, sequence_order) VALUES
    (gen_random_uuid(), (SELECT id FROM seeded_cbc), (SELECT id FROM seeded_stages WHERE code = 'CBC_EARLY_YEARS'), 'PP1', 1),
    (gen_random_uuid(), (SELECT id FROM seeded_cbc), (SELECT id FROM seeded_stages WHERE code = 'CBC_EARLY_YEARS'), 'PP2', 2),
    
    (gen_random_uuid(), (SELECT id FROM seeded_cbc), (SELECT id FROM seeded_stages WHERE code = 'CBC_LOWER_PRIMARY'), 'Grade 1', 3),
    (gen_random_uuid(), (SELECT id FROM seeded_cbc), (SELECT id FROM seeded_stages WHERE code = 'CBC_LOWER_PRIMARY'), 'Grade 2', 4),
    (gen_random_uuid(), (SELECT id FROM seeded_cbc), (SELECT id FROM seeded_stages WHERE code = 'CBC_LOWER_PRIMARY'), 'Grade 3', 5),
    
    (gen_random_uuid(), (SELECT id FROM seeded_cbc), (SELECT id FROM seeded_stages WHERE code = 'CBC_UPPER_PRIMARY'), 'Grade 4', 6),
    (gen_random_uuid(), (SELECT id FROM seeded_cbc), (SELECT id FROM seeded_stages WHERE code = 'CBC_UPPER_PRIMARY'), 'Grade 5', 7),
    (gen_random_uuid(), (SELECT id FROM seeded_cbc), (SELECT id FROM seeded_stages WHERE code = 'CBC_UPPER_PRIMARY'), 'Grade 6', 8),
    
    (gen_random_uuid(), (SELECT id FROM seeded_cbc), (SELECT id FROM seeded_stages WHERE code = 'CBC_JUNIOR_SECONDARY'), 'Grade 7 (JSS 1)', 9),
    (gen_random_uuid(), (SELECT id FROM seeded_cbc), (SELECT id FROM seeded_stages WHERE code = 'CBC_JUNIOR_SECONDARY'), 'Grade 8 (JSS 2)', 10),
    (gen_random_uuid(), (SELECT id FROM seeded_cbc), (SELECT id FROM seeded_stages WHERE code = 'CBC_JUNIOR_SECONDARY'), 'Grade 9 (JSS 3)', 11),
    
    (gen_random_uuid(), (SELECT id FROM seeded_cbc), (SELECT id FROM seeded_stages WHERE code = 'CBC_SENIOR_SCHOOL'), 'Grade 10 (SSS 1)', 12),
    (gen_random_uuid(), (SELECT id FROM seeded_cbc), (SELECT id FROM seeded_stages WHERE code = 'CBC_SENIOR_SCHOOL'), 'Grade 11 (SSS 2)', 13),
    (gen_random_uuid(), (SELECT id FROM seeded_cbc), (SELECT id FROM seeded_stages WHERE code = 'CBC_SENIOR_SCHOOL'), 'Grade 12 (SSS 3)', 14)
ON CONFLICT (education_system_id, id) DO NOTHING;


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