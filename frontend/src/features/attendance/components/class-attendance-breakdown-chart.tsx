"use client";

import { Bar, BarChart, CartesianGrid, XAxis, YAxis } from "recharts";
import {
    ChartConfig,
    ChartContainer,
    ChartTooltip,
    ChartTooltipContent,
} from "@/components/ui/chart";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Skeleton } from "@/components/ui/skeleton";
import {
    useClassAttendanceBreakdowns,
    useCurrentTermId,
} from "@/features/attendance/hooks/use-class-attendance-breakdowns";
import { getErrorMessage } from "@/lib/errors";

const chartConfig = {
    present_count: {
        label: "Present Count",
        color: "#10B981",
    },
    late_count: {
        label: "Late Count",
        color: "#F59E0B",
    },
    absent_count: {
        label: "Absent Count",
        color: "#EF4444",
    },
} satisfies ChartConfig;

interface ClassAttendanceBreakdownChartProps {
    /** Optional academic term id; when omitted the active term is resolved. */
    termId?: string;
}

/**
 * Shadcn UI grouped BarChart for the School Administrator dashboard.
 *
 * Compares Present, Late, and Absent counts across classes (grouped
 * horizontal bars). Absent is rendered in red and the backend orders classes
 * by absent count descending, so high-absenteeism classes — the truancy and
 * chronic absenteeism watch list — surface at the top.
 *
 * Backed by GET /api/v1/attendance/class-term/breakdown?academic_term_id=….
 */
export function ClassAttendanceBreakdownChart({ termId }: ClassAttendanceBreakdownChartProps) {
    // Resolve the active term only when the caller did not pass one — the
    // chart stays a zero-config dashboard section like SchoolAttendanceKPIs.
    const currentTermQuery = useCurrentTermId(!termId);
    const effectiveTermId = termId ?? currentTermQuery.data;
    const { data, isLoading, isError, error } = useClassAttendanceBreakdowns(effectiveTermId);

    const isResolvingTerm = !termId && currentTermQuery.isPending;
    const loadFailed = isError || currentTermQuery.isError;

    const heading = (
        <h3 className="text-foreground text-lg font-medium">
            Class attendance: present vs. late vs. absent breakdown
        </h3>
    );

    if (isLoading || isResolvingTerm) {
        return (
            <section className="space-y-4">
                {heading}
                <Skeleton className="h-80 w-full" />
            </section>
        );
    }

    if (loadFailed) {
        return (
            <section className="space-y-4">
                {heading}
                <Alert variant="destructive">
                    <AlertTitle>Unable to load class attendance breakdown</AlertTitle>
                    <AlertDescription>
                        {getErrorMessage(error ?? currentTermQuery.error)}
                    </AlertDescription>
                </Alert>
            </section>
        );
    }

    // Guard against a partially-loaded response before accessing items.
    const items = data?.items ?? [];

    if (items.length === 0) {
        return (
            <section className="space-y-4">
                {heading}
                <p className="text-muted-foreground text-sm">
                    No attendance summaries for this term yet. Mark attendance to see the class
                    breakdown here.
                </p>
            </section>
        );
    }

    return (
        <section className="space-y-4">
            {heading}
            <ChartContainer config={chartConfig} className="h-80 w-full">
                <BarChart
                    data={items}
                    layout="vertical"
                    margin={{ top: 5, right: 30, left: 40, bottom: 5 }}
                >
                    <CartesianGrid strokeDasharray="3 3" horizontal={false} />
                    <XAxis type="number" />
                    <YAxis
                        type="category"
                        dataKey="class_name"
                        width={100}
                        tickLine={false}
                        axisLine={false}
                    />
                    <ChartTooltip content={<ChartTooltipContent />} />
                    <Bar
                        dataKey="present_count"
                        fill="var(--color-present_count)"
                        radius={[0, 4, 4, 0]}
                    />
                    <Bar
                        dataKey="late_count"
                        fill="var(--color-late_count)"
                        radius={[0, 4, 4, 0]}
                    />
                    <Bar
                        dataKey="absent_count"
                        fill="var(--color-absent_count)"
                        radius={[0, 4, 4, 0]}
                    />
                </BarChart>
            </ChartContainer>
        </section>
    );
}
