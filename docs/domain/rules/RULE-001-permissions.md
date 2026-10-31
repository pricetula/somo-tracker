---
id: RULE-001
title: Permission Matrix
status: draft
depends_on: []
touches: [backend, frontend]
---

# Rule

Access to resources is governed by role per school membership with tenant-level overrides for Tenant Owner and Platform Operator.

Rationale:
The permission matrix defines the minimal necessary access per role to satisfy FR-FND-002 RBAC, tenant isolation FR-FND-001, and teacher fee data separation.

Implications:
Implement server-side role checks on every request per NFR-SEC-002. No teacher access to fee data or arrears status. Tenant Owner has automatic view access to all student data in every school of the tenant with audit logging. Platform Operator has support access to audit log with approval only.

## Permission Matrix

| Resource | Platform Op | Tenant Owner | Head | Admin | Teacher | Finance | Guardian |
| --- | --- | --- | --- | --- | --- | --- | --- |
| School settings | Full | Edit | View | Edit | None | None | None |
| Student records | None | View (all schools) | View | Edit | View (own classes) | View basic (billing fields only) | View (own children) |
| Attendance | None | View (all schools) | View | Edit | Edit (own classes) | None | View (own children) |
| Assessment | None | View (all schools) | View | View | Edit (own subjects) | None | View (own children) |
| Health and vaccination | None | View (all schools) | View | View | Edit (assigned role only) | None | View (own children) |
| Behavior | None | View (all schools) | View | View | Create and edit own | None | View (own children) |
| Fee structures | None | View (all schools) | View | Edit | None | Edit | None |
| Audit log | Support access with approval | View (all schools) | View | View | None | None | None |
| Payments and receipts | None | View (all schools) | View | View | None | Create and edit | View (own children) |
| Waivers, discounts, write-offs | None | View (all schools) | Approve | View | None | Propose | None |
| Fee reports and arrears lists | None | View (all schools) | View | View | None | View | None |
| Schools and ownership | Suspend tenant | Full | None | None | None | None | None |

### Notes
- Teachers have no access to fee data, including arrears status, so no child is singled out in class.
- Every waiver, discount, and write-off is audit-logged with the proposer and approver.
- Each cell must be confirmed during spec review per PRD Section 8.
