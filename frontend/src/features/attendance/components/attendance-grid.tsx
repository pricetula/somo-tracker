/**
 * AttendanceGrid — mark and view attendance records for a class on a date.
 *
 * TEACHER: mark attendance for their class periods
 * SCHOOL_ADMIN: view and manage all attendance records
 */
"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import {
    useRecordsBySlot,
    useBatchMarkAttendance,
} from "@/features/attendance/hooks/use-attendance";
import type { AttendanceStatus, StudentAttendanceMark } from "@/features/attendance/types";

// ─── Status badge helper ──────────────────────────────────────────────────

const statusOptions: { value: AttendanceStatus; label: string }[] = [
    { value: "PRESENT", label: "Present" },
    { value: "ABSENT", label: "Absent" },
    { value: "LATE", label: "Late" },
    { value: "EXCUSED", label: "Excused" },
];

// ─── Component ────────────────────────────────────────────────────────────

interface AttendanceGridProps {
    timetableSlotId: string;
    date: string;
    termId?: string;
}

export function AttendanceGrid({ timetableSlotId, date, termId }: AttendanceGridProps) {
    const { data: records, isLoading, isError } = useRecordsBySlot(timetableSlotId, date);
    const batchMark = useBatchMarkAttendance(termId);

    const [localStatuses, setLocalStatuses] = useState<Record<string, AttendanceStatus>>({});

    if (isLoading) {
        return (
            <div className="space-y-3">
                {Array.from({ length: 5 }).map((_, i) => (
                    <Skeleton key={i} className="h-10 w-full" />
                ))}
            </div>
        );
    }

    if (isError) {
        return (
            <div className="bg-destructive/10 text-destructive rounded-md p-4 text-sm">
                Failed to load attendance records. Please try again.
            </div>
        );
    }

    const items = records ?? [];

    if (items.length === 0) {
        return (
            <div className="text-muted-foreground py-8 text-center text-sm">
                No students found for this slot and date.
            </div>
        );
    }

    const getStatus = (studentId: string, currentStatus: AttendanceStatus): AttendanceStatus => {
        return localStatuses[studentId] ?? currentStatus;
    };

    const setStatus = (studentId: string, status: AttendanceStatus) => {
        setLocalStatuses((prev) => ({ ...prev, [studentId]: status }));
    };

    const handleSubmit = () => {
        const marks: StudentAttendanceMark[] = items.map((record) => ({
            student_id: record.student_id,
            status: getStatus(record.student_id, record.status),
        }));

        batchMark.mutate({
            date,
            timetable_slot_id: timetableSlotId,
            records: marks,
        });
    };

    const hasChanges = items.some(
        (record) =>
            localStatuses[record.student_id] !== undefined &&
            localStatuses[record.student_id] !== record.status
    );

    return (
        <div className="space-y-4">
            <div className="space-y-1">
                {items.map((record) => {
                    const currentStatus = getStatus(record.student_id, record.status);
                    return (
                        <div
                            key={record.id}
                            className="bg-muted/30 flex items-center justify-between rounded-md px-4 py-2"
                        >
                            <span className="text-foreground font-medium">
                                {record.student_full_name}
                            </span>
                            <div className="flex gap-1">
                                {statusOptions.map((opt) => (
                                    <Button
                                        key={opt.value}
                                        variant={
                                            currentStatus === opt.value ? "default" : "outline"
                                        }
                                        size="sm"
                                        onClick={() => setStatus(record.student_id, opt.value)}
                                        className="h-7 text-xs"
                                    >
                                        {opt.label}
                                    </Button>
                                ))}
                            </div>
                        </div>
                    );
                })}
            </div>

            {hasChanges && (
                <Button onClick={handleSubmit} disabled={batchMark.isPending}>
                    {batchMark.isPending ? "Saving..." : "Save Attendance"}
                </Button>
            )}
        </div>
    );
}
