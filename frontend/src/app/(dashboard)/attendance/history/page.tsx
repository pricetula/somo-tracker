/**
 * Attendance history page — teacher's past periods.
 * Shows periods the teacher has taught with same-day edit capability.
 */

export default async function AttendanceHistoryPage() {
    const { TeacherHistoryView } =
        await import("@/features/attendance/components/teacher-history-view");
    return <TeacherHistoryView />;
}
