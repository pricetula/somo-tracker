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
| SOM-ACADEMICS-001 | Academics Domain - Subjects, Topics, Sub-topics | draft | BE-006, BE-007, BE-009, BE-010, BE-012, BE-013, BE-014, BE-015, BE-016, BE-020, BE-021, BE-025, BE-036, FE-007, FE-008, FE-009, FE-011, FE-014, FE-019, FE-020, FE-024, FE-026, FE-027 | yes | no | 2026-11-11 |
| SOM-ENROLLMENTS-001 | Enrollments Domain - Academic Years, Terms, Enrollments | draft | BE-006, BE-007, BE-009, BE-010, BE-012, BE-013, BE-014, BE-015, BE-016, BE-020, BE-021, BE-025, BE-036, FE-007, FE-008, FE-009, FE-011, FE-014, FE-019, FE-020, FE-023, FE-026, FE-027 | yes | no | 2026-11-11 |
| SOM-FEES-001 | Fees Domain - Fee Structures, Invoices, Payments, Waivers | draft | BE-006, BE-007, BE-009, BE-010, BE-012, BE-013, BE-014, BE-015, BE-016, BE-020, BE-021, BE-025, BE-029, BE-036, FE-007, FE-008, FE-009, FE-011, FE-014, FE-019, FE-020, FE-024, FE-026, FE-027 | yes | yes | 2026-11-11 |
| SOM-HEALTH-001 | Health Domain - Health Records | draft | BE-006, BE-007, BE-009, BE-010, BE-012, BE-013, BE-014, BE-015, BE-016, BE-020, BE-021, BE-025, BE-029, BE-036, FE-007, FE-008, FE-009, FE-011, FE-012, FE-014, FE-019, FE-020, FE-023, FE-026, FE-027 | yes | yes | 2026-11-11 |
| SOM-BEHAVIOR-001 | Behavior Domain - Behavior Records | draft | BE-006, BE-007, BE-009, BE-010, BE-012, BE-013, BE-014, BE-015, BE-016, BE-020, BE-021, BE-025, BE-029, BE-036, FE-007, FE-008, FE-009, FE-011, FE-012, FE-014, FE-019, FE-020, FE-023, FE-026, FE-027 | yes | yes | 2026-11-11 |
| SOM-TIMETABLE-001 | Timetable Management | draft | BE-006, BE-007, BE-009, BE-010, BE-012, BE-013, BE-014, BE-015, BE-016, BE-020, BE-021, BE-025, BE-029, BE-036, FE-007, FE-008, FE-009, FE-011, FE-014, FE-019, FE-020, FE-026, FE-027 | yes | yes | 2026-11-11 |
| SOM-ATTENDANCE-001 | Attendance Management | draft | BE-006, BE-007, BE-009, BE-010, BE-012, BE-013, BE-014, BE-015, BE-016, BE-020, BE-021, BE-025, BE-029, BE-036, FE-007, FE-008, FE-009, FE-011, FE-014, FE-019, FE-020, FE-023, FE-026, FE-027 | yes | yes | 2026-11-11 |

## Rules

- A specification may be implemented only in `approved` status.
- Update this registry and the specification frontmatter together (one fact in one place).
- When implementation completes, set status `implemented`; after human review, `verified by human`.
- Link to specifications from PRs and commits; reference FR/NFR IDs (BE-035, FE-026).
