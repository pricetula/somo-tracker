# Somotracker — Endpoint Gap Analysis

> **Generated:** 2026-07-02  
> **Schema version:** `000001_initial_schema.up.sql` (squashed, CBC-only v5)  
> **Backend layer:** `/backend`

---

## 1. Schema-Domain Mapping

| Layer | Domain | Tables |
|-------|--------|--------|
| L1 — Platform | `tenant` | `tenants` |
| L1 — Platform | `auth` | `users`, `sessions` |
| L2 — Core CBC Actors | `cbcschools` | `cbc_schools` |
| L3 — Academic Calendar | `academicyears` | `academic_years`, `academic_terms` |
| L2 (cont.) — Core CBC Actors | `cbcstreams` | `cbc_streams` |
| L2 (cont.) — Core CBC Actors | `cbcclasses` | `cbc_classes` |
| L2 (cont.) — Core CBC Actors | `members` | `memberships`, `school_member_counts` |
| L2 (cont.) — Core CBC Actors | `invitations` | `invitations` |
| L2 (cont.) — Core CBC Actors | `imports` | `import_jobs`, `import_job_failures`, `import_job_staging` |
| L2 (Core CBC Actors) | `parents` | `cbc_parents`, `cbc_student_parents` |
| L2 (Core CBC Actors) | `students` | `cbc_students`, `cbc_student_enrollments` |
| L4 — Health & Financials | `billing` | `fee_categories`, `fee_templates`, `invoices`, `invoice_items`, `payments` |
| L4 — Health & Financials | *(none — health not yet implemented)* | `medical_incidents`, `student_health_profiles` |
| L5 — CBC Curriculum | `curriculum` | `cbc_learning_areas`, `cbc_strands`, `cbc_sub_strands`, `performance_indicators` |
| L6 — Teacher Assignments, Timetable | `timetable` | `cbc_timetable_slots`, `cbc_class_teachers` |
| L6 — Teacher Assignments, Attendance | `attendance` | `cbc_attendance_periods`, `cbc_attendance_logs` |
| L7 — Assessment Architecture | `assessment` | `assessment_weight_configs`, `assessment_blueprints`, `assessment_blueprint_indicators`, `cbc_assessment_grading_scales` |
| L8 — Assessment Execution | `assessment` (same) | `assessment_sessions`, `learner_rubric_results` |
| L8 — Assessment Execution | `portfolio` | `learner_portfolios` |
| L9 — Aggregation & Reporting | `summaries` | `cbc_term_report_cards`, `cbc_term_competency_summaries` |
| L10 — User Context | `activeschool` | `member_active_school` |

---

## 2. Implemented Endpoints (by Handler)

### 2.1 `tenant` — `/tenants`

| Method | Endpoint | Handler |
|--------|----------|---------|
| `POST` | `/tenants` | `Create` |

### 2.2 `auth` — `/api/auth`

| Method | Endpoint | Handler |
|--------|----------|---------|
| `POST` | `/api/auth/discover` | `Discover` |
| `POST` | `/api/auth/verify` | `Verify` |
| `POST` | `/api/auth/register` | `Register` |
| `GET` | `/api/auth/callback` | `MagicLinkCallback` |
| `GET` | `/api/auth/invite/callback` | `AcceptInvite` |
| `GET` | `/api/auth/me` | `Me` |
| `DELETE` | `/api/auth/session` | `Logout` |

### 2.3 `cbcschools` — `/api/v1/schools`

| Method | Endpoint | Handler |
|--------|密宗eyes-up| `"'"`
| `POST` | `/api/v1/schools` | `Create` |
| `GET` | `/api/v1/schools` | `List` |
| `PUT` | `/api/v1/schools/:id` | `Update` |
| `DELETE` | `/api/v1/schools/:id` | `Delete` |

### 2.4 `academicyears` — `/api/v1/academic-years` / `/api/v1/academic-terms`

| Method | Endpoint | Handler |
|--------|----------|---------|
| `GET` | `/api/v1/academic-years` | `ListYears` |
| `PATCH` | `/api/v1/academic-years/:id` | `PatchYear` |
| `POST` | `/api/v1/academic-years/:id/set-current` | `SetCurrentYear` |
| `DELETE` | `/api/v1/academic-years/:id` | `DeleteYear` |
| `GET` | `/api/v1/academic-terms` | `ListTerms` |
| `POST` | `/api/v1/academic-terms` | `CreateTerm` |
| `PATCH` | `/api/v1/academic-terms/:id` | `PatchTerm` |
| `DELETE` | `/api/v1/academic-terms/:id` | `DeleteTerm` |

### 2.5 `cbcstreams` — `/api/v1/streams`

| Method | Endpoint | Handler |
|--------|----------|---------|
| `GET` | `/api/v1/streams` | `List` |
| `POST` | `/api/v1/streams` | `Create` |
| `PUT` | `/api/v1/streams/:id` | `Update` |
| `DELETE` | `/api/v1/streams/:id` | `Delete` |

### 2.6 `cbcclasses` — `/api/v1/classes`

| Method | Endpoint | Handler |
|--------|----------|---------|
| `GET` | `/api/v1/classes` | `List` |
| `POST` | `/api/v1/classes` | `Create` |
| `PUT` | `/api/v1/classes/:id` | `Update` |
| `DELETE` | `/api/v1/classes` | `BulkDelete` |

### 2.7 `members` — `/api/v1/members`

| Method | Endpoint | Handler |
|--------|----------|---------|
| `GET` | `/api/v1/members` | `List` |
| `PATCH` | `/api/v1/members/:user_id/active` | `ToggleActive` |

### 2.8 `invitations` — `/api/v1/invitations`

| Method | Endpoint | Handler |
|--------|----------|---------|
| `GET` | `/api/v1/invitations` | `ListInvitations` |

### 2.9 `imports` — `/api/v1/imports/*`

| Method | Endpoint | Handler | Notes |
|--------|----------|---------|-------|
| `POST` | `/api/v1/imports/staff` | `StartImport` | |
| `GET` | `/api/v1/imports/staff/track/:id` | `TrackImport` | |
| `GET` | `/api/v1/imports/staff/track/:id/sse` | `SSETrackImport` | |
| `GET` | `/api/v1/imports/staff/:id/failures` | `ListFailedInvitations` | |
| `POST` | `/api/v1/imports/students` | `StartStudentImport` | |
| `GET` | `/api/v1/imports/students/stream` | `SSEStudentImportStream` | |
| `GET` | `/api/v1/parents` | `ListParents` | *(misc. support)* |
| `GET` | `/api/v1/classes` | `ListClasses` | *(misc. support)* |
| `GET` | `/api/v1/students` | `ListExistingStudents` | *(misc. support)* |
| `GET` | `/api/v1/academic/years` | `ListAcademicYears` | *(misc. support)* |
| `GET` | `/api/v1/academic/periods` | `ListAcademicPeriods` | *(misc. support)* |

### 2.10 `teachers` — `/api/v1/teachers`

| Method | Endpoint | Handler |
|--------|----------|---------|
| `GET` | `/api/v1/teachers` | `List` |
| `PATCH` | `/api/v1/teachers/:user_id/active` | `ToggleActive` |

### 2.11 `parents` — `/api/v1/parents`

| Method | Endpoint | Handler |
|--------|----------|---------|
| `POST` | `/api/v1/parents` | `Create` |
| `GET` | `/api/v1/parents` | `List` |
| `GET` | `/api/v1/parents/:id` | `GetDetail` |
| `PUT` | `/api/v1/parents/:id` | `Update` |
| `DELETE` | `/api/v1/parents/:id` | `Delete` |
| `POST` | `/api/v1/parents/:parent_id/students` | `LinkStudent` |
| `DELETE` | `/api/v1/parents/:parent_id/students/:student_id` | `UnlinkStudent` |

### 2.12 `students` — `/api/v1/students`

| Method | Endpoint | Handler |
|--------|----------|---------|
| `GET` | `/api/v1/students/list` | `List` |
| `POST` | `/api/v1/students` | `Create` |
| `GET` | `/api/v1/st_rest+1)(true` | `GetDetail` |
| `PUT` | `/api/v1/students/:id` | `Update` |
| `POST` | `/api/v1/students/:id/enrollments` | `CreateEnrollment` |
| `GET` | `/api/v1/students/:id/enrollments` | `ListEnrollments` |

### 2.13 `billing` — `/api/v1/billing`

| Method | Endpoint | Handler |
|--------|----------|---------|
| `POST` | `/api/v1/billing/fee-categories` | `CreateFeeCategory` |
| `GET` | `/api/v1/billing/fee-categories` | `ListFeeCategories` |
| `PUT` | `/api/v1/billing/fee-categories/:id` | `UpdateFeeCategory` |
| `DELETE` | `/api/v1/billing/fee-categories/:id` | `DeleteFeeCategory` |
| `POST` | `/api/v1/billing/fee-templates` | `CreateFeeTemplate` |
| `GET` | `/api/v1/billing/fee-templates` | `ListFeeTemplates` |
| `PUT` | `/api/v1/billing/fee-templates/:id` | `UpdateFeeTemplate` |
| `DELETE` | `/api/v1/billing/fee-templates/:id` | `DeleteFeeTemplate` |
| `POST` | `/api/v1/billing/invoices/generate` | `GenerateInvoice` |
| `GET` | `/api/v1/billing/invoices` | `ListInvoices` |
| `GET` | `/api/v1/billing/invoices/:id` | `GetInvoiceDetail` |
| `POST` | `/api/v1/billing/invoices/:id/waive` | `WaiveInvoice` |
| `POST` | `/api/v1/billing/payments` | `RecordPayment` |
| `GET` | `/api/v1/billing/payments` | `ListPayments` |

### 2.14 `curriculum` — `/api/v1/curriculum/*`

| Method | Endpoint | Handler |
|--------|----------|---------|
| `POST` | `/api/v1/curriculum/learning-areas` | `CreateLearningArea` |
| `GET` | `/api/v1/curriculum/learning-areas` | `ListLearningAreas` |
| `GET` | `/api/v1/curriculum/learning-areas/:id` | `GetLearningAreaByID` |
| `GET` | `/api/v1/curriculum/learning-areas/:id/tree` | `GetTree` |
| `PUT` | `/api/v1/curriculum/learning-areas/:id` | `UpdateLearningArea` |
| `DELETE` | `/api/v1/curriculum/learning-areas/:id` | `DeleteLearningArea` |
| `POST` | `/api/v1/curriculum/strands` | `CreateStrand` |
| `GET` | `/api/v1/curriculum/strands` | `ListStrands` |
| `PUT` | `/api/v1/curriculum/strands/:id` | `UpdateStrand` |
| `DELETE` | `/api/v1/curriculum/strands/:id` | `DeleteStrand` |
| `POST` | `/api/v1/curriculum/sub-strands` | `CreateSubStrand` |
| `GET` | `/api/v1/curriculum/sub-strands` | `ListSubStrands` |
| `PUT` | `/api/v1/curriculum/sub-strands/:id` | `UpdateSubStrand` |
| `DELETE` | `/api/v1/curriculum/sub-strands/:id` | `DeleteSubStrand` |
| `POST` | `/api/v1/curriculum/performance-indicators` | `CreatePerformanceIndicator` |
| `GET` | `/api/v1/curriculum/performance-indicators` | `ListPerformanceIndicators` |
| `PUT` | `/api/v1/curriculum/performance-indicators/:id` | `UpdatePerformanceIndicator` |
| `DELETE` | `/api/v1/curriculum/performance-indicators/:id` | `DeletePerformanceIndicator` |

### 2.15 `timetable` — `/api/v1/schools/:schoolId/timetable` / `/api/v1/schools/:schoolId/classes/:classId/teachers`

| Method | Endpoint | Handler |
|--------|----------|---------|
| `POST` | `/api/v1/schools/:schoolId/timetable/slots/bulk` | `BulkCreateSlots` |
| `GET` | `/api/v1/schools/:schoolId/timetable/slots` | `ListSlots` |
| `POST` | `/api/v1/schools/:schoolId/classes/:classId/teachers` | `AssignTeacher` |
| `DELETE` | `/api/v1/schools/:schoolId/classes/:classId/teachers/:userId` | `RemoveTeacher` |

### 2.16 `attendance` — `/api/v1/schools/:schoolId/attendance`

| Method | Endpoint | Handler |
|--------|----------|---------|
| `POST` | `/api/v1/schools/:schoolId/attendance` | `SubmitAttendance` |
| `GET` | `/api/v1/schools/:schoolId/attendance/periods/:periodId` | `GetPeriod` |

### 2.17 `assessment` — `/api/v1/assessment/*`

| Method | Endpoint | Handler |
|--------|----------|---------|
| `POST` | `/api/v1/assessment/blueprints` | `CreateBlueprint` |
| `GET` | `/api/v1/assessment/blueprints` | `ListBlueprints` |
| `GET` | `/api/v1/assessment/blueprints/:id` | `GetBlueprintDetail` |
| `PUT` | `/api/v1/assessment/blueprints/:id` | `UpdateBlueprint` |
| `DELETE` | `/api/v1/assessment/blueprints/:id` | `DeleteBlueprint` |
| `POST` | `/api/v1/assessment/blueprints/:id/indicators` | `LinkIndicators` |
| `DELETE` | `/api/v1/assessment/blueprints/:id/indicators/:indicator_id` | `UnlinkIndicator` |
| `POST` | `/api/v1/assessment/sessions` | `CreateSession` |
| `GET` | `/api/v1/assessment/sessions` | `ListSessions` |
| `GET` | `/api/v1/assessment/sessions/:id` | `GetSessionDetail` |
| `PUT` | `/api/v1/assessment/sessions/:id` | `UpdateSession` |
| `DELETE` | `/api/v1/assessment/sessions/:id` | `DeleteSession` |
| `POST` | `/api/v1/assessment/sessions/:id/results/batch` | `BatchUpsertResults` |
| `GET` | `/api/v1/assessment/sessions/:id/results` | `ListResults` |
| `GET` | `/api/v1/assessment/weight-configs` | `ListWeightConfigs` |

### 2.18 `portfolio` — `/api/v1/portfolio/entries`

| Method | Endpoint | Handler |
|--------|----------|---------|
| `POST` | `/api/v1/portfolio/entries` | `CreateEntry` |
| `GET` | `/api/v1/portfolio/entries` | `ListEntries` |
| `GET` | `/api/v1/portfolio/entries/:id` | `GetEntry` |
| `PUT` | `/api/v1/portfolio/entries/:id` | `UpdateEntry` |
| `DELETE` | `/api/v1/portfolio/entries/:id` | `DeleteEntry` |

### 2.19 `summaries` — `/api/v1/summaries`

| Method | Endpoint | Handler |
|--------|----------|---------|
| `GET` | `/api/v1/summaries` | `ListSummaries` |
| `GET` | `/api/v1/summaries/:id` | `GetSummary` |
| `PUT` | `/api/v1/summaries/:id/override` | `SetOverrideLevel` |
| `POST` | `/api/v1/summaries/calculate` | `CalculateSummaries` |
| `POST` | `/api/v1/summaries/c MEMORIES+1)(true` | `CalculateForClass` |
| `POST` | `/api/v1/summaries/:id/mark-synced` | `MarkSynced` |

### 2.20 `activeschool` — `/api/v1/active-school`

| Method | Endpoint | Handler |
|--------|----------|---------|
| `PUT` | `/api/v1/active-school` | `Switch` |
| `GET` | `/api/v1/active-school` | `Get` |

---

## 3. Missing Endpoints (Gaps)

### 3.1 `users` — No dedicated handler package

The `users` table exists but there is no `internal/users/` package. The `auth` package handles creation (via `register`) and `GetMe`, but general user management is missing.

| Proposed Endpoint | Method | Purpose |
|-------------------|--------|---------|
| `GET /api/v1/users` | `GET` | List users in a tenant |
| `GET /api/v1/users/:id` | `GET` | Get user detail |
| `PATCH /api/v1/users/:id` | `PATCH` | Update user (name, tsc_number, knec_panel_assessor_id) |
| `DELETE /api/v1/users/:id` | `DELETE` | Deactivate user (soft-delete via `is_active`) |
| `GET /api/v1/users/:id/sessions` | `GET` | List active sessions for a user |
| `DELETE /api/v1/users/:id/sessions` | `DELETE` | Revoke all sessions for a user |

### 3.2 `sessions` — No dedicated handler package

Session management is partially handled in `auth` (`Logout`), but active session listing/revocation is missing.

| Proposed Endpoint | Method | Purpose |
|-------------------|--------|---------|
| `GET /api/v1/sessions` | `GET` | List active sessions (self) |
| `DELETE /api/v1/sessions/:token` | `DELETE` | Revoke a specific session |
| `DELETE /api/v1/sessions/all` | `DELETE` | Revoke all sessions except current |

### 3.3 `tenant` — Incomplete CRUD

| Proposed Endpoint | Method | Purpose |
|-------------------|--------|---------|
| `GET /tenants/:id` | `GET` | Get tenant by ID |
| `GET /tenants` | `GET` | List tenants (system admin) |
| `PUT /tenants/:id` | `PUT` | Update tenant (name, slug) |
| `DELETE /tenants/:id` | `DELETE` | Delete tenant (cascade) |

### 3.4 `cbcschools` — Could expand

| Proposed Endpoint | Method | Purpose |
|-------------------|--------|---------|
| `GET /api/v1/schools/:id` | `GET` | Get school by ID (currently List only) |
| `GET /api/v1/schools/:id/stats` | `GET` | School stats (student count, staff count) |

### 3.5 `invitations` — Very limited

Only `ListInvitations` exists. The `invitations` table supports full CRUD and lifecycle management.

| Proposed Endpoint | Method | Purpose |
|-------------------|--------|---------|
| `POST /api/v1/invitations` | `POST` | Create & send invitation |
| `GET /api/v1/invitations/:id` | `GET` | Get invitation detail |
| `POST /api/v1/invitations/:id/resend` | `POST` | Resend invitation email |
| `POST /api/v1/invitations/:id/revoke` | `POST` | Revoke pending invitation |
| `DELETE /api/v1/invitations/:id` | `DELETE` | Delete invitation record |

### 3.6 `members` — Could expand with school-scoped operations

| Proposed Endpoint | Method | Purpose |
|-------------------|--------|---------|
| `POST /api/v1/members` | `POST` | Add a member to a school |
| `DELETE /api/v1/members/:user_id` | `DELETE` | Remove a member from a school |
| `GET /api/v1/members/:user_id` | `GET` | Get member detail |

### 3.7 `teachers` — Could expand

| Proposed Endpoint | Method | Purpose |
|-------------------|--------|---------|
| `GET /api/v1/teachers/:user_id` | `GET` | Get teacher detail |
| `PUT /api/v1/teachers/:user_id` | `PUT` | Update teacher profile |
| `GET /api/v1/teachers/:user_id/classes` | `GET` | List classes taught by teacher |
| `GET /api/v1/teachers/:user_id/timetable` | `GET` | Get teacher's timetable |

### 3.8 `students` — Could expand

| Proposed Endpoint | Method | Purpose |
|-------------------|--------|---------|
| `DELETE /api/v1/students/:id` | `DELETE` | Delete student (soft) |
| `GET /api/v1/students/:id/attendance` | `GET` | Get student attendance history |
| `GET /api/v1/students/:id/results` | `GET` | Get student assessment results |\ stripe+/v1/students/:id/portfolio` | `GET` | Get student portfolio entries |
| `GET /api/v1/students/:id/report-cards` | `GET` | Get student report cards |

### 3.9 `cbcstreams` — Could expand

| Proposed Endpoint | Method | Purpose |
|-------------------|--------|---------|
| `GET /api/v1/streams/:id` | `GET` | Get stream detail |
| `GET /api/v1/streams/:id/classes` | `GET` | List classes in stream |

### 3.10 `attendance` — Could expand

| Proposed Endpoint | Method | Purpose |
|-------------------|--------|---------|
| `GET /api/v1/schools/:schoolId/attendance` | `GET` | List attendance periods (with filters) |
| `GET /api/v1/schools/:schoolId/attendance/summary` | `GET` | Get attendance summary for a class/date range |
| `PATCH /api/v1/schools/:schoolId/attendance/periods/:periodId` | `PATCH` | Update an attendance period |
| `DELETE /api/v1/schools/:schoolId/attendance/periods/:periodId` | `DELETE` | Delete an attendance period |

### 3.11 `timetable` — Could expand

| Proposed Endpoint | Method | Purpose |
|-------------------|--------|---------|
| `PUT /api/v1/schools/:schoolId/timetable/slots/:id` | `PUT` | Update a timetable slot |
| `DELETE /api/v1/schools/:schoolId/timetable/slots/:id` | `DELETE` | Delete a timetable slot |
| `GET /api/v1/schools/:schoolId/timetable/slots/:id` | `GET` | Get timetable slot detail |
| `GET /api/v1/schools/:schoolId/classes/:classId/teachers` | `GET` | List teachers assigned to a class |

### 3.12 `assessment` — Could expand

| Proposed Endpoint | Method | Purpose |
|-------------------|--------|---------|
| `GET /api/v1/assessment/grading-scales` | `GET` | List grading scales |
| `POST /api/v1/assessment/grading-scales` | `POST` | Create grading scale |
| `PUT /api/v1/assessment/grading-scales/:id` | `PUT` | Update grading scale |
| `DELETE /api/v1/assessment/grading-scales/:id` | `DELETE` | Delete grading scale |
| `GET /api/v1/assessment/sessions/:id/stats` | `GET` | Get session statistics |

### 3.13 `summaries` — Could expand

| Proposed Endpoint | Method | Purpose |
|-------------------|--------|---------|
| `POST /api/v1/summaries/:id/compile` | `POST` | Compile a report card (trigger fn_refresh_term_attendance_summary) |
| `POST /api/v1/summaries/:id/approve` | `POST` | Teacher approves a report card |
| `POST /api/v1/summaries/:id/publish` | `POST` | Publish report card to parents |
| `GET /api/v1/summaries/:id/attendance` | `GET` | Get detailed attendance breakdown for a report card |

### 3.14 `billing` — Could expand

| Proposed Endpoint | Method | Purpose |
|-------------------|--------|---------|
| `DELETE /api/v1/billing/invoices/:id` | `DELETE` | Delete/cancel an invoice |
| `PUT /api/v1/billing/payments/:id` | `PUT` | Update a payment record |
| `DELETE /api/v1/billing/payments/:id` | `DELETE` | Reverse a payment |
| `GET /api/v1/billing/invoices/:id/payments` | `GET` | List payments for an invoice |
| `GET /api/v1/billing/reports/outstanding` | `GET` | Outstanding fees report |

### 3.15 `curriculum` — Could expand

| Proposed Endpoint | Method | Purpose |
|-------------------|--------|---------|
| `GET /api/v1/curriculum/learning-areas/:id/strands` | `GET` | List strands for a learning area |
| `GET /api/v1/curriculum/sub-strands/:id/indicators` | `GET` | List performance indicators for a sub-strand |
| `POST /api/v1/curriculum/bulk-import` | `POST` | Bulk import curriculum from KICD template |

### 3.16 `portfolio` — Could expand

| Proposed Endpoint | Method | Purpose |
|-------------------|--------|---------|
| `GET /api/v1/portfolio/entries/:id/download` | `GET` | Generate download link for portfolio evidence |
| `GET /api/v1/portfolio/students/:studentId` | `GET` | List all portfolio entries for a student |

### 3.17 `activeschool` — Could expand

| Proposed Endpoint | Method | Purpose |
|-------------------|--------|---------|
| `GET /api/v1/active-school/switchable` | `GET` | List schools the user can switch to |

---

## 4. Completely Missing Domain Modules

These tables have **no handler package at all** and would require a new `internal/` package with handler, service, and repository layers.

### 4.1 `medical_incidents` + `student_health_profiles`

| Proposed Endpoint | Method | Purpose |
|-------------------|--------|---------|
| `GET /api/v1/health/students/:studentId` | `GET` | Get student health profile |
| `PUT /api/v1/health/students/:studentId` | `PUT` | Update student health profile |
| `POST /api/v1/health/incidents` | `POST` | Log a medical incident |
| `GET /api/v1/health/incidents` | `GET` | List medical incidents (with filters) |
| `GET /api/v1/health/incidents/:id` | `GET` | Get incident detail |
| `PUT /api/v1/health/incidents/:id` | `PUT` | Update incident record |

### 4.2 `assessment_weight_configs`

These are seed/admin tables. Consider an admin-only endpoint:

| Proposed Endpoint | Method | Purpose |
|-------------------|--------|---------|
| `POST /api/v1/admin/assessment-weight-configs` | `POST` | Create new weight config (KNEC formula update) |
| `PUT /api/v1/admin/assessment-weight-configs/:id` | `PUT` | Update weight config |

---

## 5. Summary Matrix

| Domain | Implemented | Missing / Potential |
|--------|------------:|--------------------|
| `tenant` | 1 | 4 (Get, List, Update, Delete) |
| `auth` | 7 | session mgmt, user CRUD |
| `cbcschools` | 4 | 2 (GetDetail, Stats) |
| `academicyears` | 8 | — |
| `cbcstreams` | 4 | 2 (GetDetail, ListClasses) |
| `cbcclasses` | 4 | 1 (GetDetail) |
| `members` | 2 | 3 (Add, Remove, GetDetail) |
| `invitations` | 1 | 5 (Create, GetDetail, Resend, Revoke, Delete) |
| `imports` | 8 | — (support endpoints are complete) |
| `teachers` | 2 | 4 (GetDetail, Update, ListClasses, Timetable) |
| `parents` | 7 | 1 (GetLinkedStudents) |
| `students` | 6 | 5 (Delete, Attendance, Results, Portfolio, ReportCards) |
| `billing` | 14 | 5 (DeleteInvoice, UpdatePayment, ReversePayment, InvoicePayments, OutstandingReport) |
| `curriculum` | 16 | 3 (ListStrandsByArea, ListIndicatorsBySubStrand, BulkImport) |
| `timetable` | 4 | 4 (UpdateSlot, DeleteSlot, GetSlot, ListClassTeachers) |
| `attendance` | 2 | 4 (ListPeriods, Summary, UpdatePeriod, DeletePeriod) |
| `assessment` | 14 | 4 (GradingScales CRUD, SessionStats) |
| `portfolio` | 5 | 2 (Download, ListByStudent) |
| `summaries` | 6 | 4 (Compile, Approve, Publish, AttendanceBreakdown) |
| `activeschool` | 2 | 1 (ListSwitchable) |
| **Health (new)** | **0** | **6** |
