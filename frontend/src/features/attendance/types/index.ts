/**
 * Attendance feature types.
 *
 * These mirror the backend API response shapes. The canonical definitions
 * live in src/lib/api/attendance.ts; this barrel re-exports them so feature
 * consumers can import from @/features/attendance/types.
 */

import type { AttendanceStatus as ApiAttendanceStatus } from "@/lib/api/attendance";

/**
 * Returns Badge variant and className for a given attendance status.
 * Reused across teacher, admin, and parent views for consistency.
 */
export function attendanceBadgeProps(status: ApiAttendanceStatus): {
    variant: "default" | "destructive" | "secondary" | "outline";
    className?: string;
} {
    switch (status) {
        case "PRESENT":
            return { variant: "default" };
        case "ABSENT":
            return { variant: "destructive" };
        case "LATE":
            return { variant: "outline", className: "text-amber-700 border-amber-300 bg-amber-50" };
        case "EXCUSED":
            return { variant: "outline", className: "text-sky-700 border-sky-300 bg-sky-50" };
    }
}

/** Human-readable label for each status. */
export function attendanceStatusLabel(status: ApiAttendanceStatus): string {
    const labels: Record<ApiAttendanceStatus, string> = {
        PRESENT: "Present",
        ABSENT: "Absent",
        LATE: "Late",
        EXCUSED: "Excused",
    };
    return labels[status];
}

export type {
    AttendanceStatus,
    RosterStudent,
    SlotRosterResponse,
    BulkAttendanceEntry,
    BulkAttendancePayload,
    StudentAttendanceRecord,
    ChildAttendanceSummary,
    CompletionStatus,
    AdminDashboardResponse,
    AttendanceRecord,
    AttendanceSession,
    SessionStatus,
    SkipSessionPayload,
} from "@/lib/api/attendance";
