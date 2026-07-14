/**
 * ParentAttendanceSummary — parent-facing view of a child's attendance.
 *
 * Shows attendance percentage from term summaries and a list of recent periods.
 */

"use client";

import { Skeleton } from "@/components/ui/skeleton";
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";

import { useChildAttendanceSummary } from "../hooks/use-attendance";

interface ParentAttendanceSummaryProps {
    studentId: string;
    termId: string;
}

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
            <div className="text-muted-foreground flex items-center justify-center py-16">
                <p>Attendance data will appear once the term is underway.</p>
            </div>
        );
    }

    return (
        <div className="space-y-6">
            {/* Summary card */}
            <div className="rounded-lg border p-6">
                <div className="flex items-baseline gap-2">
                    <span className="text-4xl font-bold">
                        {data.attendance_percentage.toFixed(1)}%
                    </span>
                    <span className="text-muted-foreground">attendance</span>
                </div>
            </div>

            {/* Recent periods */}
            <div>
                <h3 className="mb-3 text-lg font-semibold">Recent Periods</h3>
                {data.recent_periods.length === 0 ? (
                    <p className="text-muted-foreground text-sm">No recent attendance records.</p>
                ) : (
                    <Table>
                        <TableHeader>
                            <TableRow>
                                <TableHead>Date</TableHead>
                                <TableHead>Subject</TableHead>
                                <TableHead className="w-28">Status</TableHead>
                            </TableRow>
                        </TableHeader>
                        <TableBody>
                            {data.recent_periods.map((period, idx) => (
                                <TableRow key={`${period.date}-${period.subject}-${idx}`}>
                                    <TableCell>{period.date}</TableCell>
                                    <TableCell>{period.subject}</TableCell>
                                    <TableCell>
                                        <Badge
                                            variant={
                                                period.status === "PRESENT"
                                                    ? "default"
                                                    : period.status === "ABSENT"
                                                      ? "destructive"
                                                      : "secondary"
                                            }
                                        >
                                            {period.status}
                                        </Badge>
                                    </TableCell>
                                </TableRow>
                            ))}
                        </TableBody>
                    </Table>
                )}
            </div>
        </div>
    );
}
