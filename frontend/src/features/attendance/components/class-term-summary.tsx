"use client";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import {
    useClassTermSummary,
    useRefreshSummaries,
} from "@/features/attendance/hooks/use-attendance";
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from "recharts";
import { AttendanceTermSummary } from "@/features/attendance/types";

interface ClassTermSummaryProps {
    classId: string;
    termId?: string;
}

function getBarChartData(summaries: AttendanceTermSummary[]) {
    return summaries.map((s) => ({
        name: s.learning_area_name || "Unknown",
        present: s.periods_present,
        absent: s.periods_absent,
        late: s.periods_late,
        excused: s.periods_excused,
        percentage: s.attendance_percentage,
    }));
}

function formatNumber(num: number): string {
    return new Intl.NumberFormat().format(num);
}

export function ClassTermSummary({ classId, termId }: ClassTermSummaryProps) {
    const { data: summaries, isLoading, isError } = useClassTermSummary(classId, termId);
    const refresh = useRefreshSummaries();

    const chartData = getBarChartData(summaries || []);
    const totalPresent = summaries?.reduce((sum, s) => sum + s.periods_present, 0) || 0;
    const totalAbsent = summaries?.reduce((sum, s) => sum + s.periods_absent, 0) || 0;
    const totalLate = summaries?.reduce((sum, s) => sum + s.periods_late, 0) || 0;
    const totalExcused = summaries?.reduce((sum, s) => sum + s.periods_excused, 0) || 0;
    const totalPeriods = totalPresent + totalAbsent + totalLate + totalExcused;
    const overallPercentage = totalPeriods > 0 ? (totalPresent / totalPeriods) * 100 : 0;

    if (isLoading) {
        return (
            <Card>
                <CardHeader>
                    <Skeleton className="h-6 w-48" />
                    <Skeleton className="h-4 w-64" />
                </CardHeader>
                <CardContent>
                    <div className="space-y-3">
                        {Array.from({ length: 3 }).map((_, i) => (
                            <Skeleton key={i} className="h-20 w-full" />
                        ))}
                    </div>
                </CardContent>
            </Card>
        );
    }

    if (isError) {
        return (
            <Card>
                <CardContent className="pt-6">
                    <div className="bg-destructive/10 text-destructive rounded-md p-4 text-sm">
                        Failed to load class term summary. Please try again.
                    </div>
                </CardContent>
            </Card>
        );
    }

    if (!summaries?.length) {
        return (
            <Card>
                <CardContent className="pt-6">
                    <div className="text-muted-foreground py-8 text-center text-sm">
                        No attendance summaries available for this class and term.
                    </div>
                </CardContent>
            </Card>
        );
    }

    return (
        <div className="space-y-6">
            <Card>
                <CardHeader>
                    <CardTitle>Class Term Attendance Summary</CardTitle>
                    <CardDescription>
                        Class ID: {classId}
                        {termId && ` • Term: ${termId}`}
                    </CardDescription>
                </CardHeader>
                <CardContent>
                    <div className="grid gap-6 md:grid-cols-4">
                        <div className="space-y-2">
                            <p className="text-muted-foreground text-sm font-medium">
                                Overall Attendance
                            </p>
                            <p className="text-3xl font-bold text-emerald-600">
                                {overallPercentage.toFixed(1)}%
                            </p>
                        </div>
                        <div className="space-y-2">
                            <p className="text-muted-foreground text-sm font-medium">
                                Total Present
                            </p>
                            <p className="text-3xl font-bold text-emerald-600">
                                {formatNumber(totalPresent)}
                            </p>
                        </div>
                        <div className="space-y-2">
                            <p className="text-muted-foreground text-sm font-medium">
                                Total Absent
                            </p>
                            <p className="text-destructive text-3xl font-bold">
                                {formatNumber(totalAbsent)}
                            </p>
                        </div>
                        <div className="space-y-2">
                            <p className="text-muted-foreground text-sm font-medium">
                                Late / Excused
                            </p>
                            <p className="text-3xl font-bold text-amber-600">
                                {formatNumber(totalLate + totalExcused)}
                            </p>
                        </div>
                    </div>
                </CardContent>
            </Card>

            {chartData.length > 0 && (
                <Card>
                    <CardHeader>
                        <CardTitle>Learning Area Breakdown</CardTitle>
                        <CardDescription>Attendance periods by learning area</CardDescription>
                    </CardHeader>
                    <CardContent>
                        <div className="h-80 w-full">
                            <ResponsiveContainer width="100%" height="100%">
                                <BarChart data={chartData}>
                                    <CartesianGrid strokeDasharray="3 3" />
                                    <XAxis dataKey="name" />
                                    <YAxis />
                                    <Tooltip />
                                    <Bar dataKey="present" fill="#22c55e" name="Present" />
                                    <Bar dataKey="absent" fill="#ef4444" name="Absent" />
                                    <Bar dataKey="late" fill="#f59e0b" name="Late" />
                                    <Bar dataKey="excused" fill="#8b5cf6" name="Excused" />
                                </BarChart>
                            </ResponsiveContainer>
                        </div>
                    </CardContent>
                </Card>
            )}

            {termId && (
                <div className="flex justify-end">
                    <Button
                        variant="outline"
                        size="sm"
                        onClick={() => refresh.mutate(termId)}
                        disabled={refresh.isPending}
                    >
                        {refresh.isPending ? "Refreshing..." : "Refresh Summary"}
                    </Button>
                </div>
            )}

            <Card>
                <CardHeader>
                    <CardTitle>Student Summaries</CardTitle>
                    <CardDescription>
                        Individual student attendance per learning area
                    </CardDescription>
                </CardHeader>
                <CardContent>
                    <div className="space-y-3">
                        {summaries.map((summary) => (
                            <div
                                key={summary.id}
                                className="flex items-center justify-between rounded-md border p-4"
                            >
                                <div className="space-y-1">
                                    <p className="font-medium">Student: {summary.student_id}</p>
                                    <p className="text-muted-foreground text-sm">
                                        {summary.learning_area_name || "Unknown"}
                                    </p>
                                </div>
                                <div className="space-y-1 text-right">
                                    <p className="font-medium text-emerald-600">
                                        {summary.attendance_percentage.toFixed(1)}%
                                    </p>
                                    <p className="text-muted-foreground text-xs">
                                        {formatNumber(summary.periods_present)} /{" "}
                                        {formatNumber(summary.periods_total)}
                                    </p>
                                </div>
                            </div>
                        ))}
                    </div>
                </CardContent>
            </Card>
        </div>
    );
}
