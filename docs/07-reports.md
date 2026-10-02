# 07 — Reports & Analytics

## Overview
Dashboard with academic, attendance, behavior, and finance analytics. Aggregates data from multiple domains for school leadership.

---

## Backend Changes

### 1. Report Queries (New SQLC Files)
**Files:**
- `backend/db/queries/reports_academic.sql`
- `backend/db/queries/reports_attendance.sql`
- `backend/db/queries/reports_behavior.sql`
- `backend/db/queries/reports_finance.sql`

### 2. Key Report Endpoints

#### Academic Reports
```
GET /api/reports/academic/grade-distribution
  Query: class_room_id, term_id, subject_id
  Returns: { grade, count, percentage }[]

GET /api/reports/academic/class-performance
  Query: class_room_id, term_id
  Returns: { subject, average, pass_rate, student_count }[]

GET /api/reports/academic/student-progress
  Query: student_id, term_from, term_to
  Returns: term-by-term subject scores
```

#### Attendance Reports
```
GET /api/reports/attendance/summary
  Query: date_from, date_to, grade, stream
  Returns: { total_sessions, submitted, in_progress, missed, rate }[]

GET /api/reports/attendance/student
  Query: student_id, date_from, date_to
  Returns: { date, status, class, subject }[]

GET /api/reports/attendance/class
  Query: class_room_id, date_from, date_to
  Returns: per-student attendance rates
```

#### Finance Reports
```
GET /api/reports/finance/collection
  Query: date_from, date_to
  Returns: { period, invoiced, collected, outstanding }[]

GET /api/reports/finance/outstanding
  Query: class_room_id, grade
  Returns: student balances with aging
```

### 3. Service & Handler
- `ReportsService` — orchestrates queries
- `ReportsHandler` — all `/api/reports/*` endpoints

### 4. Export Endpoints
```
GET /api/reports/export/attendance?format=csv|pdf
GET /api/reports/export/grades?format=csv|pdf
GET /api/reports/export/finance?format=csv|pdf
```

---

## Frontend Changes

### 1. Feature Module
```
src/features/reports/
├── components/
│   ├── reports-dashboard.tsx
│   ├── grade-distribution-chart.tsx
│   ├── attendance-summary-chart.tsx
│   ├── finance-collection-chart.tsx
│   ├── report-filters.tsx
│   └── export-button.tsx
├── hooks/
│   └── use-reports.ts
├── services/
│   └── api.ts
├── types/
│   └── report.ts
└── index.ts
```

### 2. Page
**File:** `src/app/(dashboard)/reports/page.tsx`

```tsx
import { ReportsDashboard } from "@/features/reports";

export default function ReportsPage() {
    return (
        <div className="space-y-6">
            <h1 className="text-2xl font-semibold">Reports & Analytics</h1>
            <ReportsDashboard />
        </div>
    );
}
```

### 3. Navigation
Already in nav: `{ title: "Reports", url: "/reports", icon: <BarChart3Icon /> }`

---

## Acceptance Criteria
- [ ] Dashboard with key metrics cards (enrollment, attendance rate, collection rate)
- [ ] Grade distribution charts (bar/pie)
- [ ] Attendance trends (line chart over time)
- [ ] Finance collection trends
- [ ] Filter by date range, grade, stream, class
- [ ] Export to CSV/PDF
- [ ] Responsive charts (recharts or similar)
- [ ] Role-based visibility (admins see all, teachers see own classes)

---

## Dependencies
- Requires: All core modules (Attendance, Assessments, Finance, Enrollments)
- Blocked by: Assessments (04), Finance (06)

---

## Estimated Effort
- Backend: ~12 hours (aggregation queries, exports)
- Frontend: ~15 hours (dashboard, charts, exports)
- **Total: ~27 hours**