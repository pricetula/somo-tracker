-- Goose migration
-- +goose Up
-- 00007: Report card domain + generation + lock triggers

CREATE TABLE IF NOT EXISTS school_academic_setting (
    school_id UUID PRIMARY KEY REFERENCES school(id) ON DELETE CASCADE,
    default_grading_scale_id UUID REFERENCES grading_scale(id) ON DELETE SET NULL,
    absent_counts_as_zero BOOLEAN NOT NULL DEFAULT FALSE,
    not_submitted_counts_as_zero BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_school_academic_setting_updated_at
BEFORE UPDATE ON school_academic_setting FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS report_card (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    school_id UUID NOT NULL,
    academic_year_id UUID NOT NULL,
    student_enrollment_id UUID NOT NULL,
    academic_term_id UUID NOT NULL,
    status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','published')),
    days_recorded INT NOT NULL DEFAULT 0 CHECK (days_recorded >= 0),
    days_present INT NOT NULL DEFAULT 0 CHECK (days_present >= 0),
    days_late INT NOT NULL DEFAULT 0 CHECK (days_late >= 0),
    days_left_early INT NOT NULL DEFAULT 0 CHECK (days_left_early >= 0),
    days_absent INT NOT NULL DEFAULT 0 CHECK (days_absent >= 0),
    days_excused INT NOT NULL DEFAULT 0 CHECK (days_excused >= 0),
    attendance_rate NUMERIC(5,1) GENERATED ALWAYS AS (
        CASE WHEN days_recorded = 0 THEN NULL
             ELSE round(100.0 * (days_present + days_late + days_left_early) / days_recorded, 1) END
    ) STORED,
    class_teacher_remarks TEXT,
    head_remarks TEXT,
    generated_at TIMESTAMPTZ,
    generated_by UUID REFERENCES "user"(id) ON DELETE SET NULL,
    published_at TIMESTAMPTZ,
    published_by UUID REFERENCES "user"(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (student_enrollment_id, academic_term_id),
    FOREIGN KEY (school_id, academic_year_id, student_enrollment_id)
        REFERENCES student_enrollment (school_id, academic_year_id, id) ON DELETE CASCADE,
    FOREIGN KEY (school_id, academic_year_id, academic_term_id)
        REFERENCES academic_term (school_id, academic_year_id, id)
);
CREATE INDEX IF NOT EXISTS idx_report_card_school ON report_card (school_id);
CREATE TRIGGER trg_report_card_updated_at
BEFORE UPDATE ON report_card FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS report_card_subject (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    school_id UUID NOT NULL,
    report_card_id UUID NOT NULL,
    course_offering_id UUID NOT NULL,
    subject_id UUID NOT NULL,
    subject_name TEXT NOT NULL,
    percentage NUMERIC(5,2),
    grading_band_id UUID REFERENCES grading_scale_band(id) ON DELETE SET NULL,
    grade_label TEXT,
    assessments_counted SMALLINT NOT NULL DEFAULT 0,
    teacher_remarks TEXT,
    UNIQUE (report_card_id, course_offering_id),
    FOREIGN KEY (school_id, report_card_id) REFERENCES report_card (school_id, id) ON DELETE CASCADE,
    FOREIGN KEY (school_id, course_offering_id, subject_id) REFERENCES course_offering (school_id, id, subject_id)
);
CREATE INDEX IF NOT EXISTS idx_report_card_subject_card ON report_card_subject (report_card_id);

CREATE OR REPLACE FUNCTION enforce_report_card_lock() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE v_new jsonb;
BEGIN
  IF app.is_system_context(TG_RELID) THEN
    IF TG_OP = 'DELETE' THEN RETURN OLD; END IF; RETURN NEW;
  END IF;

  IF app.is_manager(OLD.school_id) THEN
    IF OLD.status = 'published' THEN
      IF TG_OP = 'UPDATE' THEN v_new := to_jsonb(NEW); END IF;
      INSERT INTO locked_edit_log (school_id, table_name, row_id, action, changed_by, old_data, new_data)
      VALUES (OLD.school_id, 'report_card', OLD.id, TG_OP, app.user_id(), to_jsonb(OLD), v_new);
    END IF;
    IF TG_OP = 'UPDATE' AND NEW.status = 'published' AND OLD.status <> 'published' THEN
      NEW.published_at := NOW();
      NEW.published_by := app.user_id();
    END IF;
  ELSE
    IF TG_OP = 'DELETE' THEN
      RAISE EXCEPTION 'only a head or admin can delete a report card' USING ERRCODE = 'insufficient_privilege';
    END IF;
    IF OLD.status = 'published' OR NEW.status = 'published'
       OR NEW.head_remarks IS DISTINCT FROM OLD.head_remarks
       OR NEW.days_recorded <> OLD.days_recorded OR NEW.days_present <> OLD.days_present
       OR NEW.days_late <> OLD.days_late OR NEW.days_left_early <> OLD.days_left_early
       OR NEW.days_absent <> OLD.days_absent OR NEW.days_excused <> OLD.days_excused THEN
      RAISE EXCEPTION 'only class_teacher_remarks of a draft report card can be edited by a teacher' USING ERRCODE = 'insufficient_privilege';
    END IF;
  END IF;
  IF TG_OP = 'DELETE' THEN RETURN OLD; END IF;
  RETURN NEW;
END;
$$;

CREATE TRIGGER trg_report_card_lock BEFORE UPDATE OR DELETE ON report_card
  FOR EACH ROW EXECUTE FUNCTION enforce_report_card_lock();

CREATE OR REPLACE FUNCTION enforce_report_card_subject_lock() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE v_status text;
BEGIN
  IF app.is_system_context(TG_RELID) THEN
    IF TG_OP = 'DELETE' THEN RETURN OLD; END IF; RETURN NEW;
  END IF;
  IF app.is_manager(OLD.school_id) THEN
    IF TG_OP = 'DELETE' THEN RETURN OLD; END IF; RETURN NEW;
  END IF;
  IF TG_OP = 'DELETE' THEN
    RAISE EXCEPTION 'only a head or admin can delete report card lines' USING ERRCODE = 'insufficient_privilege';
  END IF;
  SELECT status INTO v_status FROM report_card WHERE id = OLD.report_card_id;
  IF v_status = 'published'
     OR NEW.percentage IS DISTINCT FROM OLD.percentage
     OR NEW.grading_band_id IS DISTINCT FROM OLD.grading_band_id
     OR NEW.grade_label IS DISTINCT FROM OLD.grade_label
     OR NEW.assessments_counted <> OLD.assessments_counted
     OR NEW.subject_name <> OLD.subject_name
     OR NEW.course_offering_id <> OLD.course_offering_id THEN
    RAISE EXCEPTION 'only teacher_remarks of a draft report card can be edited by a teacher' USING ERRCODE = 'insufficient_privilege';
  END IF;
  RETURN NEW;
END;
$$;

CREATE TRIGGER trg_report_card_subject_lock BEFORE UPDATE OR DELETE ON report_card_subject
  FOR EACH ROW EXECUTE FUNCTION enforce_report_card_subject_lock();

CREATE OR REPLACE FUNCTION generate_report_card(p_enrollment_id uuid, p_term_id uuid) RETURNS uuid
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public, pg_temp AS $$
DECLARE
  e student_enrollment%ROWTYPE;
  t academic_term%ROWTYPE;
  st school_academic_setting%ROWTYPE;
  att RECORD;
  v_card uuid; v_status text; v_mgr boolean;
  v_abs_zero boolean; v_nosub_zero boolean; v_remarks jsonb;
BEGIN
  SELECT * INTO e FROM student_enrollment WHERE id = p_enrollment_id;
  IF NOT FOUND THEN RAISE EXCEPTION 'enrollment % not found', p_enrollment_id; END IF;

  IF app.user_id() IS NULL THEN
    IF session_user <> current_user AND NOT (SELECT rolsuper FROM pg_roles WHERE rolname = session_user) THEN
      RAISE EXCEPTION 'no user context' USING ERRCODE = 'insufficient_privilege';
    END IF;
    v_mgr := TRUE;
  ELSE
    v_mgr := app.is_manager(e.school_id);
    IF NOT v_mgr AND NOT app.is_homeroom(e.class_room_id) THEN
      RAISE EXCEPTION 'only a head, admin or the class homeroom teacher can generate this report card' USING ERRCODE = 'insufficient_privilege';
    END IF;
  END IF;

  SELECT * INTO t FROM academic_term
   WHERE id = p_term_id AND school_id = e.school_id AND academic_year_id = e.academic_year_id;
  IF NOT FOUND THEN RAISE EXCEPTION 'term does not belong to the enrollment''s academic year'; END IF;

  SELECT * INTO st FROM school_academic_setting WHERE school_id = e.school_id;
  v_abs_zero := COALESCE(st.absent_counts_as_zero, FALSE);
  v_nosub_zero := COALESCE(st.not_submitted_counts_as_zero, TRUE);

  SELECT id, status INTO v_card, v_status FROM report_card
   WHERE student_enrollment_id = p_enrollment_id AND academic_term_id = p_term_id;
  IF v_status = 'published' THEN
    IF NOT v_mgr THEN
      RAISE EXCEPTION 'published report cards can only be regenerated by a head or admin' USING ERRCODE = 'insufficient_privilege';
    END IF;
    INSERT INTO locked_edit_log (school_id, table_name, row_id, action, changed_by, old_data)
    SELECT e.school_id, 'report_card', rc.id, 'REGENERATE', app.user_id(), to_jsonb(rc)
      FROM report_card rc WHERE rc.id = v_card;
  END IF;

  SELECT count(*) AS recorded,
         count(*) FILTER (WHERE r.status = 'present') AS present,
         count(*) FILTER (WHERE r.status = 'late') AS late,
         count(*) FILTER (WHERE r.status = 'left_early') AS left_early,
         count(*) FILTER (WHERE r.status = 'absent') AS absent,
         count(*) FILTER (WHERE r.status = 'excused') AS excused
    INTO att
    FROM attendance_record r
    JOIN attendance_session s ON s.id = r.attendance_session_id
   WHERE r.student_id = e.student_id
     AND s.kind = 'daily' AND s.status = 'submitted'
     AND s.class_room_id = e.class_room_id
     AND s.session_date BETWEEN t.starts_on AND t.ends_on
     AND s.session_date >= e.enrolled_on
     AND (e.left_on IS NULL OR s.session_date <= e.left_on);

  INSERT INTO report_card (school_id, academic_year_id, student_enrollment_id, academic_term_id,
                           days_recorded, days_present, days_late, days_left_early, days_absent, days_excused,
                           generated_at, generated_by)
  VALUES (e.school_id, e.academic_year_id, e.id, t.id,
          att.recorded, att.present, att.late, att.left_early, att.absent, att.excused,
          NOW(), app.user_id())
  ON CONFLICT (student_enrollment_id, academic_term_id) DO UPDATE SET
      days_recorded = EXCLUDED.days_recorded, days_present = EXCLUDED.days_present,
      days_late = EXCLUDED.days_late, days_left_early = EXCLUDED.days_left_early,
      days_absent = EXCLUDED.days_absent, days_excused = EXCLUDED.days_excused,
      generated_at = EXCLUDED.generated_at, generated_by = EXCLUDED.generated_by
  RETURNING id INTO v_card;

  SELECT COALESCE(jsonb_object_agg(course_offering_id::text, teacher_remarks) FILTER (WHERE teacher_remarks IS NOT NULL), '{}'::jsonb)
    INTO v_remarks FROM report_card_subject WHERE report_card_id = v_card;
  DELETE FROM report_card_subject WHERE report_card_id = v_card;

  INSERT INTO report_card_subject (school_id, report_card_id, course_offering_id, subject_id, subject_name,
                                   percentage, grading_band_id, grade_label, assessments_counted, teacher_remarks)
  SELECT e.school_id, v_card, co.id, co.subject_id, sub.name,
         agg.pct, b.id, b.label, COALESCE(agg.n, 0), v_remarks ->> co.id::text
    FROM course_enrollment ce
    JOIN course_offering co ON co.id = ce.course_offering_id
    JOIN subject sub ON sub.id = co.subject_id
    LEFT JOIN LATERAL (
          SELECT round(100 * sum(x.w * x.frac) / NULLIF(sum(x.w), 0), 2) AS pct, count(*)::int AS n
            FROM (SELECT COALESCE(a.weight, 1) AS w,
                         CASE WHEN r.status = 'scored' THEN r.score / a.max_score ELSE 0 END AS frac
                    FROM assessment a
                    JOIN assessment_result r ON r.assessment_id = a.id AND r.student_id = e.student_id
                   WHERE a.course_offering_id = co.id
                     AND a.academic_term_id = t.id
                     AND a.status IN ('submitted','published')
                     AND ((r.status = 'scored' AND r.score IS NOT NULL)
                          OR (r.status = 'absent' AND v_abs_zero)
                          OR (r.status = 'not_submitted' AND v_nosub_zero))) x
    ) agg ON TRUE
    LEFT JOIN LATERAL (
          SELECT gb.id, gb.label FROM grading_scale_band gb
           WHERE gb.grading_scale_id = st.default_grading_scale_id
             AND agg.pct IS NOT NULL
             AND (gb.min_score IS NULL OR agg.pct >= gb.min_score)
             AND (gb.max_score IS NULL OR agg.pct <= gb.max_score)
           ORDER BY gb.position LIMIT 1
    ) b ON TRUE
   WHERE ce.student_enrollment_id = p_enrollment_id
     AND ce.academic_year_id = e.academic_year_id;

  RETURN v_card;
END;
$$;

-- +goose Down
DROP TRIGGER IF EXISTS trg_report_card_subject_lock ON report_card_subject;
DROP TRIGGER IF EXISTS trg_report_card_lock ON report_card;
DROP FUNCTION IF EXISTS enforce_report_card_subject_lock();
DROP FUNCTION IF EXISTS enforce_report_card_lock();
DROP FUNCTION IF EXISTS generate_report_card(uuid, uuid);
DROP TRIGGER IF EXISTS trg_report_card_updated_at ON report_card;
DROP TRIGGER IF EXISTS trg_school_academic_setting_updated_at ON school_academic_setting;
DROP TABLE IF EXISTS report_card_subject;
DROP TABLE IF EXISTS report_card;
DROP TABLE IF EXISTS school_academic_setting;
