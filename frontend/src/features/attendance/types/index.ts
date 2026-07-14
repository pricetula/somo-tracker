/**
 * Attendance feature types.
 *
 * These mirror the backend API response shapes. The canonical definitions
 * live in src/lib/api/attendance.ts; this barrel re-exports them so feature
 * consumers can import from @/features/attendance/types.
 */

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
} from "@/lib/api/attendance";
