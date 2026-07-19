/**
 * Attendance feature types.
 *
 * These mirror the backend API response shapes. The canonical definitions
 * live in src/lib/api/attendance.ts; this barrel re-exports them so feature
 * consumers can import from @/features/attendance/types.
 */

import type { AttendanceStatus as ApiAttendanceStatus } from "@/lib/api/attendance";

/**
 * Returns Badge variant for a given attendance status.
 * Pure shadcn — no custom Tailwind colors, theme drives appearance.
 * Operational indicators use allowed exceptions:
 *   PRESENT → text-emerald-600   ABSENT → text-destructive
 *   LATE / EXCUSED → outline variant (theme-driven)
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
            return { variant: "outline" };
        case "EXCUSED":
            return { variant: "outline" };
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

/** Badge className for attendance statuses (used in StaticTable contexts). */
export function attendanceBadgeClass(status: ApiAttendanceStatus): string {
    switch (status) {
        case "PRESENT":
            return "text-emerald-600";
        case "ABSENT":
            return "text-destructive";
        default:
            return "text-muted-foreground";
    }
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
