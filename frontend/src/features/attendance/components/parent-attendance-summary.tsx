/**
 * ParentAttendanceSummary — parent-facing view of a child's attendance.
 *
 * Shows attendance percentage from term summaries and a list of recent periods
 * using the shared DataTable component.
 */

"use client";

import Link from "next/link";
import { CalendarX, ClipboardList, Flag } from "lucide-react";

import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
import { DataTable } from "@/components/shared/data-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";

import { AttendanceEmptyState } from "./attendance-empty-state";
import { useChildAttendanceSummary } from "../hooks/use-attendance";
import { attendanceBadgeProps, attendanceStatusLabel } from "../types";
import type { AttendanceStatus, StudentAttendanceRecord } from "../types";

interface ParentAttendanceSummaryProps {
    studentId: string;
    termId: string;
}

// ─── Columns for recent periods ───────────────────────────────────────────

const recentPeriodsColumns: DataTableColumn<StudentAttendanceRecord>[] = [
    {
        id: "date",
        header: "Date",
        cell: (row) => <span>{row.date}</span>,
    },
    {
        id: "subject",
        header: "Subject",
        width: "minmax(120px, 1fr)",
        cell: (row) => <span>{row.subject}</span>,
    },
    {
        id: "status",
        header: "Status",
        width: "100px",
        cell: (row) => (
            <Badge
                variant={attendanceBadgeProps(row.status as AttendanceStatus).variant}
                className={attendanceBadgeProps(row.status as AttendanceStatus).className}
            >
                {attendanceStatusLabel(row.status as AttendanceStatus)}
            </Badge>
        ),
    },
];

// ─── Component ────────────────────────────────────────────────────────────

export function ParentAttendanceSummary({ studentId, termId }: ParentAttendanceSummaryProps) {
    const { data, isLoading, isError } = useChildAttendanceSummary(studentId, termId);

    if (isLoading) {
        return (
            <div className="space-y-4">
                <Skeleton className="h-24 w-full rounded-lg" />
                <Skeleton className="h-8 w-48" />
                <Skeleton className="h-8 w-full" />
                <Skeleton className="h-8 w-full" />
            </div>
        );
    }

    if (isError) {
        return (
            <div className="border-destructive/50 text-destructive rounded-md border p-4">
                Failed to load attendance data.
            </div>
        );
    }

    if (!data) {
        return (
            <AttendanceEmptyState
                icon={ClipboardList}
                title="No attendance data yet"
                description="Attendance records will appear here once the term is underway and marks have been recorded."
            />
        );
    }

    const recentPeriods = data.recent_periods ?? [];
    const hasData = recentPeriods.length > 0 || data.attendance_percentage > 0;

    // Compute status counts for context
    const statusCounts: Record<string, number> = {};
    for (const p of recentPeriods) {
        statusCounts[p.status] = (statusCounts[p.status] || 0) + 1;
    }

    if (!hasData) {
        return (
            <AttendanceEmptyState
                icon={ClipboardList}
                title="No attendance data yet"
                description="Attendance records will appear here once the term is underway and marks have been recorded."
            />
        );
    }

    return (
        <div className="space-y-6">
            {/* Summary card */}
            <div className="space-y-4 rounded-lg border p-6">
                <div className="flex items-baseline gap-2">
                    <span className="text-4xl font-bold">
                        {data.attendance_percentage.toFixed(1)}%
                    </span>
                    <span className="text-muted-foreground">attendance</span>
                </div>
                <Progress value={data.attendance_percentage} className="h-2" />
                {/* Status breakdown */}
                {statusCounts.PRESENT > 0 && (
                    <div className="flex flex-wrap gap-2">
                        {Object.entries(statusCounts)
                            .filter(([, c]) => c > 0)
                            .map(([status, count]) => (
                                <Badge
                                    key={status}
                                    variant={
                                        attendanceBadgeProps(status as AttendanceStatus).variant
                                    }
                                    className={
                                        attendanceBadgeProps(status as AttendanceStatus).className
                                    }
                                >
                                    {attendanceStatusLabel(status as AttendanceStatus)}: {count}
                                </Badge>
                            ))}
                    </div>
                )}
            </div>

            {/* Recent periods */}
            <div>
                <h3 className="mb-3 text-lg font-semibold">Recent Periods</h3>
                {recentPeriods.length === 0 ? (
                    <div className="text-muted-foreground flex flex-col items-center gap-2 py-8 text-center">
                        <CalendarX className="size-6" />
                        <p className="">No recent attendance records in the last 30 days.</p>
                    </div>
                ) : (
                    <DataTable
                        queryKey={["attendance", "parent-summary", studentId, termId]}
                        queryFn={() =>
                            Promise.resolve({ items: recentPeriods, total: recentPeriods.length })
                        }
                        columns={recentPeriodsColumns}
                        getRowId={(row, idx) => `${row.date}-${row.subject}-${idx}`}
                        height={Math.min(recentPeriods.length * 44 + 50, 400)}
                        pageSize={100}
                        emptyState="No recent attendance records."
                        noResultsState="No records match your search."
                    />
                )}
            </div>

            {/* Behaviour cross-link */}
            <div className="rounded-lg border p-4">
                <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                        <Flag className="text-muted-foreground h-4 w-4" />
                        <span className="font-medium">Behaviour Notes</span>
                    </div>
                    <Button variant="outline" size="sm" asChild>
                        <Link href={`/behavior`}>View behaviour notes</Link>
                    </Button>
                </div>
                <p className="text-muted-foreground mt-1 text-xs">
                    Behaviour notes logged by teachers during the term are available in the
                    behaviour section alongside attendance records.
                </p>
            </div>
        </div>
    );
}
