# Specification registry

| ID | Title | Status | FR/NFR IDs | API contract change? | Migrations? | Updated |
|----|-------|--------|------------|----------------------|-------------|---------|
| SOM-AUTHZ-001 | Authz Permission Matrix | implemented | BE-020, BE-021, BE-016, BE-017, BE-018 | no | no | 2026-11-11 |
| SOM-STUDENTS-001 | Students Domain - CRUD and Guardian Links | draft | BE-006, BE-007, BE-009, BE-010, BE-012, BE-013, BE-014, BE-015, BE-016, BE-020, BE-021, BE-025, BE-036, FE-007, FE-008, FE-009, FE-011, FE-014, FE-019, FE-020, FE-026, FE-027 | yes | no | 2026-11-11 |
| SOM-ROLE-OWNER | Role — Tenant Owner (OWNER) | draft | BE-020, BE-021, BE-016, BE-017 | no | no | 2026-11-11 |
| SOM-ROLE-ADMIN | Role — School Admin (ADMIN) | draft | BE-020, BE-021, BE-016, BE-017 | no | no | 2026-11-11 |
| SOM-ROLE-HEADTEACHER | Role — Headteacher (HEADTEACHER) | draft | BE-020, BE-021, BE-016, BE-017 | no | no | 2026-11-11 |
| SOM-ROLE-TEACHER | Role — Teacher (TEACHER) | draft | BE-020, BE-021, BE-016, BE-017 | no | no | 2026-11-11 |
| SOM-ROLE-GUARDIAN | Role — Guardian (GUARDIAN) | draft | BE-020, BE-021, BE-016, BE-017 | no | no | 2026-11-11 |
| SOM-ROLE-FINANCE | Role — Finance (FINANCE) | draft | BE-020, BE-021, BE-016, BE-017 | no | no | 2026-11-11 |
| SOM-ROLE-PLATFORMOPERATOR | Role — Platform Operator (PLATFORMOPERATOR) | draft | BE-020, BE-021, BE-016, BE-017 | no | no | 2026-11-11 |

## Rules

- A specification may be implemented only in `approved` status.
- Update this registry and the specification frontmatter together (one fact in one place).
- When implementation completes, set status `implemented`; after human review, `verified by human`.
- Link to specifications from PRs and commits; reference FR/NFR IDs (BE-035, FE-026).
