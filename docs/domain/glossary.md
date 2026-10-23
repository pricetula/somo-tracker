---
id: DOMAIN-GLOSSARY
title: Domain Glossary
status: draft
depends_on: []
touches: []
---

# Glossary

Entry format:
- **Term**: definition
- Context: where used
- References: links to model / rules

- **Tenant**: an organization account that owns one or more schools; the boundary for data isolation and guardian identity
  - Context: multi-tenancy, RBAC
  - References: FR-FND-001, ADR-0001 Tenancy Model

- **School**: a single institution within a tenant, with its own classes, staff, students, and fees
  - Context: school structure, onboarding
  - References: FR-ONB-001

- **Membership**: the link between a user and a school, carrying exactly one role. A user can hold memberships in several schools of the same tenant, each with its own role
  - Context: auth, roles
  - References: FR-FND-006

- **Tenant Owner**: a tenant-level role held by one or more users, with authority over the tenant's schools and automatic view access to all student data in them
  - Context: governance, school creation
  - References: FR-FND-007

- **Academic Year / Term**: a school-defined year containing three terms
  - Context: enrollment, assessment, fees
  - References: FR-ONB-002

- **Class Level**: PP1, PP2, Grade 1 to 9
  - Context: school structure, CBC
  - References: FR-ONB-002

- **Stream**: a subdivision of a class level e.g., Grade 4 East
  - Context: class organization, timetable
  - References: FR-ONB-002

- **Enrollment**: the link between one student and one class/stream for one term. A student may have at most one active enrollment per term
  - Context: student lifecycle
  - References: FR-ONB-003

- **Needs Attention item**: a system-generated task requiring staff action
  - Context: dashboard
  - References: FR-DASH-002

- **Absenteeism hotspot**: a class or stream whose 5-day attendance rate is more than 10 percentage points below its trailing 4-week average. Threshold configurable per tenant
  - Context: attendance analytics
  - References: FR-DASH-004

- **Performance level**: CBC rating EE, ME, AE, BE assigned to a learner per learning area, strand, or sub-strand
  - Context: assessment
  - References: FR-ASM-002

- **UPI/NEMIS number**: optional unique pupil identifier
  - Context: identifiers
  - References: FR-ONB-006

