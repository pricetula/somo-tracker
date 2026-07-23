-- ================================================================================
-- 🌱 SEED DATA: KNEC Assessment Weight Configs + Dev Tenant + Demo School
-- ================================================================================
-- This seed file is a complete replacement of the previous multi-system seed.
-- CBC-only: no education_systems, curriculum_stages, grades, or other generic
-- catalog tables exist. Only KNEC official formula data is seeded.

-- ---------------------------------------------------------------------------
-- A) ASSESSMENT WEIGHT CONFIGS — official KNEC weighting formula (7 rows)
-- ---------------------------------------------------------------------------
-- KPSEA: 60% SBA (G4+G5) + 40% KPSEA written (G6)
-- KJSEA: 20% SBA (G7+G8) + 20% KPSEA result + 60% KJSEA written (G9)

INSERT INTO assessment_weight_configs (id, grade_level, assessment_type_code, target_exam, weight_percent, effective_from, notes) VALUES
    -- KPSEA (Primary School Transition) Breakdown: Cumulative 100%
    (gen_random_uuid(), 'G4', 'KNEC_SBA_Project',  'KPSEA', 20.00, 2023, 'Grade 4 SBA → 20% of KPSEA composite'),
    (gen_random_uuid(), 'G5', 'KNEC_SBA_Project',  'KPSEA', 20.00, 2023, 'Grade 5 SBA → 20% of KPSEA composite'),
    (gen_random_uuid(), 'G6', 'KNEC_SBA_Project',  'KPSEA', 20.00, 2023, 'Grade 6 SBA → 20% of KPSEA composite'),
    (gen_random_uuid(), 'G6', 'National_KPSEA',    'KPSEA', 40.00, 2023, 'KPSEA National Written Exam → 40% of KPSEA composite'),

    -- KJSEA (Junior Secondary Transition to Senior School) Breakdown: Cumulative 100%
    (gen_random_uuid(), 'G6', 'National_KPSEA',    'KJSEA', 20.00, 2024, 'KPSEA carry-forward score → 20% of Senior School placement'),
    (gen_random_uuid(), 'G7', 'KNEC_SBA_Project',  'KJSEA', 10.00, 2024, 'Grade 7 SBA → 10% (Half of the 20% JSS SBA pool)'),
    (gen_random_uuid(), 'G8', 'KNEC_SBA_Project',  'KJSEA', 10.00, 2024, 'Grade 8 SBA → 10% (Half of the 20% JSS SBA pool)'),
    (gen_random_uuid(), 'G9', 'National_KJSEA',    'KJSEA', 60.00, 2024, 'KJSEA National Exam → 60% of Senior School placement')
ON CONFLICT (grade_level, assessment_type_code, target_exam, effective_from) DO NOTHING;

-- ---------------------------------------------------------------------------
-- B) SAMPLE TENANT (dev/demo environment)
-- ---------------------------------------------------------------------------

INSERT INTO tenants (id, name, slug, stytch_org_id)
VALUES (
    gen_random_uuid(),
    'Demo School Trust',
    'demo-school-trust',
    'stytch-org-demo-001'
)
ON CONFLICT (slug) DO NOTHING;

-- ---------------------------------------------------------------------------
-- C) SAMPLE CBC SCHOOL
-- ---------------------------------------------------------------------------

INSERT INTO cbc_schools (id, tenant_id, name, knec_school_code, nemis_institution_code, county, sub_county, ward, school_type, is_active)
SELECT
    gen_random_uuid(),
    t.id,
    'Demo CBC Primary School',
    '12345678',
    'NEMIS-DEMO-001',
    'Nairobi',
    'Westlands',
    'Parklands',
    'Public',
    true
FROM tenants t
WHERE t.slug = 'demo-school-trust'
ON CONFLICT (tenant_id, id) DO NOTHING;
