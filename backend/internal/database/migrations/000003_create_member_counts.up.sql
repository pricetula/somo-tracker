-- Create member_counts table
CREATE TABLE IF NOT EXISTS member_counts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    students INTEGER NOT NULL DEFAULT 0,
    admins INTEGER NOT NULL DEFAULT 0,
    nurses INTEGER NOT NULL DEFAULT 0,
    teachers INTEGER NOT NULL DEFAULT 0,
    parents INTEGER NOT NULL DEFAULT 0,
    finance INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Ensure exactly one aggregate row exists (fresh DB starts at zero).
DELETE FROM member_counts;
INSERT INTO member_counts (id, students, admins, nurses, teachers, parents, finance, created_at, updated_at)
VALUES (gen_random_uuid(), 0, 0, 0, 0, 0, 0, now(), now());

-- Function to update student count based on changes in cbc_students
CREATE OR REPLACE FUNCTION trg_update_student_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF NEW.is_active THEN
            UPDATE member_counts SET students = students + 1, updated_at = now();
        END IF;
        RETURN NEW;
    ELSIF TG_OP = 'UPDATE' THEN
        IF OLD.is_active IS DISTINCT FROM NEW.is_active THEN
            IF NEW.is_active THEN
                UPDATE member_counts SET students = students + 1, updated_at = now();
            ELSE
                UPDATE member_counts SET students = students - 1, updated_at = now();
            END IF;
        END IF;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        IF OLD.is_active THEN
            UPDATE member_counts SET students = students - 1, updated_at = now();
        END IF;
        RETURN OLD;
    END IF;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_student_count
AFTER INSERT OR UPDATE OR DELETE ON cbc_students
FOR EACH ROW EXECUTE FUNCTION trg_update_student_count();

-- Function to update membership counts based on changes in memberships.
-- Handles all four mutation shapes:
--   INSERT            -> increment NEW.role
--   UPDATE (role only) -> decrement OLD.role, increment NEW.role (is_active unchanged)
--   UPDATE (active)    -> decrement OLD.role / increment NEW.role by is_active delta
--   DELETE             -> decrement OLD.role
CREATE OR REPLACE FUNCTION trg_update_membership_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF NEW.is_active THEN
            CASE NEW.role
                WHEN 'SCHOOL_ADMIN' THEN UPDATE member_counts SET admins = admins + 1;
                WHEN 'TEACHER' THEN UPDATE member_counts SET teachers = teachers + 1;
                WHEN 'NURSE' THEN UPDATE member_counts SET nurses = nurses + 1;
                WHEN 'PARENT' THEN UPDATE member_counts SET parents = parents + 1;
                WHEN 'FINANCE' THEN UPDATE member_counts SET finance = finance + 1;
                WHEN 'SYSTEM_ADMIN' THEN NULL; -- not counted
            END CASE;
            UPDATE member_counts SET updated_at = now();
        END IF;
        RETURN NEW;
    ELSIF TG_OP = 'UPDATE' THEN
        -- Role change while remaining active: decrement old, increment new.
        IF OLD.is_active AND NEW.is_active AND OLD.role IS DISTINCT FROM NEW.role THEN
            CASE OLD.role
                WHEN 'SCHOOL_ADMIN' THEN UPDATE member_counts SET admins = admins - 1;
                WHEN 'TEACHER' THEN UPDATE member_counts SET teachers = teachers - 1;
                WHEN 'NURSE' THEN UPDATE member_counts SET nurses = nurses - 1;
                WHEN 'PARENT' THEN UPDATE member_counts SET parents = parents - 1;
                WHEN 'FINANCE' THEN UPDATE member_counts SET finance = finance - 1;
                WHEN 'SYSTEM_ADMIN' THEN NULL;
            END CASE;
            CASE NEW.role
                WHEN 'SCHOOL_ADMIN' THEN UPDATE member_counts SET admins = admins + 1;
                WHEN 'TEACHER' THEN UPDATE member_counts SET teachers = teachers + 1;
                WHEN 'NURSE' THEN UPDATE member_counts SET nurses = nurses + 1;
                WHEN 'PARENT' THEN UPDATE member_counts SET parents = parents + 1;
                WHEN 'FINANCE' THEN UPDATE member_counts SET finance = finance + 1;
                WHEN 'SYSTEM_ADMIN' THEN NULL;
            END CASE;
            UPDATE member_counts SET updated_at = now();
        END IF;

        -- Activation/deactivation change (role may also change; each branch
        -- only fires for the side that transitioned, so no double counting).
        IF OLD.is_active IS DISTINCT FROM NEW.is_active THEN
            IF OLD.is_active THEN
                CASE OLD.role
                    WHEN 'SCHOOL_ADMIN' THEN UPDATE member_counts SET admins = admins - 1;
                    WHEN 'TEACHER' THEN UPDATE member_counts SET teachers = teachers - 1;
                    WHEN 'NURSE' THEN UPDATE member_counts SET nurses = nurses - 1;
                    WHEN 'PARENT' THEN UPDATE member_counts SET parents = parents - 1;
                    WHEN 'FINANCE' THEN UPDATE member_counts SET finance = finance - 1;
                    WHEN 'SYSTEM_ADMIN' THEN NULL;
                END CASE;
            END IF;
            IF NEW.is_active THEN
                CASE NEW.role
                    WHEN 'SCHOOL_ADMIN' THEN UPDATE member_counts SET admins = admins + 1;
                    WHEN 'TEACHER' THEN UPDATE member_counts SET teachers = teachers + 1;
                    WHEN 'NURSE' THEN UPDATE member_counts SET nurses = nurses + 1;
                    WHEN 'PARENT' THEN UPDATE member_counts SET parents = parents + 1;
                    WHEN 'FINANCE' THEN UPDATE member_counts SET finance = finance + 1;
                    WHEN 'SYSTEM_ADMIN' THEN NULL;
                END CASE;
            END IF;
            UPDATE member_counts SET updated_at = now();
        END IF;

        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        IF OLD.is_active THEN
            CASE OLD.role
                WHEN 'SCHOOL_ADMIN' THEN UPDATE member_counts SET admins = admins - 1;
                WHEN 'TEACHER' THEN UPDATE member_counts SET teachers = teachers - 1;
                WHEN 'NURSE' THEN UPDATE member_counts SET nurses = nurses - 1;
                WHEN 'PARENT' THEN UPDATE member_counts SET parents = parents - 1;
                WHEN 'FINANCE' THEN UPDATE member_counts SET finance = finance - 1;
                WHEN 'SYSTEM_ADMIN' THEN NULL;
            END CASE;
            UPDATE member_counts SET updated_at = now();
        END IF;
        RETURN OLD;
    END IF;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_membership_count
AFTER INSERT OR UPDATE OR DELETE ON memberships
FOR EACH ROW EXECUTE FUNCTION trg_update_membership_count();
