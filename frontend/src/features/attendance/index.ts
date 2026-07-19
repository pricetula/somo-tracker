/**
 * Attendance feature — public API barrel.
 *
 * External code must import ONLY from this file — never from internal paths.
 */

// Components
export { TeacherAttendanceRoster } from "./components/teacher-attendance-roster";
export { TeacherAttendanceLanding } from "./components/teacher-attendance-landing";
export { TeacherHistoryView } from "./components/teacher-history-view";
export { ParentAttendanceSummary } from "./components/parent-attendance-summary";
export { ParentAttendanceLanding } from "./components/parent-attendance-landing";
export { AdminAttendanceDashboard } from "./components/admin-attendance-dashboard";
export { StudentHistoryView } from "./components/student-history-view";
export { AttendanceEmptyState } from "./components/attendance-empty-state";

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
