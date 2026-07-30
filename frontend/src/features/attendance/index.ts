/**
 * Attendance feature — public API barrel.
 */

export { AttendanceCalendar } from "./components/attendance-calendar";
export type { DaySummary } from "./components/attendance-calendar";
export { SessionList } from "./components/session-list";
export { AttendanceGrid } from "./components/attendance-grid";
export { AttendanceTimeline } from "./components/attendance-timeline";
export { AttendanceMarkPage } from "./components/attendance-mark-page";
export { SummaryTable } from "./components/summary-table";

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
    CreateSessionPayload,
    UpdateSessionPayload,
    StudentAttendanceMark,
    BatchMarkPayload,
    BatchMarkResult,
    UpdateRecordPayload,
    CalendarDayStatus,
    CalendarStatusListResponse,
    DayStatus,
} from "./types";
