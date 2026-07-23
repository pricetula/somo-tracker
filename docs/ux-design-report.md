# UX Design Report: SomoTracker Dashboard Components by User Role

This report analyzes the user experience (UX) design of the SomoTracker dashboard components, categorized by the primary user roles. The analysis focuses on information architecture, workflow efficiency, clarity, consistency, and data visualization where applicable.

## 1. System Administrator

**Primary Responsibilities:** School management, academic year configuration, user management (admins, teachers, parents, nurses, finance).

**Key Components & Areas:**
*   `/src/app/(dashboard)/admins`
*   `/src/app/(dashboard)/academic-years`
*   `/src/app/(dashboard)/@modal/(.)schools/new` (School creation)
*   `/src/features/dashboard/components/system-admin-dashboard.tsx`
*   `/src/features/academic-years/components/academic-year-form.tsx`
*   Bulk invite components (`src/components/shared/bulk-invite`)
*   Import functionalities (`src/app/(dashboard)/imports`)

**UX Analysis:**
*   **Information Architecture:** The clear separation of academic terms/years and user management (admins, teachers, parents, nurses) in the dashboard navigation suggests a logical information architecture for system-level controls. Modals for new school/academic year creation are good for focused tasks.
*   **Workflow Efficiency:** Bulk import functionalities (e.g., `bulk-invite-file-importer.tsx`, `bulk-invite-manual-form.tsx`) are crucial for efficiency when onboarding large numbers of users or data. The presence of these components indicates an attempt to streamline these processes.
*   **Clarity & Feedback:** For import processes, components like `step-column-mapping.tsx`, `step-data-review.tsx`, `step-streaming.tsx`, and `step-upload.tsx` suggest a guided, multi-step workflow. This is excellent for clarity, providing users with feedback at each stage of a potentially complex operation. Error reporting for imports (`import-job-detail.tsx`) is also critical.
*   **Consistency:** Assuming standard Shadcn UI components are used for forms and tables, a consistent look and feel across different management screens (e.g., managing admins vs. academic years) should be maintained.

**Recommendations:**
*   **Enhanced Audit Logs:** Ensure all critical system admin actions (school creation, major configuration changes, bulk imports) have clear, easily accessible audit logs with details on who, what, and when.
*   **Configuration Wizards:** For complex initial setups (e.g., new school setup, academic year rollovers), consider guided wizards beyond simple forms, especially if there are interdependencies between settings.
*   **Role-Based Access Visualization:** For user management, a clear visual representation of assigned roles and permissions would enhance clarity and prevent errors.

## 2. School Administrator

**Primary Responsibilities:** Managing school-specific data: students, classes, teachers, curriculum, finance, attendance, behavior, and reports.

**Key Components & Areas:**
*   `/src/app/(dashboard)/classes`, `/students`, `/teachers`, `/parents`, `/curriculum`, `/finance`, `/assessments`, `/attendance`, `/behavior`, `/health`, `/reports`, `/settings`
*   `/src/features/dashboard/components/school-admin-dashboard.tsx`
*   Lists and detail views for all major entities (e.g., `academic-years-list.tsx`, `class-detail-view.tsx`, `student-profile-card.tsx`)
*   Forms for creating/editing entities (e.g., `add-class-form.tsx`, `student-form.tsx`)
*   Bulk enrollment and import for students (`enroll-students-panel.tsx`, `students-import`)

**UX Analysis:**
*   **Information Architecture:** The dashboard is highly modular, with distinct sections for each functional area (classes, students, finance, etc.). This organization is good for discoverability. The use of nested routes and modals (e.g., `(.)students/[id]/link-parent`) indicates thoughtful task-specific flows.
*   **Workflow Efficiency:** Components like `enroll-students-panel.tsx` and the `students-import` feature suggest a focus on efficient bulk operations, which is critical for school administration. Dedicated forms for creating entities (`add-class-form.tsx`, `create-parent-form.tsx`) should streamline data entry.
*   **Clarity & Feedback:** The detailed views (e.g., `class-detail-view.tsx`, `student-profile-card.tsx`) are essential for providing comprehensive information. The curriculum tree (`curriculum-tree.tsx`) is a good visual for complex curricular structures. Clear dialogs for creating new items (`create-indicator-dialog.tsx`, `create-learning-area-dialog.tsx`) are positive.
*   **Data Visualization:** The presence of `data-table.tsx` with filtering capabilities indicates a strong foundation for managing lists of entities. The reports section (`/src/app/(dashboard)/reports`) likely offers aggregated data.

**Recommendations:**
*   **Cross-Functional Dashboards:** While modularity is good, the main school admin dashboard (`school-admin-dashboard.tsx`) should provide quick access to key metrics and alerts across different domains (e.g., unreviewed behavior notes, upcoming fee deadlines, attendance anomalies).
*   **Search & Filtering Enhancements:** Ensure robust global search and advanced filtering options are available across all data tables, as school admins deal with large datasets.
*   **Notifications & Alerts:** Implement a clear system for notifications regarding important events (e.g., new behavior notes, student health incidents, pending approvals).

## 3. Teacher

**Primary Responsibilities:** Managing their assigned classes, students, marking attendance, recording behavior, and conducting assessments.

**Key Components & Areas:**
*   `/src/app/(dashboard)/classes/[id]` (class-specific view)
*   `/src/app/(dashboard)/students/[id]` (student-specific view)
*   `/src/app/(dashboard)/attendance/mark/[slot_id]/[date]`
*   `/src/app/(dashboard)/behavior`
*   `/src/app/(dashboard)/assessments` (grading)
*   `/src/app/(dashboard)/timetable`
*   `/src/features/dashboard/components/teacher-dashboard.tsx`
*   `/src/features/attendance/components/attendance-mark-page.tsx`
*   `/src/features/behavior/components/create-behavior-note-dialog.tsx`, `teacher-behavior-view.tsx`
*   `/src/features/assessments/components/rubric-grading-matrix.tsx`, `grading-sheet.tsx`
*   `/src/features/classteachers/components/assign-teacher-dialog.tsx`

**UX Analysis:**
*   **Information Architecture:** The direct paths to mark attendance, record behavior, and manage assessments suggest that core teacher workflows are easily accessible. The `teacher-dashboard.tsx` should act as a central hub.
*   **Workflow Efficiency:** Marking attendance (`attendance-mark-page.tsx`) and grading (`rubric-grading-matrix.tsx`, `grading-sheet.tsx`) are frequent tasks. These components should be optimized for quick data entry (e.g., keyboard shortcuts, clear visual states, bulk actions if applicable).
*   **Clarity & Feedback:** The presence of specific grading components like `rubric-grading-matrix.tsx` indicates an intention to provide clear structures for assessment. Behavior notes (`create-behavior-note-dialog.tsx`) need to be easy to create and categorize.
*   **Data Visualization:** `teacher-performance-radar.tsx`, `teacher-kpi-cards.tsx` in analytics hint at performance feedback for teachers.

**Recommendations:**
*   **Streamlined Data Entry:** For attendance and grading, minimize clicks and form fields. Consider "quick entry" modes or templates for common scenarios.
*   **Student Contextual Information:** When viewing a student, teachers should have quick access to a summary of their attendance, behavior, and assessment performance without navigating away. `student-profile-card.tsx` should be robust for this.
*   **Lesson Planning Integration:** While not directly visible, linking curriculum elements to the timetable or lesson planning tools would greatly enhance the teacher's workflow.
*   **Mobile Responsiveness:** Teachers often need to interact with the system on the go. Ensure all core functionalities are highly usable on tablets and phones.

## 4. Parent

**Primary Responsibilities:** Viewing student progress, attendance, behavior, and possibly financial invoices.

**Key Components & Areas:**
*   `/src/app/(dashboard)/reports/student/[id]`
*   `/src/app/(dashboard)/finance/invoices/[id]`
*   `/src/features/dashboard/components/parent-dashboard.tsx`
*   `/src/features/assessments/components/parent-assessments-view.tsx`
*   `/src/features/behavior/components/parent-behavior-view.tsx`
*   `/src/features/students/components/student-profile-card.tsx`
*   `link-parent-dialog.tsx`, `link-parent-form.tsx` (for associating parents with students)

**UX Analysis:**
*   **Information Architecture:** The parent dashboard and specific views for student reports, assessments, and behavior suggest a clear, student-centric organization. Parents typically access information related to their child(ren) and finance.
*   **Workflow Efficiency:** The primary "workflow" for parents is information consumption. The ease of finding reports, grades, and communication is key.
*   **Clarity & Feedback:** Reports should be easily understandable, potentially with visual summaries (charts from `/src/features/analytics/components/student-term-overall`, `/src/features/analytics/components/student-behavior`). Language should be clear and jargon-free.
*   **Data Visualization:** Analytics components like `level-distribution-bar.tsx`, `overall-gauge.tsx`, `behavior-pie-chart.tsx`, `behavior-trend-line.tsx` are critical for parents to quickly grasp their child's performance and areas of concern.

**Recommendations:**
*   **Simplified Reporting Language:** Ensure reports are presented in parent-friendly language, avoiding educational jargon. Provide clear explanations for metrics.
*   **Proactive Notifications:** Implement notifications for parents regarding critical updates (e.g., low attendance, new behavior notes, upcoming invoice due dates).
*   **Communication Channel:** A clear and easy-to-use communication channel with teachers/school administration directly from the parent dashboard.
*   **"My Children" Overview:** If a parent has multiple children, the `parent-dashboard.tsx` should provide a consolidated, at-a-glance overview of each child's key status indicators.

## 5. Nurse

**Primary Responsibilities:** Managing student health records and incidents.

**Key Components & Areas:**
*   `/src/app/(dashboard)/health`
*   `/src/app/(dashboard)/health/students/[studentId]`
*   `/src/features/dashboard/components/nurse-dashboard.tsx`
*   `/src/features/health/components/create-incident-dialog.tsx`
*   `/src/features/health/components/incident-list.tsx`
*   `/src/features/health/components/student-health-view.tsx`

**UX Analysis:**
*   **Information Architecture:** A dedicated "Health" section within the dashboard is appropriate. The ability to drill down to specific student health views (`student-health-view.tsx`) is crucial.
*   **Workflow Efficiency:** Creating new incident reports (`create-incident-dialog.tsx`) should be quick and capture essential information efficiently, especially in urgent situations.
*   **Clarity & Feedback:** Incident lists (`incident-list.tsx`) should clearly display critical information such as date, type of incident, and resolution status. Student health views should aggregate all relevant medical information clearly.
*   **Data Security & Privacy:** While not directly a UX concern, the design should implicitly reinforce data privacy, ensuring only authorized personnel have access to sensitive health information.

**Recommendations:**
*   **Emergency Quick-Entry:** For critical incidents, a "quick entry" form or button accessible from multiple points for immediate logging of incidents.
*   **Alerts & Reminders:** System to remind nurses of follow-ups, medication schedules, or upcoming appointments for students.
*   **Integration with Student Profiles:** Seamless access to a student's basic profile information (e.g., parent contact, known allergies) directly from the health view.

## 6. Finance Administrator

**Primary Responsibilities:** Managing fee categories, templates, invoices, and financial records.

**Key Components & Areas:**
*   `/src/app/(dashboard)/finance`
*   `/src/app/(dashboard)/finance/fee-categories`
*   `/src/app/(dashboard)/finance/fee-templates`
*   `/src/app/(dashboard)/finance/invoices`
*   `/src/features/dashboard/components/finance-dashboard.tsx`
*   `/src/features/fee-categories/components/fee-categories-list.tsx`
*   `/src/features/fee-templates/components/fee-templates-list.tsx`
*   `/src/features/finance-invoices/components/invoices-list.tsx`, `invoice-detail.tsx`

**UX Analysis:**
*   **Information Architecture:** The clear separation into fee categories, templates, and invoices within the finance section is logical for financial management.
*   **Workflow Efficiency:** Components for managing lists (`fee-categories-list.tsx`, `fee-templates-list.tsx`, `invoices-list.tsx`) should facilitate quick sorting, filtering, and bulk actions (e.g., generating multiple invoices).
*   **Clarity & Feedback:** Invoice detail views (`invoice-detail.tsx`) need to be comprehensive yet easy to read, with clear status indicators (paid, overdue, partially paid). Financial reports should be generated clearly.
*   **Reporting Capabilities:** The finance section likely requires robust reporting features, potentially integrating with analytics components for financial trends.

**Recommendations:**
*   **Automated Billing & Reminders:** Features for setting up recurring invoices and automated reminders for overdue payments would greatly enhance efficiency.
*   **Reconciliation Tools:** Tools to easily reconcile payments against invoices, potentially with import capabilities for bank statements.
*   **Financial Dashboards:** The `finance-dashboard.tsx` should prominently display key financial health indicators, such as outstanding balances, revenue trends, and upcoming payment deadlines.

---

**General UX Considerations (Applicable to all roles):**

*   **Responsiveness:** All components and layouts should be fully responsive and optimized for various screen sizes, from desktop to mobile, given the diverse environments users might access the system from.
*   **Loading States & Skeletons:** Implement clear loading indicators and skeleton screens (e.g., `skeleton.tsx`, `skeleton-rows.tsx`) to manage user expectations during data fetching.
*   **Error Handling:** Consistent and informative error messages (as per the `AGENTS.md` spec) should guide users when issues arise, preventing frustration.
*   **Accessibility:** Adherence to WCAG guidelines is crucial for all users, especially those with disabilities. Utilizing Shadcn components helps with this, but custom implementations need careful review.
*   **Navigation & Search:** A consistent and intuitive global navigation, along with powerful search capabilities, will allow users to quickly find what they need, regardless of their role. The `app-sidebar.tsx` and `nav-main.tsx` are critical for this.
