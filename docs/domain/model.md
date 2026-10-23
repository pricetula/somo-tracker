---
id: DOMAIN-MODEL
title: Domain Model
status: draft
depends_on: []
touches: []
---

# Domain Model

## Core entities

- **Tenant** -> owns many **School**
- **School** -> has many **User** via **Membership**; owns AcademicYears, ClassLevels, Streams
- **User** -> has one identity per tenant; can have multiple Memberships each with a role scoped to a School
- **Membership** -> User + School + Role. Role is one of Platform Operator, Tenant Owner, Headteacher, School Administrator/Registrar, Teacher, Finance Officer, Guardian
- **AcademicYear** -> contains three **Term**
- **ClassLevel** -> PP1, PP2, Grade 1-9
- **Stream** -> subdivision of ClassLevel within School
- **Student** -> has lifecycle states applicant, active, transferred out, withdrawn, graduated; linked to Guardian
- **Enrollment** -> Student + ClassLevel + Stream + Term. Invariant: one active enrollment per student per term
- **Attendance** -> daily mark per enrollment: Present, Late, Absent, Excused. Edits after day close require reason and audit
- **LearningArea** -> per ClassLevel, contains **Strand**, contains **SubStrand**
- **Assessment** -> learner + LearningArea/Strand/SubStrand + Term + Performance level EE/ME/AE/BE + optional numeric score + comment
- **ReportCard** -> per learner per term, generated PDF from Assessment + attendance summary, standard table format
- **FeeStructure** -> per School per ClassLevel per Term with categories tuition, boarding, transport, lunch, other
- **Payment** -> per student, method, reference, date, amount, receipt issued
- **AuditLog** -> create/update/delete/view events on health, behavior, fee records with actor, timestamp, record ID
- **Invite** -> email invite to staff/guardian with status pending/accepted/expired, expires after 7 days
- **NeedsAttention** -> system-generated tasks e.g., class attendance not marked by 10:00, timetable slot without teacher, invite pending >3 days

## Relationships & invariants

- Tenant data isolation: no query may return another tenant's records. FR-FND-001
- Membership role scoping: user permissions are per School membership. FR-FND-006
- Tenant Owner minimum one per tenant; cannot remove last owner. FR-FND-007
- Enrollment uniqueness: one active enrollment per student per term enforced DB. FR-ONB-003
- Student lifecycle transitions recorded with date and reason. FR-ONB-004
- Attendance edit after close requires reason and audit. FR-ATT-002
- Health and vaccination data treated as sensitive personal data with audit on every view. FR-WEL-001

See glossary for domain terms: docs/domain/glossary.md
