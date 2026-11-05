# Migration Roadmap – Remaining Domain Tables

Status: planning. Do not implement until spec is approved.

## Principles
* All tables use `uuidv7()` PK, `created_at/updated_at/deleted_at`, soft delete partial index.
* Every multi-tenant table gets RLS + tenant isolation policy.

## 1. attendance
Daily attendance per enrollment.

Columns:
* id uuidv7 PK
* enrollment_id uuid FK enrollments.id RESTRICT
* school_day date NOT NULL
* status enum(PRESENT,LATE,ABSENT,EXCUSED) NOT NULL
* marked_by uuid FK users.id
* marked_at timestamptz NOT NULL DEFAULT NOW()
* edit_reason text
* created_at/updated_at/deleted_at
Unique enrollment_id, school_day.
RLS via schools.

## 2. fee_structures
Fee categories per school/class/term.

Columns:
* id uuidv7 PK
* school_id uuid FK schools.id RESTRICT
* class_level_id uuid FK class_levels.id RESTRICT
* term_id uuid FK terms.id RESTRICT
* category varchar(64) NOT NULL
* amount_minor bigint NOT NULL
* description text
* created_at/updated_at/deleted_at
Unique school_id, class_level_id, term_id, category.

## 3. payments
Student payments and receipts.

Columns:
* id uuidv7 PK
* student_id uuid FK students.id RESTRICT
* tenant_id uuid FK tenants.id
* amount_minor bigint NOT NULL
* method varchar(64)
* reference varchar(128)
* paid_at date NOT NULL
* receipt_url text
* created_at/updated_at/deleted_at

## 4. audit_logs
Immutable audit trail.

Columns:
* id uuidv7 PK
* tenant_id uuid
* school_id uuid
* actor_user_id uuid
* action varchar(32) NOT NULL
* table_name varchar(64) NOT NULL
* record_id uuid NOT NULL
* changed_fields jsonb
* occurred_at timestamptz NOT NULL DEFAULT NOW()
No soft delete. App role INSERT/SELECT only.

## 5. invites
Staff/guardian invites.

Columns:
* id uuidv7 PK
* tenant_id uuid FK tenants.id
* school_id uuid FK schools.id RESTRICT
* email_normalized text NOT NULL
* role user_role NOT NULL
* status enum(PENDING,ACCEPTED,EXPIRED) NOT NULL
* token_hash bytea NOT NULL
* expires_at timestamptz NOT NULL
* created_at/updated_at
Unique email_normalized, school_id, status=PENDING.

## 6. needs_attention
System generated tasks.

Columns:
* id uuidv7 PK
* tenant_id uuid FK tenants.id
* school_id uuid FK schools.id RESTRICT
* type varchar(64) NOT NULL
* entity_type varchar(64) NOT NULL
* entity_id uuid NOT NULL
* message text
* status enum(OPEN,ACKNOWLEDGED,RESOLVED) NOT NULL DEFAULT 'OPEN'
* created_at/updated_at/deleted_at

## Optional health/behavior tables
To be specced separately:
* health_records, vaccinations, behavior_records with audit on view per FR-WEL-001.

Each table requires approved spec, migration file, integration test, RLS policy, and `make compliance` before merge.
