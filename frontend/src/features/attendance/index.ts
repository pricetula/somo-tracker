/**
 * Attendance feature — public API barrel.
 *
 * External code must import ONLY from this file — never from internal paths.
 */

// Components
export { TeacherAttendanceRoster } from "./components/teacher-attendance-roster";
export { AdminAttendanceDashboard } from "./components/admin-attendance-dashboard";
export { ParentAttendanceSummary } from "./components/parent-attendance-summary";
export { StudentHistoryView } from "./components/student-history-view";

// Hooks
export {
    useSlotRoster,
    useBulkMarkAttendance,
    useAdminDashboard,
    useStudentHistory,
    useUpdateAttendanceRecord,
    useChildAttendanceSummary,
    useComputeAttendanceSummaries,
    attendanceKeys,
} from "./hooks/use-attendance";

// Types
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
} from "./types";
