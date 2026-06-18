// ─── Types ─────────────────────────────────────────────────────────────────
export type {
    CbcTimetableSlot,
    CbcTimetableSlotCreatePayload,
    CbcTimetableSlotUpdatePayload,
    CbcSlotConflict,
    OperatingDay,
    LearningAreaOption,
    TeacherOption,
    DuplicateDayPayload,
    CopyFromClassPayload,
    BulkOperationResult,
    AttendanceStatus,
    CbcAttendancePeriod,
    CbcAttendanceLog,
    AttendanceStudentRow,
    AttendanceSlotColumn,
    OfflineAttendanceEntry,
    CbcLearningArea,
    CbcClassTeacher,
    UserBrief,
    ClassDetail,
} from "./types";

// ─── Timetable Components ─────────────────────────────────────────────────
export { CbcTimetablePage } from "./components/timetable/cbc-timetable-page";
export { CbcTimetableGrid } from "./components/timetable/cbc-timetable-grid";
export { CbcSlotBlock } from "./components/timetable/cbc-slot-block";
export { CbcSlotEditorSidePanel } from "./components/timetable/cbc-slot-editor-side-panel";
export { CbcBulkActions } from "./components/timetable/cbc-bulk-actions";

// ─── Attendance Components ────────────────────────────────────────────────
export { CbcAttendancePage } from "./components/attendance/cbc-attendance-page";
export { CbcAttendanceGrid } from "./components/attendance/cbc-attendance-grid";
export { CbcAttendanceStudentRow } from "./components/attendance/cbc-attendance-student-row";

// ─── Dashboard Components ─────────────────────────────────────────────────
export { MyAttendanceSection } from "./components/dashboard/my-attendance";

// ─── API Clients ──────────────────────────────────────────────────────────
export * as timetableApi from "./api/timetable";
export * as attendanceApi from "./api/attendance";

// ─── Hooks ────────────────────────────────────────────────────────────────
export {
    cbcTimetableKeys,
    useCbcTimetableSlots,
    useCreateCbcTimetableSlot,
    useUpdateCbcTimetableSlot,
    useDeleteCbcTimetableSlot,
    useSlotAttendanceCount,
    useSlotConflictCheck,
    useCbcLearningAreas,
    useCbcTeachers,
    useCbcClassTeachers,
    useRoomAutocomplete,
    useDuplicateDay,
    useCopyTimetableFromClass,
    useOperatingDays,
} from "./hooks/use-cbc-timetable";

export {
    cbcAttendanceKeys,
    useCbcAttendancePeriods,
    useCreateCbcAttendancePeriod,
    useCbcAttendanceLogs,
    useCbcClassStudents,
    useSaveAttendanceMark,
    useSaveAttendanceBatch,
    useTeacherTodaySlots,
} from "./hooks/use-cbc-attendance";
