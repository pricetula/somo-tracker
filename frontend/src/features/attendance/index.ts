/**
 * Attendance feature — public API barrel.
 *
 * Adds calendar status view exports.
 */

// Export existing components
export { AttendanceCalendar } from "./components/attendance-calendar";
export type { DaySummary } from "./components/attendance-calendar";
export { AttendanceGrid } from "./components/attendance-grid";
export { AttendanceTimeline } from "./components/attendance-timeline";
export { AttendanceMarkPage } from "./components/attendance-mark-page";
export { SummaryTable } from "./components/summary-table";

// Export new calendar status view
export { CalendarStatusView } from "./components/calendar-status-view";

// Export hooks
export {
    attendanceKeys,
    useAttendanceSessions,
    useAttendanceSession,
    useSessionsForClassDate,
    useAttendanceRecords,
    useRecordsBySlot,
    useRecordsByStudent,
    useRecordsByClassDate,
    useStudentTermSummary,
    useClassTermSummary,
    useCalendarStatus,
    useCreateSession,
    useUpdateSession,
    useBatchMarkAttendance,
    useUpdateRecord,
    useRefreshSummaries,
} from "./hooks/use-attendance";

export type {
    AttendanceStatus,
    SessionStatus,
    AttendanceRecord,
    AttendanceSession,
    SessionWithEnrichedData,
    RecordWithEnrichedData,
    AttendanceTermSummary,
    SessionListResponse,
    RecordListResponse,
    SummaryListResponse,
    RefreshSummaryResponse,
    CalendarStatusListResponse,
    DayStatus,
    // Payloads
    CreateSessionPayload,
    UpdateSessionPayload,
    StudentAttendanceMark,
    BatchMarkPayload,
    BatchMarkResult,
    UpdateRecordPayload,
    CalendarDayStatus,
} from "./types";
