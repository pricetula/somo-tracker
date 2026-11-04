# Migration Roadmap – Remaining Domain Tables

Status: planning. Do not implement until spec is approved.

## Principles
* All tables use `uuidv7()` PK, `created_at/updated_at/deleted_at`, soft delete partial index.
* Every multi-tenant table gets RLS + tenant isolation policy.
* Performance levels are per-school, not enum, to stay education-system-agnostic.

## 1. performance_levels
Education-system-agnostic rating scale per school, used to map raw results to a qualitative level.

Columns:
* id uuidv7 PK
* school_id uuid FK schools.id RESTRICT
* code varchar(16) NOT NULL
* label varchar(128) NOT NULL
* description text
* sequence_index int NOT NULL
* created_at/updated_at/deleted_at timestamptz
Unique school_id, code. Unique school_id, sequence_index.
Note: Some schools may use only raw scores. performance_level_id on assessments is therefore nullable and mapping is school-defined.

## 2. attendance
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

## 3. assessments
Learner performance per subject/topic/subtopic/term with both raw results and optional mapped performance level.

Columns:
* id uuidv7 PK
* student_id uuid FK students.id RESTRICT
* classroom_id uuid FK classrooms.id RESTRICT
* subject_id uuid FK subjects.id RESTRICT
* topic_id uuid FK topics.id RESTRICT
* subtopic_id uuid FK subtopics.id RESTRICT
* term_id uuid FK terms.id RESTRICT
* assessment_date date NOT NULL
* raw_score numeric
* max_score numeric
* performance_level_id uuid FK performance_levels.id RESTRICT  -- nullable, school-dependent
* comment text
* created_at/updated_at/deleted_at
Unique student_id, subject_id, topic_id, subtopic_id, term_id, assessment_date.
RLS via schools.

Start order consideration: assessments depends on performance_levels for mapping, but assessments can be built first with nullable performance_level_id and raw_score only, then performance_levels added later.

## 4. report_cards
Generated term report per student.

Columns:
* id uuidv7 PK
* student_id uuid FK students.id RESTRICT
* term_id uuid FK terms.id RESTRICT
* classroom_id uuid FK classrooms.id RESTRICT
* pdf_url text
* generated_at timestamptz
* created_at/updated_at/deleted_at
Unique student_id, term_id.

## 5. fee_structures
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

## 6. payments
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

## 7. audit_logs
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

## 8. invites
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

## 9. needs_attention
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
