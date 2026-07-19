# Somo Tracker — Feature Inventory for Landing Page

> **Purpose:** Single source of truth for all product features, to be used as reference when crafting the marketing website (SvelteKit + shadcn-svelte under `public/`).

---

## 1. Product Identity

| Attribute      | Value                                     |
|----------------|-------------------------------------------|
| **Product**    | Somo Tracker                              |
| **Tagline**    | *(TBD — to be crafted from feature set)*  |
| **Type**       | Multi-tenant school management platform   |
| **Frontend**   | Next.js (educational dashboard)           |
| **Marketing**  | SvelteKit + shadcn-svelte (`public/`)     |
| **Backend**    | Go (Fiber) REST API                       |
| **Auth**       | Stytch B2B (magic link / SSO / MFA)      |
| **Audience**   | Schools, teachers, administrators, parents, nurses |

---

## 2. Core App Features

### 2.1 Authentication & Multi-Tenancy

- **Magic-link login** — passwordless email-based authentication via Stytch B2B.
- **Multi-tenant (school) architecture** — each school is a self-contained tenant with full data isolation.
- **Role-based access control** — distinct dashboards and permissions per role:
  - **System Admin** — super-admin oversight across schools.
  - **School Admin** — manages school-wide settings, staff, and curriculum.
  - **Teacher** — classroom-facing tools (assessments, behavior).
  - **Parent/Guardian** — read-only visibility into student performance and health.
  - **Nurse** — health incident logging and student health profile management.
  - **Finance Staff** — billing, invoicing, and payment tracking.
- **Invitation-based onboarding** — bulk staff invitations with role assignment.
- **Registration flow** — new schools self-register via discovery + magic link.

*Relevant files:* `backend/internal/auth/`, `frontend/src/features/auth/`, `backend/internal/invitations/`, `frontend/src/features/invitations/`

### 2.2 Student Management

- **Student profiles** — comprehensive record with admission number, personal details, and enrollment history.
- **Enrollment / disenrollment** — per-term enrollment with grade-level and class assignment.
- **Batch student import** — CSV/Excel bulk import with progress tracking and row-level failure reporting.
- **Parent linking** — associate students with parent/guardian accounts.
- **Enrollment timeline** — visual history of a student's academic journey across terms.

*Relevant files:* `backend/internal/students/`, `frontend/src/features/students/`, `backend/internal/imports/`, `backend/internal/parents/`

### 2.3 School & Academic Structure

- **Multi-school management** — system admins can create and switch between schools.
- **Classes** — create and manage class groups with teacher assignments.
- **Streams** — sub-divide classes into streams (e.g., 4A, 4B).
- **Education levels** — configurable education system levels (e.g., Pre-Primary, Lower Primary, Upper Primary, Junior Secondary).
- **Grade levels** — grade progression within each education level.
- **Academic years & terms** — full academic calendar management with term dates.
- **Class teachers** — assign homeroom teachers to classes.

*Relevant files:* `backend/internal/cbcschools/`, `backend/internal/cbcclasses/`, `backend/internal/cbcstreams/`, `backend/internal/academicyears/`, `backend/internal/classteachers/`, `frontend/src/features/school/`, `frontend/src/features/classes/`, `frontend/src/features/streams/`, `frontend/src/features/academic-years/`, `frontend/src/features/classteachers/`

### 2.4 Curriculum & Assessment

#### Curriculum Builder
- **Learning areas** (subjects) — create and manage subjects by education and grade level.
- **Strands** — curriculum strands within each learning area.
- **Sub-strands** — fine-grained curriculum topics.
- **Performance indicators** — measurable outcomes at the lowest curriculum level.
- **Curriculum tree** — hierarchical view of the full curriculum structure.

#### Grading & Assessments
- **Grading scale profiles** — define custom grading scales with performance-level ranges (EE, ME, AE, BE — or custom).
- **Scale ranges** — set score boundaries mapped to performance levels.
- **Assessment sessions** — lifecycle-managed sessions (DRAFT → PENDING_APPROVAL → PUBLISHED).
  - Quantitative scoring: bulk-enter student scores with automatic performance-level snapshots.
  - Rubric grading: indicator-level outcome grades with CBC performance levels.
- **Approval workflow** — assessments require admin approval before publication.
- **Weight configurations** — configurable weightings for different assessment components.
- **Parent assessment view** — parents see published results for their children.
- **Term-grade aggregation** — roll-up of all assessments per term per student.

*Relevant files:* `backend/internal/curriculum/`, `backend/internal/assessments/`, `frontend/src/features/curriculum/`, `frontend/src/features/assessments/`

### 2.5 Behaviour Management

- **Behaviour categories** — school-configurable incident types with optional default severity.
- **Behaviour notes** — teacher-submitted incident reports.
- **Approval queue** — admin reviews and approves/rejects notes before they reach parents.
- **Severity levels** — configurable severity per note.
- **Teacher notes view** — teachers see their own submitted notes and their status.

*Relevant files:* `backend/internal/behavior/`, `frontend/src/features/behavior/`

### 2.7 Timetable & Scheduling

- **Time blocks** — define daily time blocks (periods) with start/end times.
- **Timetable slots** — assign subjects/activities to specific time blocks and days.
- **Day replication** — copy a schedule from one day to another.
- **Blueprint templates** — reusable timetable templates.
- **Day-of-week organisation** — full week grid view with enriched slot details.

*Relevant files:* `backend/internal/timetablestructure/`, `frontend/src/features/timetable-structure/`

### 2.8 Health & Medical

- **Student health profiles** — blood group, allergies, chronic conditions, medications, immunisations, doctor info.
- **Medical incident logging** — nurses log incidents with symptoms, actions taken, and timestamps.
- **Incident list** — searchable, filterable incident history per student.
- **Nurse dashboard** — dedicated interface for nursing staff.

*Relevant files:* `backend/internal/health/`, `frontend/src/features/health/`, `frontend/src/features/nurses/`

### 2.9 Finance & Billing

#### Fee Configuration
- **Fee categories** — configurable categories (Tuition, Transport, Lunch, etc.) with mandatory flag.
- **Fee templates** — per-term, per-grade-level fee amounts linked to categories.
- **Bulk invoice generation** — auto-generate invoices from fee templates or custom items.

#### Invoicing & Payments
- **Per-student invoices** — itemised invoices per term with due amounts.
- **Payment recording** — record partial, full, or waived payments.
- **Payment tracking** — statuses: Unpaid, Partial, Paid, Waived.
- **Invoice detail view** — nested items, payments, and full history.
- **Finance staff management** — assign finance roles within a school.

*Relevant files:* `backend/internal/billing/`, `frontend/src/features/fee-categories/`, `frontend/src/features/fee-templates/`, `frontend/src/features/finance-invoices/`, `frontend/src/features/finance/`

### 2.10 Staff & Role Management

- **Teachers** — manage teacher profiles, active status, and assignments.
- **Admins** — school admin profiles and role management.
- **Nurses** — nurse profiles and active status.
- **Finance staff** — finance role assignment and management.
- **Members list** — paginated, searchable list of all staff by role.
- **Active/inactive toggling** — deactivate staff without deleting records.
- **Bulk staff invitations** — invite multiple staff members via CSV/Excel.

*Relevant files:* `backend/internal/teachers/`, `backend/internal/members/`, `frontend/src/features/teachers/`, `frontend/src/features/admin/`, `frontend/src/features/nurses/`, `frontend/src/features/members/`

### 2.11 Data Import & Export

- **Student bulk import** — CSV/Excel-based import with background processing.
- **Staff bulk invitation** — CSV/Excel-based invitation workflow.
- **Import progress tracking** — real-time job status (processing, completed, failed, cancelled).
- **Failure reporting** — row-level error details with failure type categorisation.
- **Cancellable imports** — in-progress imports can be cancelled.
- **Supported formats:** CSV, XLSX (via `papaparse` + `xlsx` libraries).

*Relevant files:* `backend/internal/imports/`, `frontend/src/features/import-jobs/`

### 2.12 Role-Specific Dashboards

| Role              | Dashboard Components                      |
|-------------------|-------------------------------------------|
| **Teacher**       | Class roster, assessment grading, behavior notes |
| **School Admin**  | Assessment approvals, staff management, curriculum setup, behaviour queue |
| **System Admin**  | Cross-school oversight, school creation |
| **Nurse**         | Health incident logging, student health profiles |
| **Parent**        | Published assessment results, health info |
| **Finance Staff** | Invoice generation, payment recording, fee configuration |

*Relevant files:* `frontend/src/features/dashboard/components/`

---

## 3. Technical Highlights (for landing page)

| Feature                | Detail                                              |
|------------------------|-----------------------------------------------------|
| **Stack**              | Go (Fiber) + Next.js + PostgreSQL                   |
| **Authentication**     | Stytch B2B — passwordless magic links, MFA-ready    |
| **Multi-tenant**       | Full data isolation per school (tenant-scoped queries) |
| **RBAC**               | 6+ roles with distinct dashboards & permissions     |
| **CBC-aligned**        | Competency-Based Curriculum rubric (EE/ME/AE/BE)    |
| **Bulk operations**    | CSV/XLSX import for students & staff with live progress |
| **Dark mode**          | via `next-themes` + shadcn theming                  |
| **Responsive**         | Tailwind CSS, shadcn components                     |
| **Virtualized lists**  | TanStack Virtual for large data sets                |
| **Real-time query**    | TanStack React Query with cache invalidation        |
| **Type-safe**          | Full TypeScript frontend, OpenAPI code generation   |
| **Testing**            | Vitest, Playwright, Go unit + integration tests     |

---

## 4. Marketing Page Sections (Suggested)

Based on the feature inventory above, the landing page could include:

1. **Hero** — "Somo Tracker: Modern School Management"
2. **Multi-Tenant / RBAC** — "Designed for every role — teachers, admins, parents, nurses, finance"
3. **Curriculum & Assessment** — "CBC-aligned curriculum builder + full grading workflow"
5. **Behaviour** — "Manage behaviour notes with admin approval before parent notification"
6. **Health** — "Student health profiles and medical incident logging"
7. **Finance** — "Fee configuration, invoice generation, and payment tracking"
8. **Timetable** — "Period scheduling and day replication"
9. **Bulk Import** — "Import students and staff in bulk via CSV/Excel"
10. **Security & Auth** — "Passwordless magic-link authentication with MFA"
11. **Pricing / CTA** — (TBD)
12. **Footer** — links, contact, legal

---

> **Next step:** Use this document as the source of truth when building `public/src/routes/` pages. Each section can be mapped to a component under `public/src/lib/components/marketing/` per the `public/AGENTS.md` contract.
