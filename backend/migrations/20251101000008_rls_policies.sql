-- Goose migration
-- +goose Up
-- 00008: Row Level Security policies

CREATE INDEX IF NOT EXISTS idx_student_guardian_guardian ON student_guardian (guardian_id);
CREATE INDEX IF NOT EXISTS idx_course_offering_teacher_user ON course_offering_teacher (user_id);

-- 6a. reference data: any member reads, managers write
DO $$
DECLARE t text;
BEGIN
  FOREACH t IN ARRAY ARRAY[
    'academic_year','academic_term','school_calendar_event','grade_level','class_room',
    'education_system','subject','topic','subtopic','learning_outcome',
    'assessment_category','grading_scale','course_offering'
  ] LOOP
    EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
    EXECUTE format('DROP POLICY IF EXISTS rls_read ON %I', t);
    EXECUTE format('CREATE POLICY rls_read ON %I FOR SELECT USING (app.is_member(school_id))', t);
    EXECUTE format('DROP POLICY IF EXISTS rls_write ON %I', t);
    EXECUTE format('CREATE POLICY rls_write ON %I FOR ALL USING (app.is_manager(school_id)) WITH CHECK (app.is_manager(school_id))', t);
  END LOOP;
END $$;

-- 6b. staff-only reference data
DO $$
DECLARE t text;
BEGIN
  FOREACH t IN ARRAY ARRAY[
    'school_academic_setting','behaviour_category','incident_type',
    'course_offering_teacher','course_enrollment'
  ] LOOP
    EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
    EXECUTE format('DROP POLICY IF EXISTS rls_read ON %I', t);
    EXECUTE format('CREATE POLICY rls_read ON %I FOR SELECT USING (app.is_staff(school_id))', t);
    EXECUTE format('DROP POLICY IF EXISTS rls_write ON %I', t);
    EXECUTE format('CREATE POLICY rls_write ON %I FOR ALL USING (app.is_manager(school_id)) WITH CHECK (app.is_manager(school_id))', t);
  END LOOP;
END $$;

-- 6c. grading bands
ALTER TABLE grading_scale_band ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS rls_read ON grading_scale_band;
CREATE POLICY rls_read ON grading_scale_band FOR SELECT
  USING (EXISTS (SELECT 1 FROM grading_scale s WHERE s.id = grading_scale_id AND app.is_member(s.school_id)));
DROP POLICY IF EXISTS rls_write ON grading_scale_band;
CREATE POLICY rls_write ON grading_scale_band FOR ALL
  USING (EXISTS (SELECT 1 FROM grading_scale s WHERE s.id = grading_scale_id AND app.is_manager(s.school_id)))
  WITH CHECK (EXISTS (SELECT 1 FROM grading_scale s WHERE s.id = grading_scale_id AND app.is_manager(s.school_id)));

-- 6d. people
ALTER TABLE student ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS rls_read ON student;
CREATE POLICY rls_read ON student FOR SELECT USING (app.can_view_people(school_id) OR app.guardian_of(id));
DROP POLICY IF EXISTS rls_write ON student;
CREATE POLICY rls_write ON student FOR ALL USING (app.is_manager(school_id)) WITH CHECK (app.is_manager(school_id));

ALTER TABLE student_enrollment ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS rls_read ON student_enrollment;
CREATE POLICY rls_read ON student_enrollment FOR SELECT USING (app.can_view_people(school_id) OR app.guardian_of(student_id));
DROP POLICY IF EXISTS rls_write ON student_enrollment;
CREATE POLICY rls_write ON student_enrollment FOR ALL USING (app.is_manager(school_id)) WITH CHECK (app.is_manager(school_id));

ALTER TABLE guardian ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS rls_read ON guardian;
CREATE POLICY rls_read ON guardian FOR SELECT
  USING (user_id = app.user_id() OR app.in_tenant_manager(tenant_id) OR app.staff_sees_guardian(id));
DROP POLICY IF EXISTS rls_write ON guardian;
CREATE POLICY rls_write ON guardian FOR ALL USING (app.in_tenant_manager(tenant_id)) WITH CHECK (app.in_tenant_manager(tenant_id));

ALTER TABLE student_guardian ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS rls_read ON student_guardian;
CREATE POLICY rls_read ON student_guardian FOR SELECT USING (app.can_view_student(student_id) OR app.guardian_is_me(guardian_id));
DROP POLICY IF EXISTS rls_write ON student_guardian;
CREATE POLICY rls_write ON student_guardian FOR ALL USING (app.manages_student(student_id)) WITH CHECK (app.manages_student(student_id));

-- 6e. syllabus coverage
ALTER TABLE syllabus_coverage ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS rls_read ON syllabus_coverage;
CREATE POLICY rls_read ON syllabus_coverage FOR SELECT USING (app.is_staff(school_id));
DROP POLICY IF EXISTS rls_write ON syllabus_coverage;
CREATE POLICY rls_write ON syllabus_coverage FOR ALL
  USING (app.is_manager(school_id) OR app.teaches(course_offering_id))
  WITH CHECK (app.is_manager(school_id) OR app.teaches(course_offering_id));

-- 6f. attendance
ALTER TABLE attendance_session ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS rls_read ON attendance_session;
CREATE POLICY rls_read ON attendance_session FOR SELECT USING (app.is_staff(school_id));
DROP POLICY IF EXISTS rls_insert ON attendance_session;
CREATE POLICY rls_insert ON attendance_session FOR INSERT WITH CHECK (app.can_mark_attendance(school_id, class_room_id, course_offering_id));
DROP POLICY IF EXISTS rls_update ON attendance_session;
CREATE POLICY rls_update ON attendance_session FOR UPDATE
  USING (app.is_manager(school_id) OR (status = 'open' AND app.can_mark_attendance(school_id, class_room_id, course_offering_id)))
  WITH CHECK (app.is_manager(school_id) OR app.can_mark_attendance(school_id, class_room_id, course_offering_id));
DROP POLICY IF EXISTS rls_delete ON attendance_session;
CREATE POLICY rls_delete ON attendance_session FOR DELETE USING (app.is_manager(school_id));

ALTER TABLE attendance_record ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS rls_read ON attendance_record;
CREATE POLICY rls_read ON attendance_record FOR SELECT USING (app.is_staff(school_id) OR app.guardian_of(student_id));
DROP POLICY IF EXISTS rls_write ON attendance_record;
CREATE POLICY rls_write ON attendance_record FOR ALL
  USING (app.attendance_session_editable(attendance_session_id))
  WITH CHECK (app.attendance_session_editable(attendance_session_id));

-- 6g. assessments
ALTER TABLE assessment ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS rls_read ON assessment;
CREATE POLICY rls_read ON assessment FOR SELECT USING (app.is_staff(school_id));
DROP POLICY IF EXISTS rls_read_guardian ON assessment;
CREATE POLICY rls_read_guardian ON assessment FOR SELECT USING (app.guardian_sees_assessment(id));
DROP POLICY IF EXISTS rls_write ON assessment;
CREATE POLICY rls_write ON assessment FOR ALL
  USING (app.is_manager(school_id) OR app.teaches(course_offering_id))
  WITH CHECK (app.is_manager(school_id) OR app.teaches(course_offering_id));

ALTER TABLE assessment_result ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS rls_read ON assessment_result;
CREATE POLICY rls_read ON assessment_result FOR SELECT
  USING (app.is_staff(school_id) OR (app.guardian_of(student_id) AND app.assessment_published(assessment_id)));
DROP POLICY IF EXISTS rls_write ON assessment_result;
CREATE POLICY rls_write ON assessment_result FOR ALL
  USING (app.is_manager(school_id) OR app.teaches_assessment(assessment_id))
  WITH CHECK (app.is_manager(school_id) OR app.teaches_assessment(assessment_id));

DO $$
DECLARE t text;
BEGIN
  FOREACH t IN ARRAY ARRAY['assessment_outcome','assessment_outcome_result','quiz'] LOOP
    EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
    EXECUTE format('DROP POLICY IF EXISTS rls_read ON %I', t);
    EXECUTE format('CREATE POLICY rls_read ON %I FOR SELECT USING (app.is_staff(school_id))', t);
    EXECUTE format('DROP POLICY IF EXISTS rls_write ON %I', t);
    EXECUTE format('CREATE POLICY rls_write ON %I FOR ALL USING (app.is_manager(school_id) OR app.teaches_assessment(assessment_id)) WITH CHECK (app.is_manager(school_id) OR app.teaches_assessment(assessment_id))', t);
  END LOOP;
END $$;

-- 6h. question bank
ALTER TABLE question ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS rls_read ON question;
CREATE POLICY rls_read ON question FOR SELECT USING (app.is_staff(school_id));
DROP POLICY IF EXISTS rls_write ON question;
CREATE POLICY rls_write ON question FOR ALL
  USING (app.is_manager(school_id) OR created_by = app.user_id())
  WITH CHECK (app.is_manager(school_id) OR (app.is_staff(school_id) AND created_by = app.user_id()));

ALTER TABLE question_option ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS rls_read ON question_option;
CREATE POLICY rls_read ON question_option FOR SELECT USING (app.is_staff(school_id));
DROP POLICY IF EXISTS rls_write ON question_option;
CREATE POLICY rls_write ON question_option FOR ALL
  USING (app.can_edit_question(question_id)) WITH CHECK (app.can_edit_question(question_id));

ALTER TABLE quiz_question ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS rls_read ON quiz_question;
CREATE POLICY rls_read ON quiz_question FOR SELECT USING (app.is_staff(school_id));
DROP POLICY IF EXISTS rls_write ON quiz_question;
CREATE POLICY rls_write ON quiz_question FOR ALL
  USING (app.is_manager(school_id) OR app.teaches_assessment(quiz_id))
  WITH CHECK (app.is_manager(school_id) OR app.teaches_assessment(quiz_id));

-- 6i. behaviour
ALTER TABLE behaviour_record ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS rls_read ON behaviour_record;
CREATE POLICY rls_read ON behaviour_record FOR SELECT
  USING (app.is_staff(school_id) OR (guardian_visible AND app.guardian_of(student_id)));
DROP POLICY IF EXISTS rls_insert ON behaviour_record;
CREATE POLICY rls_insert ON behaviour_record FOR INSERT
  WITH CHECK (app.is_staff(school_id) AND recorded_by = app.user_id());
DROP POLICY IF EXISTS rls_update ON behaviour_record;
CREATE POLICY rls_update ON behaviour_record FOR UPDATE
  USING (app.is_manager(school_id) OR recorded_by = app.user_id())
  WITH CHECK (app.is_staff(school_id));
DROP POLICY IF EXISTS rls_delete ON behaviour_record;
CREATE POLICY rls_delete ON behaviour_record FOR DELETE USING (app.is_manager(school_id) OR recorded_by = app.user_id());

-- 6j. incidents
ALTER TABLE incident ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS rls_read ON incident;
CREATE POLICY rls_read ON incident FOR SELECT
  USING (app.is_manager(school_id) OR reported_by = app.user_id()
         OR (app.is_staff(school_id) AND NOT is_confidential));
DROP POLICY IF EXISTS rls_insert ON incident;
CREATE POLICY rls_insert ON incident FOR INSERT
  WITH CHECK (app.is_staff(school_id) AND reported_by = app.user_id());
DROP POLICY IF EXISTS rls_update ON incident;
CREATE POLICY rls_update ON incident FOR UPDATE
  USING (app.is_manager(school_id) OR (reported_by = app.user_id() AND status = 'reported'))
  WITH CHECK (app.is_manager(school_id) OR reported_by = app.user_id());
DROP POLICY IF EXISTS rls_delete ON incident;
CREATE POLICY rls_delete ON incident FOR DELETE USING (app.is_manager(school_id));

ALTER TABLE incident_participant ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS rls_read ON incident_participant;
CREATE POLICY rls_read ON incident_participant FOR SELECT USING (app.can_see_incident(incident_id));
DROP POLICY IF EXISTS rls_write ON incident_participant;
CREATE POLICY rls_write ON incident_participant FOR ALL
  USING (app.can_edit_incident(incident_id)) WITH CHECK (app.can_edit_incident(incident_id));

ALTER TABLE incident_attachment ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS rls_read ON incident_attachment;
CREATE POLICY rls_read ON incident_attachment FOR SELECT USING (app.can_see_incident(incident_id));
DROP POLICY IF EXISTS rls_write ON incident_attachment;
CREATE POLICY rls_write ON incident_attachment FOR ALL
  USING (app.can_edit_incident(incident_id)) WITH CHECK (app.can_edit_incident(incident_id));

DO $$
DECLARE t text;
BEGIN
  FOREACH t IN ARRAY ARRAY['incident_action','incident_guardian_notification'] LOOP
    EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
    EXECUTE format('DROP POLICY IF EXISTS rls_read ON %I', t);
    EXECUTE format('CREATE POLICY rls_read ON %I FOR SELECT USING (app.can_see_incident(incident_id))', t);
    EXECUTE format('DROP POLICY IF EXISTS rls_write ON %I', t);
    EXECUTE format('CREATE POLICY rls_write ON %I FOR ALL USING (app.is_manager(school_id)) WITH CHECK (app.is_manager(school_id))', t);
  END LOOP;
END $$;

-- 6k. report cards
ALTER TABLE report_card ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS rls_read ON report_card;
CREATE POLICY rls_read ON report_card FOR SELECT
  USING (app.is_staff(school_id) OR (status = 'published' AND app.guardian_of_enrollment(student_enrollment_id)));
DROP POLICY IF EXISTS rls_insert ON report_card;
CREATE POLICY rls_insert ON report_card FOR INSERT WITH CHECK (app.is_manager(school_id));
DROP POLICY IF EXISTS rls_update ON report_card;
CREATE POLICY rls_update ON report_card FOR UPDATE
  USING (app.is_manager(school_id) OR app.is_homeroom_of_enrollment(student_enrollment_id))
  WITH CHECK (app.is_manager(school_id) OR app.is_homeroom_of_enrollment(student_enrollment_id));
DROP POLICY IF EXISTS rls_delete ON report_card;
CREATE POLICY rls_delete ON report_card FOR DELETE USING (app.is_manager(school_id));

ALTER TABLE report_card_subject ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS rls_read ON report_card_subject;
CREATE POLICY rls_read ON report_card_subject FOR SELECT
  USING (app.is_staff(school_id) OR app.guardian_sees_report_card(report_card_id));
DROP POLICY IF EXISTS rls_insert ON report_card_subject;
CREATE POLICY rls_insert ON report_card_subject FOR INSERT WITH CHECK (app.is_manager(school_id));
DROP POLICY IF EXISTS rls_update ON report_card_subject;
CREATE POLICY rls_update ON report_card_subject FOR UPDATE
  USING (app.is_manager(school_id) OR app.teaches(course_offering_id))
  WITH CHECK (app.is_manager(school_id) OR app.teaches(course_offering_id));
DROP POLICY IF EXISTS rls_delete ON report_card_subject;
CREATE POLICY rls_delete ON report_card_subject FOR DELETE USING (app.is_manager(school_id));

-- 6l. audit log
ALTER TABLE locked_edit_log ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS rls_read ON locked_edit_log;
CREATE POLICY rls_read ON locked_edit_log FOR SELECT USING (app.is_manager(school_id));
DROP POLICY IF EXISTS rls_insert ON locked_edit_log;
CREATE POLICY rls_insert ON locked_edit_log FOR INSERT
  WITH CHECK (app.is_manager(school_id) AND changed_by = app.user_id());

-- +goose Down
DO $$
DECLARE r record;
BEGIN
  FOR r IN SELECT schemaname, tablename, policyname FROM pg_policies WHERE schemaname = 'public' AND policyname LIKE 'rls\_%' LOOP
    EXECUTE format('DROP POLICY IF EXISTS %I ON %I.%I', r.policyname, r.schemaname, r.tablename);
  END LOOP;
  FOR r IN SELECT c.relname FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE n.nspname = 'public' AND c.relkind = 'r' AND c.relrowsecurity LOOP
    EXECUTE format('ALTER TABLE %I DISABLE ROW LEVEL SECURITY', r.relname);
  END LOOP;
END $$;
DROP INDEX IF EXISTS idx_student_guardian_guardian;
DROP INDEX IF EXISTS idx_course_offering_teacher_user;
