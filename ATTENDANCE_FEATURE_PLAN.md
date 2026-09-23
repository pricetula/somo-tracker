# Attendance Sessions Feature — Implementation Plan

## Overview

Build an admin-facing attendance dashboard listing all class attendance sessions with status (Submitted / In Progress / Missed), showing teacher name (link), class name (link), status badge, and session time.

---

## Backend Changes

### 1. New Service Method: `ListAttendanceSessions`

**File:** `backend/internal/services/attendance_service.go`

Add to `AttendanceService` interface:

```go
type AttendanceSession struct {
    SlotID           uuid.UUID `json:"slot_id"`
    ClassID          uuid.UUID `json:"class_id"`
    ClassName        string    `json:"class_name"`
    Grade            string    `json:"grade"`
    Stream           string    `json:"stream"`
    Subject          string    `json:"subject"`
    TeacherName      string    `json:"teacher_name"`
    TeacherID        uuid.UUID `json:"teacher_id"`
    DayOfWeek        int       `json:"day_of_week"`
    TimeSlotName     string    `json:"time_slot_name"`
    StartTime        string    `json:"start_time"`      // HH:MM
    EndTime          string    `json:"end_time"`        // HH:MM
    AttendanceDate   string    `json:"attendance_date"` // YYYY-MM-DD
    Status           string    `json:"status"`          // SUBMITTED | IN_PROGRESS | MISSED
    TotalStudents    int       `json:"total_students"`
    RecordedStudents int       `json:"recorded_students"`
}

type ListAttendanceSessionsParams struct {
    SchoolID       uuid.UUID
    Page           int
    Limit          int
    Search         string      // class name, teacher name, subject
    StatusFilter   string      // SUBMITTED | IN_PROGRESS | MISSED | ""
    DateFrom       *time.Time  // optional
    DateTo         *time.Time  // optional
    GradeFilter    []string
    StreamFilter   []string
    SubjectFilter  []string
    TeacherFilter  []string
}

ListAttendanceSessions(ctx context.Context, params ListAttendanceSessionsParams) ([]AttendanceSession, int, error)
```

**Logic for Status Computation:**

- Get all class timetable slots for the school (with class, subject, teacher, time slot details)
- For each slot, generate attendance dates based on day_of_week within the school's academic term/year date range (or DateFrom/DateTo filters)
- For each (slot, date) pair:
    - Count total students in the class_room
    - Count attendance records in `timetable_attendance` for that slot_id + attendance_date
    - Status:
        - `SUBMITTED` if `recorded_students == total_students && total_students > 0`
        - `IN_PROGRESS` if `recorded_students > 0 && recorded_students < total_students`
        - `MISSED` if `recorded_students == 0 && attendance_date < today`
        - (Future dates with 0 records → exclude or show as UPCOMING if needed)

**Optimization:** Use a single query with window functions or CTEs to avoid N+1. Consider materialized view or pre-computed daily rollup if data grows.

### 2. New API Endpoint

**File:** `backend/internal/api/attendance_handler.go`

```go
// @Summary List attendance sessions for admin dashboard
// @Tags Attendance
// @Produce json
// @Param page query int false "Page number"
// @Param limit query int false "Page size"
// @Param search query string false "Search class/teacher/subject"
// @Param status query string false "Filter: SUBMITTED, IN_PROGRESS, MISSED"
// @Param date_from query string false "YYYY-MM-DD"
// @Param date_to query string false "YYYY-MM-DD"
// @Param grades query string false "Comma-separated grade labels"
// @Param streams query string false "Comma-separated stream names"
// @Param subjects query string false "Comma-separated subject names"
// @Param teachers query string false "Comma-separated teacher names"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/attendance/sessions [get]
func (h *AttendanceHandler) ListAttendanceSessions(c fiber.Ctx) error
```

Register in `router.go`:

```go
protected.Get("/attendance/sessions", r.Attendance.ListAttendanceSessions)
```

---

## Frontend Changes

### 3. New Feature Module: `src/features/attendance/`

```
src/features/attendance/
├── components/
│   ├── attendance-table.tsx      # DataTable with columns
│   └── status-badge.tsx          # Reusable status badge component
├── hooks/
│   └── use-attendance-sessions.ts  # useQuery wrapper
├── services/
│   └── api.ts                    # API client calls
├── types/
│   └── attendance.ts             # TypeScript interfaces
└── index.ts                      # Public exports
```

### 4. Types (`types/attendance.ts`)

```ts
export type AttendanceStatus = "SUBMITTED" | "IN_PROGRESS" | "MISSED";

export interface AttendanceSession {
    slotId: string;
    classId: string;
    className: string;
    grade: string;
    stream: string;
    subject: string;
    teacherName: string;
    teacherId: string;
    dayOfWeek: number;
    timeSlotName: string;
    startTime: string; // "08:00"
    endTime: string; // "09:00"
    attendanceDate: string; // "2025-01-15"
    status: AttendanceStatus;
    totalStudents: number;
    recordedStudents: number;
}

export interface ListAttendanceSessionsParams {
    page?: number;
    limit?: number;
    search?: string;
    status?: AttendanceStatus | "";
    dateFrom?: string;
    dateTo?: string;
    grades?: string[];
    streams?: string[];
    subjects?: string[];
    teachers?: string[];
}

export interface ListAttendanceSessionsResponse {
    items: AttendanceSession[];
    total: number;
    page: number;
    limit: number;
}
```

### 5. API Service (`services/api.ts`)

```ts
import { api } from "@/lib/api/client";
import type {
    AttendanceSession,
    ListAttendanceSessionsParams,
    ListAttendanceSessionsResponse,
} from "../types/attendance";

export async function listAttendanceSessions(
    params: ListAttendanceSessionsParams = {},
): Promise<ListAttendanceSessionsResponse> {
    const searchParams = new URLSearchParams();
    Object.entries(params).forEach(([key, value]) => {
        if (value === undefined || value === null || value === "") return;
        if (Array.isArray(value)) {
            value.forEach((v) => searchParams.append(key, v));
        } else {
            searchParams.set(key, String(value));
        }
    });
    return api.get<ListAttendanceSessionsResponse>(
        `/api/attendance/sessions?${searchParams.toString()}`,
    );
}
```

### 6. Hook (`hooks/use-attendance-sessions.ts`)

```ts
import { useQuery } from "@tanstack/react-query";
import { listAttendanceSessions } from "../services/api";
import type {
    AttendanceSession,
    ListAttendanceSessionsParams,
} from "../types/attendance";

export const attendanceKeys = {
    sessions: (params: ListAttendanceSessionsParams) =>
        ["attendance", "sessions", params] as const,
};

export function useAttendanceSessions(
    params: ListAttendanceSessionsParams = {},
) {
    return useQuery({
        queryKey: attendanceKeys.sessions(params),
        queryFn: () => listAttendanceSessions(params),
        staleTime: 30_000,
    });
}
```

### 7. Status Badge Component (`components/status-badge.tsx`)

```tsx
"use client";
import { cn } from "@/lib/utils";
import type { AttendanceStatus } from "../types/attendance";

const statusConfig: Record<
    AttendanceStatus,
    { label: string; className: string }
> = {
    SUBMITTED: {
        label: "Submitted",
        className: "bg-emerald-100 text-emerald-800",
    },
    IN_PROGRESS: {
        label: "In Progress",
        className: "bg-amber-100 text-amber-800",
    },
    MISSED: { label: "Missed", className: "bg-rose-100 text-rose-800" },
};

export function StatusBadge({ status }: { status: AttendanceStatus }) {
    const { label, className } = statusConfig[status];
    return (
        <span
            className={cn(
                "inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium",
                className,
            )}
        >
            {label}
        </span>
    );
}
```

### 8. Data Table Component (`components/attendance-table.tsx`)

```tsx
"use client";
import Link from "next/link";
import { DataTable } from "@/components/shared/data-table";
import { listAttendanceSessions } from "../services/api";
import { useAttendanceSessions } from "../hooks/use-attendance-sessions";
import { StatusBadge } from "./status-badge";
import type {
    AttendanceSession,
    ListAttendanceSessionsParams,
} from "../types/attendance";
import { useGrades } from "@/features/grades/hooks/use-grades";
import { useStreams } from "@/features/streams/hooks/use-streams";
import { useMemo } from "react";

function listWithFilters(params: ListAttendanceSessionsParams) {
    return listAttendanceSessions(params);
}

export function AttendanceTable() {
    const { data: grades = [] } = useGrades();
    const { data: streams = [] } = useStreams();

    const gradeOptions = useMemo(
        () =>
            grades.map((g) => ({
                label: g.local_label,
                value: g.local_label,
                id: g.id,
            })),
        [grades],
    );
    const streamOptions = useMemo(
        () => streams.map((s) => ({ label: s.name, value: s.name, id: s.id })),
        [streams],
    );

    const filterGroups = useMemo(
        () => [
            {
                id: "status",
                label: "Status",
                items: [
                    {
                        id: "status-filter",
                        label: "Status",
                        type: "sub_menu_multi" as const,
                        submenu: [
                            {
                                id: "SUBMITTED",
                                label: "Submitted",
                                value: "SUBMITTED",
                            },
                            {
                                id: "IN_PROGRESS",
                                label: "In Progress",
                                value: "IN_PROGRESS",
                            },
                            { id: "MISSED", label: "Missed", value: "MISSED" },
                        ],
                    },
                ],
            },
            {
                id: "grade",
                label: "Grade",
                items: [
                    {
                        id: "grade-filter",
                        label: "Grade",
                        type: "sub_menu_multi" as const,
                        submenu:
                            gradeOptions.length > 0
                                ? gradeOptions.map((opt) => ({
                                      id: opt.id,
                                      label: opt.label,
                                      value: opt.value,
                                  }))
                                : [
                                      {
                                          id: "empty",
                                          label: "No grades",
                                          value: "",
                                      },
                                  ],
                    },
                ],
            },
            {
                id: "stream",
                label: "Stream",
                items: [
                    {
                        id: "stream-filter",
                        label: "Stream",
                        type: "sub_menu_multi" as const,
                        submenu:
                            streamOptions.length > 0
                                ? streamOptions.map((opt) => ({
                                      id: opt.id,
                                      label: opt.label,
                                      value: opt.value,
                                  }))
                                : [
                                      {
                                          id: "empty",
                                          label: "No streams",
                                          value: "",
                                      },
                                  ],
                    },
                ],
            },
        ],
        [gradeOptions, streamOptions],
    );

    const columns = useMemo(
        () => [
            {
                id: "className",
                header: "Class",
                cell: (row: AttendanceSession) => (
                    <Link
                        href={`/classes/${row.classId}`}
                        className="underline underline-offset-4 hover:no-underline"
                    >
                        {row.className}
                    </Link>
                ),
                width: "2fr",
            },
            {
                id: "teacherName",
                header: "Teacher",
                cell: (row: AttendanceSession) => (
                    <Link
                        href={`/teachers/${row.teacherId}`}
                        className="underline underline-offset-4 hover:no-underline"
                    >
                        {row.teacherName}
                    </Link>
                ),
                width: "1.5fr",
            },
            {
                id: "subject",
                header: "Subject",
                cell: (row: AttendanceSession) => row.subject,
                width: "1.5fr",
            },
            {
                id: "dateTime",
                header: "Date & Time",
                cell: (row: AttendanceSession) => (
                    <div className="space-y-0.5">
                        <span>{row.attendanceDate}</span>
                        <span className="text-muted-foreground text-[0.625rem]">
                            {row.timeSlotName} · {row.startTime}–{row.endTime}
                        </span>
                    </div>
                ),
                width: "1.5fr",
            },
            {
                id: "status",
                header: "Status",
                cell: (row: AttendanceSession) => (
                    <StatusBadge status={row.status} />
                ),
                width: "1fr",
                align: "center",
            },
            {
                id: "progress",
                header: "Progress",
                cell: (row: AttendanceSession) => (
                    <div className="flex items-center gap-2">
                        <div className="flex-1 max-w-32 h-1.5 bg-muted rounded-full overflow-hidden">
                            <div
                                className="h-full bg-primary transition-all"
                                style={{
                                    width: `${row.totalStudents > 0 ? (row.recordedStudents / row.totalStudents) * 100 : 0}%`,
                                }}
                            />
                        </div>
                        <span className="text-xs text-muted-foreground w-20 text-right">
                            {row.recordedStudents}/{row.totalStudents}
                        </span>
                    </div>
                ),
                width: "1.5fr",
            },
        ],
        [],
    );

    return (
        <DataTable<
            AttendanceSession,
            ListAttendanceSessionsParams,
            {
                items: AttendanceSession[];
                total: number;
                page: number;
                limit: number;
            }
        >
            queryKey={["attendance", "sessions"]}
            queryFn={listWithFilters}
            getRowId={(row) => row.slotId + "-" + row.attendanceDate}
            columns={columns}
            isSearchable
            searchPlaceholder="Search class, teacher, subject…"
            filterGroups={filterGroups}
            pageSize={50}
            height={500}
        />
    );
}
```

### 9. Page Route

**File:** `src/app/(dashboard)/attendance/page.tsx`

```tsx
import { AttendanceTable } from "@/features/attendance/components/attendance-table";

export default function AttendancePage() {
    return (
        <div className="space-y-6">
            <h1 className="text-2xl font-semibold text-foreground">
                Attendance Sessions
            </h1>
            <AttendanceTable />
        </div>
    );
}
```

### 10. Navigation

Add to sidebar navigation (likely in `src/components/app-sidebar.tsx` or similar):

```tsx
{ href: "/attendance", label: "Attendance", icon: CalendarCheck },
```

---

## Database Considerations

### Indexes Needed

The attendance service will query across multiple tables. Ensure these indexes exist:

- `timetable_attendance (class_timetable_slot_id, attendance_date)` — already exists
- `timetable_attendance (attendance_date)` — already exists
- Consider composite index on `class_timetable_slots (school_id, academic_term_id, day_of_week)` for slot listing

### Query Optimization

The `ListAttendanceSessions` service method will likely need a complex CTE:

1. Get all active class timetable slots for the school (with class, subject, teacher, time slot)
2. Generate date series for each slot's day_of_week within the academic term or filter range
3. Left join attendance records
4. Aggregate student counts and recorded counts per (slot, date)
5. Compute status
6. Apply filters and pagination

---

## Implementation Order

1. **Backend Service** — Add `ListAttendanceSessions` to `AttendanceService` interface and implementation
2. **Backend Handler** — Add `ListAttendanceSessions` endpoint to `AttendanceHandler`
3. **Router** — Register `GET /api/attendance/sessions`
4. **Frontend Types** — Create `types/attendance.ts`
5. **Frontend API** — Create `services/api.ts`
6. **Frontend Hook** — Create `hooks/use-attendance-sessions.ts`
7. **Frontend Components** — Create `status-badge.tsx` and `attendance-table.tsx`
8. **Frontend Page** — Create `src/app/(dashboard)/attendance/page.tsx`
9. **Navigation** — Add to sidebar
10. **Testing** — Verify with sample data

---

## Notes

- Follow pure-shadcn guidelines: no custom colors, flat layout, semantic CSS variables
- Use DataTable component (TanStack Virtual) for performance
- No back buttons or headers in page — rely on app shell
- Status computation must handle edge cases: classes with 0 students, future dates, holidays
- Consider adding "UPCOMING" status for future dates with 0 records if UX requires
- Pagination: default 50, max 200 per backend convention
- Filters: grade, stream, status, date range, search (class/teacher/subject)
