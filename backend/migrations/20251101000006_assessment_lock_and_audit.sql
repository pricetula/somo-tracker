-- Goose migration
-- +goose Up
-- 00006: Assessment workflow + lock audit

ALTER TABLE assessment DROP CONSTRAINT IF EXISTS assessment_status_check;
ALTER TABLE assessment ADD CONSTRAINT assessment_status_check
    CHECK (status IN ('draft','scheduled','marking','submitted','published'));

CREATE TABLE IF NOT EXISTS locked_edit_log (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    school_id UUID NOT NULL REFERENCES school(id) ON DELETE CASCADE,
    table_name TEXT NOT NULL,
    row_id UUID,
    action TEXT NOT NULL,
    changed_by UUID REFERENCES "user"(id) ON DELETE SET NULL,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    old_data JSONB,
    new_data JSONB
);
CREATE INDEX IF NOT EXISTS idx_locked_edit_log_school_time ON locked_edit_log (school_id, changed_at DESC);

CREATE OR REPLACE FUNCTION enforce_assessment_lock() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE v_new jsonb;
BEGIN
  IF app.is_system_context(TG_RELID) THEN
    IF TG_OP = 'DELETE' THEN RETURN OLD; END IF; RETURN NEW;
  END IF;

  IF app.is_manager(OLD.school_id) THEN
    IF OLD.status IN ('submitted','published') THEN
      IF TG_OP = 'UPDATE' THEN v_new := to_jsonb(NEW); END IF;
      INSERT INTO locked_edit_log (school_id, table_name, row_id, action, changed_by, old_data, new_data)
      VALUES (OLD.school_id, 'assessment', OLD.id, TG_OP, app.user_id(), to_jsonb(OLD), v_new);
    END IF;
  ELSE
    IF OLD.status IN ('submitted','published') THEN
      RAISE EXCEPTION 'assessment is % and can only be edited by a head or admin', OLD.status USING ERRCODE = 'insufficient_privilege';
    END IF;
    IF TG_OP = 'UPDATE' AND NEW.status = 'published' THEN
      RAISE EXCEPTION 'only a head or admin can publish an assessment' USING ERRCODE = 'insufficient_privilege';
    END IF;
  END IF;
  IF TG_OP = 'DELETE' THEN RETURN OLD; END IF;
  RETURN NEW;
END;
$$;

CREATE TRIGGER trg_assessment_lock BEFORE UPDATE OR DELETE ON assessment
  FOR EACH ROW EXECUTE FUNCTION enforce_assessment_lock();

CREATE OR REPLACE FUNCTION enforce_assessment_child_lock() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
  v_assessment uuid; v_status text; v_school uuid; v_old jsonb; v_new jsonb;
BEGIN
  IF app.is_system_context(TG_RELID) THEN
    IF TG_OP = 'DELETE' THEN RETURN OLD; END IF; RETURN NEW;
  END IF;

  IF TG_OP = 'DELETE' THEN v_assessment := OLD.assessment_id; ELSE v_assessment := NEW.assessment_id; END IF;
  SELECT status, school_id INTO v_status, v_school FROM assessment WHERE id = v_assessment;

  IF v_status IN ('submitted','published') THEN
    IF NOT app.is_manager(v_school) THEN
      RAISE EXCEPTION 'assessment is % and can only be edited by a head or admin', v_status USING ERRCODE = 'insufficient_privilege';
    END IF;
    IF TG_OP <> 'INSERT' THEN v_old := to_jsonb(OLD); END IF;
    IF TG_OP <> 'DELETE' THEN v_new := to_jsonb(NEW); END IF;
    INSERT INTO locked_edit_log (school_id, table_name, row_id, action, changed_by, old_data, new_data)
    VALUES (v_school, TG_TABLE_NAME, COALESCE((v_old->>'id')::uuid, (v_new->>'id')::uuid),
            TG_OP, app.user_id(), v_old, v_new);
  END IF;
  IF TG_OP = 'DELETE' THEN RETURN OLD; END IF;
  RETURN NEW;
END;
$$;

CREATE TRIGGER trg_assessment_result_lock BEFORE INSERT OR UPDATE OR DELETE ON assessment_result
  FOR EACH ROW EXECUTE FUNCTION enforce_assessment_child_lock();
CREATE TRIGGER trg_assessment_outcome_lock BEFORE INSERT OR UPDATE OR DELETE ON assessment_outcome
  FOR EACH ROW EXECUTE FUNCTION enforce_assessment_child_lock();
CREATE TRIGGER trg_assessment_outcome_result_lock BEFORE INSERT OR UPDATE OR DELETE ON assessment_outcome_result
  FOR EACH ROW EXECUTE FUNCTION enforce_assessment_child_lock();
CREATE TRIGGER trg_quiz_lock BEFORE INSERT OR UPDATE OR DELETE ON quiz
  FOR EACH ROW EXECUTE FUNCTION enforce_assessment_child_lock();

-- +goose Down
DROP TRIGGER IF EXISTS trg_quiz_lock ON quiz;
DROP TRIGGER IF EXISTS trg_assessment_outcome_result_lock ON assessment_outcome_result;
DROP TRIGGER IF EXISTS trg_assessment_outcome_lock ON assessment_outcome;
DROP TRIGGER IF EXISTS trg_assessment_result_lock ON assessment_result;
DROP TRIGGER IF EXISTS trg_assessment_lock ON assessment;
DROP FUNCTION IF EXISTS enforce_assessment_child_lock();
DROP FUNCTION IF EXISTS enforce_assessment_lock();
DROP TABLE IF EXISTS locked_edit_log;
ALTER TABLE assessment DROP CONSTRAINT assessment_status_check;
ALTER TABLE assessment ADD CONSTRAINT assessment_status_check
    CHECK (status IN ('draft','scheduled','marking','published'));
