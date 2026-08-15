DROP TRIGGER IF EXISTS trg_membership_count ON memberships;
DROP TRIGGER IF EXISTS trg_student_count ON cbc_students;
DROP FUNCTION IF EXISTS trg_update_membership_count();
DROP FUNCTION IF EXISTS trg_update_student_count();
DROP TABLE IF EXISTS member_counts;
