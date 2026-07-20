/**
 * Attendance feature — public API barrel.
 */

export { SessionList } from "./components/session-list";
export { AttendanceGrid } from "./components/attendance-grid";
export { AttendanceTimeline } from "./components/attendance-timeline";
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
} from "./types";
