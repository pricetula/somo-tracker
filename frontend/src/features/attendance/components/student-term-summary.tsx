"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import {
    useStudentTermSummary,
    useRefreshSummaries,
} from "@/features/attendance/hooks/use-attendance";
import { PieChart, Pie, Cell, ResponsiveContainer, Legend, Tooltip } from "recharts";
import { AttendanceTermSummary } from "@/features/attendance/types";

interface StudentTermSummaryProps {
    studentId: string;
    termId?: string;
}

// Colors for the chart
const CHART_COLORS = [
    "#ef4444", // red-500
    "#f97316", // orange-500
    "#eab308", // yellow-500
    "#84cc16", // green-500
    "#22c55e", // green-400
    "#06b6d4", // cyan-500
    "#3b82f6", // blue-500
    "#8b5cf6", // violet-500
    "#ec4899", // pink-500
];

function getChartData(summary: AttendanceTermSummary[]) {
    // Aggregate all learning areas for the student
    const learningAreas = summary.map((s) => s.learning_area_name || "Unknown");
    const uniqueAreas = Array.from(new Set(learningAreas));

    return uniqueAreas.map((area, index) => {
        const areasData = summary.filter((s) => s.learning_area_name === area);
        const totalPeriods = areasData.reduce((sum, s) => sum + s.periods_total, 0);
        const presentPeriods = areasData.reduce((sum, s) => sum + s.periods_present, 0);
        const percentage = totalPeriods > 0 ? (presentPeriods / totalPeriods) * 100 : 0;

        return {
            name: area,
            value: percentage,
            present: presentPeriods,
            total: totalPeriods,
            color: CHART_COLORS[index % CHART_COLORS.length],
        };
    });
}

function formatNumber(num: number): string {
    return new Intl.NumberFormat().format(num);
}

export function StudentTermSummary({ studentId, termId }: StudentTermSummaryProps) {
    const { data: summaries, isLoading, isError } = useStudentTermSummary(studentId, termId);
    const refresh = useRefreshSummaries();

    const [selectedLearningArea, setSelectedLearningArea] = useState<string | null>(null);

    const filteredSummaries = selectedLearningArea
        ? summaries?.filter((s) => s.learning_area_name === selectedLearningArea) || []
        : summaries || [];

    const chartData = getChartData(filteredSummaries);
    const totalPeriods = filteredSummaries.reduce((sum, s) => sum + s.periods_total, 0);
    const totalPresent = filteredSummaries.reduce((sum, s) => sum + s.periods_present, 0);
    const overallPercentage = totalPeriods > 0 ? (totalPresent / totalPeriods) * 100 : 0;

    const uniqueLearningAreas = Array.from(
        new Set(summaries?.map((s) => s.learning_area_name).filter(Boolean) || [])
    );

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
                        Failed to load student term summary. Please try again.
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
                        No attendance summaries available for this student.
                    </div>
                </CardContent>
            </Card>
        );
    }

    return (
        <div className="space-y-6">
            <Card>
                <CardHeader>
                    <CardTitle>Student Attendance Summary</CardTitle>
                    <CardDescription>
                        Attendance overview for Student ID: {studentId}
                        {termId && ` • Term: ${termId}`}
                    </CardDescription>
                </CardHeader>
                <CardContent>
                    <div className="grid gap-6 md:grid-cols-3">
                        <div className="space-y-2">
                            <p className="text-muted-foreground text-sm font-medium">
                                Overall Attendance
                            </p>
                            <p className="text-3xl font-bold text-emerald-600">
                                {overallPercentage.toFixed(1)}%
                            </p>
                            <p className="text-muted-foreground text-sm">
                                {formatNumber(totalPresent)} / {formatNumber(totalPeriods)} periods
                            </p>
                        </div>

                        <div className="space-y-2">
                            <p className="text-muted-foreground text-sm font-medium">Present</p>
                            <p className="text-3xl font-bold text-emerald-600">
                                {formatNumber(totalPresent)}
                            </p>
                        </div>

                        <div className="space-y-2">
                            <p className="text-muted-foreground text-sm font-medium">
                                Absent + Late + Excused
                            </p>
                            <p className="text-destructive text-3xl font-bold">
                                {formatNumber(totalPeriods - totalPresent)}
                            </p>
                        </div>
                    </div>

                    {uniqueLearningAreas.length > 0 && (
                        <div className="mt-6">
                            <p className="text-muted-foreground mb-2 text-sm font-medium">
                                Filter by Learning Area
                            </p>
                            <div className="flex flex-wrap gap-2">
                                <Button
                                    variant={selectedLearningArea === null ? "default" : "outline"}
                                    size="sm"
                                    onClick={() => setSelectedLearningArea(null)}
                                >
                                    All Areas ({summaries.length})
                                </Button>
                                {uniqueLearningAreas.map((area) => {
                                    const count = summaries.filter(
                                        (s) => s.learning_area_name === area
                                    ).length;
                                    return (
                                        <Button
                                            key={area}
                                            variant={
                                                selectedLearningArea === area
                                                    ? "default"
                                                    : "outline"
                                            }
                                            size="sm"
                                            onClick={() => setSelectedLearningArea(area + "")}
                                        >
                                            {area} ({count})
                                        </Button>
                                    );
                                })}
                            </div>
                        </div>
                    )}
                </CardContent>
            </Card>

            {chartData.length > 0 && (
                <Card>
                    <CardHeader>
                        <CardTitle>Learning Area Breakdown</CardTitle>
                        <CardDescription>Attendance percentage by learning area</CardDescription>
                    </CardHeader>
                    <CardContent>
                        <div className="h-80 w-full">
                            <ResponsiveContainer width="100%" height="100%">
                                <PieChart>
                                    <Pie
                                        data={chartData}
                                        cx="50%"
                                        cy="50%"
                                        outerRadius={100}
                                        fill="#8884d8"
                                        dataKey="value"
                                        label={({ name, percent }) =>
                                            percent && percent > 0
                                                ? `${name}: ${(percent * 100).toFixed(0)}%`
                                                : null
                                        }
                                        labelLine={false}
                                    >
                                        {chartData.map((entry, index) => (
                                            <Cell key={`cell-${index}`} fill={entry.color} />
                                        ))}
                                    </Pie>
                                    <Tooltip formatter={(value) => [`${value}%`, "Attendance"]} />
                                    <Legend />
                                </PieChart>
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
                    <CardTitle>Detailed Summary</CardTitle>
                    <CardDescription>Detailed breakdown by learning area</CardDescription>
                </CardHeader>
                <CardContent>
                    <div className="space-y-3">
                        {filteredSummaries.map((summary) => (
                            <div
                                key={summary.id}
                                className="flex items-center justify-between rounded-md border p-4"
                            >
                                <div className="space-y-1">
                                    <p className="font-medium">
                                        {summary.learning_area_name || "Unknown"}
                                    </p>
                                    <p className="text-muted-foreground text-sm">
                                        {formatNumber(summary.periods_present)} /{" "}
                                        {formatNumber(summary.periods_total)} periods
                                    </p>
                                </div>
                                <div className="space-y-1 text-right">
                                    <p className="font-medium text-emerald-600">
                                        {summary.attendance_percentage.toFixed(1)}%
                                    </p>
                                    <p className="text-muted-foreground text-xs">
                                        Present: {formatNumber(summary.periods_present)}, Absent:{" "}
                                        {formatNumber(summary.periods_absent)}, Late:{" "}
                                        {formatNumber(summary.periods_late)}, Excused:{" "}
                                        {formatNumber(summary.periods_excused)}
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
