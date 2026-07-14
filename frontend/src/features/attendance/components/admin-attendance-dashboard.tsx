/**
 * AdminAttendanceDashboard — school-wide attendance completion view.
 *
 * Shows per-class period completion status for a given date.
 * Clicking a row navigates to that class's detail.
 */

"use client";

import { useMemo } from "react";
import { useRouter } from "next/navigation";
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
import { Button } from "@/components/ui/button";

import { useAdminDashboard } from "../hooks/use-attendance";

interface AdminAttendanceDashboardProps {
    date?: string;
}

export function AdminAttendanceDashboard({ date }: AdminAttendanceDashboardProps) {
    const { data, isLoading, isError } = useAdminDashboard(date);
    const router = useRouter();

    const completionSummary = useMemo(() => {
        if (!data) return { total: 0, complete: 0, incomplete: 0 };
        const complete = data.classes.filter((c) => c.is_complete).length;
        return {
            total: data.classes.length,
            complete,
            incomplete: data.classes.length - complete,
        };
    }, [data]);

    if (isLoading) {
        return (
            <div className="space-y-3">
                <Skeleton className="h-8 w-48" />
                <Skeleton className="h-8 w-full" />
                <Skeleton className="h-8 w-full" />
                <Skeleton className="h-8 w-full" />
            </div>
        );
    }

    if (isError) {
        return (
            <div className="border-destructive/50 text-destructive rounded-md border p-4">
                Failed to load attendance dashboard.
            </div>
        );
    }

    if (!data || data.classes.length === 0) {
        return (
            <div className="text-muted-foreground flex items-center justify-center py-16">
                <p>No classes scheduled for {date ?? "today"}.</p>
            </div>
        );
    }

    return (
        <div className="space-y-4">
            <div className="flex items-center gap-4">
                <h2 className="text-xl font-semibold">Attendance — {data.date}</h2>
                <Badge variant="outline">
                    {completionSummary.complete} of {completionSummary.total} complete
                </Badge>
                {completionSummary.incomplete > 0 && (
                    <Badge variant="secondary" className="bg-amber-100 text-amber-800">
                        {completionSummary.incomplete} incomplete
                    </Badge>
                )}
            </div>

            <Table>
                <TableHeader>
                    <TableRow>
                        <TableHead>Period</TableHead>
                        <TableHead>Class</TableHead>
                        <TableHead className="w-32 text-right">Marked</TableHead>
                        <TableHead className="w-28">Status</TableHead>
                        <TableHead className="w-20" />
                    </TableRow>
                </TableHeader>
                <TableBody>
                    {data.classes.map((cls) => (
                        <TableRow key={cls.slot_id}>
                            <TableCell className="text-muted-foreground">
                                {cls.period_name}
                            </TableCell>
                            <TableCell className="font-medium">{cls.class_name}</TableCell>
                            <TableCell className="text-right">
                                {cls.marked_slots}/{cls.total_slots}
                            </TableCell>
                            <TableCell>
                                <Badge
                                    variant={cls.is_complete ? "default" : "secondary"}
                                    className={
                                        cls.is_complete
                                            ? ""
                                            : "bg-amber-100 text-amber-800 hover:bg-amber-100"
                                    }
                                >
                                    {cls.is_complete ? "Complete" : "Incomplete"}
                                </Badge>
                            </TableCell>
                            <TableCell>
                                <Button
                                    variant="ghost"
                                    size="sm"
                                    onClick={() =>
                                        router.push(
                                            `/attendance/register?slot_id=${cls.slot_id}&date=${data.date}`
                                        )
                                    }
                                >
                                    View
                                </Button>
                            </TableCell>
                        </TableRow>
                    ))}
                </TableBody>
            </Table>
        </div>
    );
}
