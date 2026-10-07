import { AttendanceTable } from "@/features/attendance/components/attendance-table";

export default function AttendancePage() {
    return (
        <div className="space-y-6">
            <h1 className="text-foreground text-2xl font-semibold">Attendance Sessions</h1>
            <AttendanceTable />
        </div>
    );
}
