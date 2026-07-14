/**
 * TeacherHistoryView — shows past periods the teacher has taught.
 * Same-day edit allowed; after that, read-only.
 */

"use client";

import { Skeleton } from "@/components/ui/skeleton";

// TODO: Fetch teacher's past slots with attendance records.
// The real implementation should call GET /api/v1/timetable-slots?teacher_id=:id&date_before=today
// and list them with their marking status.

export function TeacherHistoryView() {
    return (
        <div className="space-y-6">
            <h1 className="text-2xl font-bold">Attendance History</h1>
            <p className="text-muted-foreground text-sm">
                Past periods you have taught. Same-day edits are allowed; older records are
                read-only. Contact your admin for corrections after the same-day window closes.
            </p>

            {/* TODO: Replace with actual list of past periods */}
            <div className="space-y-3">
                <Skeleton className="h-16 w-full rounded-lg" />
                <Skeleton className="h-16 w-full rounded-lg" />
                <Skeleton className="h-16 w-full rounded-lg" />
            </div>
        </div>
    );
}
