# SomoTracker Backend Table Documentation

Documenting all database tables, relationships, and schema conventions from `/backend/internal/database/migrations/*.up.sql`. Follows backend AGENTS.md compliance.

## Table Directory Structure

```
├── tenants      # Core identity anchoring
├── users        # Authentication & roles
├── sessions     # API session management
├── cbc_schools  # School infrastructure
├── academic_years # Calendar management
├── academic_terms # Term-specific data
├── cbc_students # Learner records
├── cbc_classes # Class groupings
├── cbc_student_parents # Family connections

...[additional tables from migration files]...
```

## Table Relationships Overview

### Tenant-Scholarship Chain (Core Compliance)
```
Tenants \u2192 cbc_schools \u2192 academic_years \u2192 academic_terms
          \u2192 cbc_students \u2192 cbc_classes
          \u2026              \u2192 cbc_student_enrollments
              \u2026              \u2192 student_assessments
```

### Multi-Tenant Isolation
- All tables use composite (tenant_id, id) foreign keys
- Row-Level Security (RLS) enforced on every school/student/entity

## Detailed Table Definitions

### Tenants
- **id**: UUID (primary key)
- **name/slug/stytch_org_id**: Identity anchors
- **lgislation/compliance**: Stytch integration required

### Users
- **tenant_id**: Required for tenant scoping
- **roles/user_role**: Must match enum values
- **tsc_number**: Teacher-specific field

### Assessment_Sessions (Legacy):
- **status**: Must be one of 'DRAFT', 'SUBMITTED', 'PUBLISHED'
- **trigger_summary**: Auto-calls summary refresh on 'PUBLISHED'

### Materialized Summaries
- **student_term_subject_summaries**:
  - Aggregates all Published assessment sessions per student/term
  - Blends quantitative scores and rubric outcomes
  - Maintains via fn_refresh_term_subject_summary_for_session

### Compliance Triggers
- All foreign key references use composite tenant scoping
- Insert/update triggers for atomic updates
- Status transitions validated through business rules

## Schema Conventions

1. **Tenant Scoping**: Every table has explicit tenant_id
2. **Immutable Relationships**: Foreign keys enforce historical records
3. **Materialized Views**: Kept transactionally consistent via triggers
4. **Audit Columns**: created_at/updated_at on all major entities

## Database Health Checks
- Run `SELECT count(*) FROM cbc_students WHERE tenant_id IS NULL` to verify isolation
- Verify all enumeration types match AGENTS.md definitions
- Test RLS policies with `pg_row_security_check` queries