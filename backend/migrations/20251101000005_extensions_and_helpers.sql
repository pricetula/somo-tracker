-- Goose migration
-- +goose Up
-- 00005: Extensions and RLS helper functions

CREATE EXTENSION IF NOT EXISTS pg_uuidv7;

CREATE OR REPLACE FUNCTION set_updated_at() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$;

CREATE SCHEMA IF NOT EXISTS app;

CREATE OR REPLACE FUNCTION app.user_id() RETURNS uuid LANGUAGE sql STABLE AS $$
  SELECT NULLIF(current_setting('app.user_id', true), '')::uuid
$$;

CREATE OR REPLACE FUNCTION app.is_system_context(p_relid oid) RETURNS boolean LANGUAGE sql STABLE AS $$
  SELECT (SELECT relowner FROM pg_class WHERE oid = p_relid) = (SELECT oid FROM pg_roles WHERE rolname = current_user)
      OR (SELECT rolsuper FROM pg_roles WHERE rolname = current_user)
$$;

CREATE OR REPLACE FUNCTION app.has_role(p_school uuid, p_roles text[]) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public, pg_temp AS $$
  SELECT EXISTS (
    SELECT 1 FROM membership m
    WHERE m.user_id = app.user_id() AND m.school_id = p_school AND m.role = ANY (p_roles))
$$;

CREATE OR REPLACE FUNCTION app.is_member(p_school uuid) RETURNS boolean LANGUAGE sql STABLE AS $$
  SELECT app.has_role(p_school, ARRAY['tenant_owner','head','admin','teacher','finance','guardian','platform_operator'])
$$;
CREATE OR REPLACE FUNCTION app.is_manager(p_school uuid) RETURNS boolean LANGUAGE sql STABLE AS $$
  SELECT app.has_role(p_school, ARRAY['tenant_owner','head','admin'])
$$;
CREATE OR REPLACE FUNCTION app.is_staff(p_school uuid) RETURNS boolean LANGUAGE sql STABLE AS $$
  SELECT app.has_role(p_school, ARRAY['tenant_owner','head','admin','teacher'])
$$;
CREATE OR REPLACE FUNCTION app.can_view_people(p_school uuid) RETURNS boolean LANGUAGE sql STABLE AS $$
  SELECT app.has_role(p_school, ARRAY['tenant_owner','head','admin','teacher','finance'])
$$;

CREATE OR REPLACE FUNCTION app.in_tenant_manager(p_tenant uuid) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public, pg_temp AS $$
  SELECT EXISTS (
    SELECT 1 FROM membership m JOIN school s ON s.id = m.school_id
    WHERE m.user_id = app.user_id() AND s.tenant_id = p_tenant AND m.role IN ('tenant_owner','head','admin'))
$$;

CREATE OR REPLACE FUNCTION app.teaches(p_offering uuid) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public, pg_temp AS $$
  SELECT EXISTS (SELECT 1 FROM course_offering_teacher t
                 WHERE t.course_offering_id = p_offering AND t.user_id = app.user_id())
$$;

CREATE OR REPLACE FUNCTION app.is_homeroom(p_class_room uuid) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public, pg_temp AS $$
  SELECT EXISTS (SELECT 1 FROM class_room c
                 WHERE c.id = p_class_room AND c.homeroom_teacher_id = app.user_id())
$$;

CREATE OR REPLACE FUNCTION app.guardian_of(p_student uuid) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public, pg_temp AS $$
  SELECT EXISTS (
    SELECT 1 FROM student_guardian sg JOIN guardian g ON g.id = sg.guardian_id
    WHERE sg.student_id = p_student AND g.user_id = app.user_id() AND sg.can_view_records)
$$;

CREATE OR REPLACE FUNCTION app.guardian_is_me(p_guardian uuid) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public, pg_temp AS $$
  SELECT EXISTS (SELECT 1 FROM guardian g WHERE g.id = p_guardian AND g.user_id = app.user_id())
$$;

CREATE OR REPLACE FUNCTION app.can_mark_attendance(p_school uuid, p_class_room uuid, p_offering uuid) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public, pg_temp AS $$
  SELECT app.is_manager(p_school) OR app.is_homeroom(p_class_room) OR app.teaches(p_offering)
$$;

-- Additional helpers used later
CREATE OR REPLACE FUNCTION app.attendance_session_editable(p_session uuid) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public, pg_temp AS $$
  SELECT EXISTS (
    SELECT 1 FROM attendance_session s
    WHERE s.id = p_session
      AND (app.is_manager(s.school_id)
           OR (s.status = 'open' AND app.can_mark_attendance(s.school_id, s.class_room_id, s.course_offering_id))))
$$;

CREATE OR REPLACE FUNCTION app.teaches_assessment(p_assessment uuid) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public, pg_temp AS $$
  SELECT EXISTS (
    SELECT 1 FROM assessment a JOIN course_offering_teacher t ON t.course_offering_id = a.course_offering_id
    WHERE a.id = p_assessment AND t.user_id = app.user_id())
$$;

CREATE OR REPLACE FUNCTION app.assessment_published(p_assessment uuid) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public, pg_temp AS $$
  SELECT EXISTS (SELECT 1 FROM assessment a WHERE a.id = p_assessment AND a.status = 'published')
$$;

CREATE OR REPLACE FUNCTION app.guardian_sees_assessment(p_assessment uuid) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public, pg_temp AS $$
  SELECT EXISTS (
    SELECT 1 FROM assessment a
    JOIN course_enrollment ce ON ce.course_offering_id = a.course_offering_id
    JOIN student_enrollment se ON se.id = ce.student_enrollment_id
    WHERE a.id = p_assessment AND a.status = 'published' AND app.guardian_of(se.student_id))
$$;

CREATE OR REPLACE FUNCTION app.can_edit_question(p_question uuid) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public, pg_temp AS $$
  SELECT EXISTS (SELECT 1 FROM question q
                 WHERE q.id = p_question AND (app.is_manager(q.school_id) OR q.created_by = app.user_id()))
$$;

CREATE OR REPLACE FUNCTION app.can_see_incident(p_incident uuid) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public, pg_temp AS $$
  SELECT EXISTS (
    SELECT 1 FROM incident i
    WHERE i.id = p_incident
      AND (app.is_manager(i.school_id) OR i.reported_by = app.user_id()
           OR (app.is_staff(i.school_id) AND NOT i.is_confidential)))
$$;

CREATE OR REPLACE FUNCTION app.can_edit_incident(p_incident uuid) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public, pg_temp AS $$
  SELECT EXISTS (
    SELECT 1 FROM incident i
    WHERE i.id = p_incident
      AND (app.is_manager(i.school_id) OR (i.reported_by = app.user_id() AND i.status = 'reported')))
$$;

CREATE OR REPLACE FUNCTION app.is_homeroom_of_enrollment(p_enrollment uuid) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public, pg_temp AS $$
  SELECT EXISTS (SELECT 1 FROM student_enrollment se JOIN class_room c ON c.id = se.class_room_id
                 WHERE se.id = p_enrollment AND c.homeroom_teacher_id = app.user_id())
$$;

CREATE OR REPLACE FUNCTION app.guardian_sees_report_card(p_report_card uuid) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public, pg_temp AS $$
  SELECT EXISTS (SELECT 1 FROM report_card rc JOIN student_enrollment se ON se.id = rc.student_enrollment_id
                 WHERE rc.id = p_report_card AND rc.status = 'published' AND app.guardian_of(se.student_id))
$$;

CREATE OR REPLACE FUNCTION app.guardian_of_enrollment(p_enrollment uuid) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public, pg_temp AS $$
  SELECT EXISTS (SELECT 1 FROM student_enrollment se WHERE se.id = p_enrollment AND app.guardian_of(se.student_id))
$$;

CREATE OR REPLACE FUNCTION app.can_view_student(p_student uuid) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public, pg_temp AS $$
  SELECT EXISTS (SELECT 1 FROM student s WHERE s.id = p_student AND app.can_view_people(s.school_id))
$$;

CREATE OR REPLACE FUNCTION app.manages_student(p_student uuid) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public, pg_temp AS $$
  SELECT EXISTS (SELECT 1 FROM student s WHERE s.id = p_student AND app.is_manager(s.school_id))
$$;

CREATE OR REPLACE FUNCTION app.staff_sees_guardian(p_guardian uuid) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public, pg_temp AS $$
  SELECT EXISTS (
    SELECT 1 FROM student_guardian sg JOIN student s ON s.id = sg.student_id
    WHERE sg.guardian_id = p_guardian AND app.can_view_people(s.school_id))
$$;

-- +goose Down
DROP FUNCTION IF EXISTS app.guardian_of_enrollment(uuid);
DROP FUNCTION IF EXISTS app.guardian_sees_report_card(uuid);
DROP FUNCTION IF EXISTS app.is_homeroom_of_enrollment(uuid);
DROP FUNCTION IF EXISTS app.can_edit_incident(uuid);
DROP FUNCTION IF EXISTS app.can_see_incident(uuid);
DROP FUNCTION IF EXISTS app.can_edit_question(uuid);
DROP FUNCTION IF EXISTS app.guardian_sees_assessment(uuid);
DROP FUNCTION IF EXISTS app.assessment_published(uuid);
DROP FUNCTION IF EXISTS app.teaches_assessment(uuid);
DROP FUNCTION IF EXISTS app.attendance_session_editable(uuid);
DROP FUNCTION IF EXISTS app.can_mark_attendance(uuid, uuid, uuid);
DROP FUNCTION IF EXISTS app.guardian_is_me(uuid);
DROP FUNCTION IF EXISTS app.guardian_of(uuid);
DROP FUNCTION IF EXISTS app.is_homeroom(uuid);
DROP FUNCTION IF EXISTS app.teaches(uuid);
DROP FUNCTION IF EXISTS app.in_tenant_manager(uuid);
DROP FUNCTION IF EXISTS app.can_view_people(uuid);
DROP FUNCTION IF EXISTS app.is_staff(uuid);
DROP FUNCTION IF EXISTS app.is_manager(uuid);
DROP FUNCTION IF EXISTS app.is_member(uuid);
DROP FUNCTION IF EXISTS app.has_role(uuid, text[]);
DROP FUNCTION IF EXISTS app.is_system_context(oid);
DROP FUNCTION IF EXISTS app.user_id();
DROP FUNCTION IF EXISTS set_updated_at();
DROP SCHEMA IF EXISTS app CASCADE;
