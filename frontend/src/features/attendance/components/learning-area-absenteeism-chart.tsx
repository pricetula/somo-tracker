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
    useCurrentTermId,
    useLearningAreaAttendanceBreakdowns,
} from "@/features/attendance/hooks/use-learning-area-attendance-breakdowns";
import { getErrorMessage } from "@/lib/errors";

const chartConfig = {
    periods_present: {
        label: "Periods Present",
        color: "#10B981",
    },
    periods_absent: {
        label: "Periods Absent",
        color: "#EF4444",
    },
    periods_excused: {
        label: "Periods Excused",
        color: "#F59E0B",
    },
} satisfies ChartConfig;

interface LearningAreaAbsenteeismChartProps {
    /** Optional academic term id; when omitted the active term is resolved. */
    termId?: string;
}

/**
 * Shadcn UI grouped BarChart for the School Administrator dashboard.
 *
 * Compares Periods Present, Periods Absent, and Periods Excused per learning
 * area (grouped horizontal bars). Absent is rendered in red and the backend
 * orders learning areas by absent period count descending, so subjects with
 * high truancy / absenteeism — the disengagement hotspot watch — surface at
 * the top. Hover tooltips break down the period metrics per subject.
 *
 * Backed by GET /api/v1/attendance/class-learning-area/breakdown?academic_term_id=….
 */
export function LearningAreaAbsenteeismChart({ termId }: LearningAreaAbsenteeismChartProps) {
    // Resolve the active term only when the caller did not pass one — the
    // chart stays a zero-config dashboard section like SchoolAttendanceKPIs.
    const currentTermQuery = useCurrentTermId(!termId);
    const effectiveTermId = termId ?? currentTermQuery.data;
    const { data, isLoading, isError, error } =
        useLearningAreaAttendanceBreakdowns(effectiveTermId);

    const isResolvingTerm = !termId && currentTermQuery.isPending;
    const loadFailed = isError || currentTermQuery.isError;

    const heading = (
        <h3 className="text-foreground text-lg font-medium">
            Learning area attendance: present vs. absent vs. excused breakdown
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
                    <AlertTitle>Unable to load learning area attendance breakdown</AlertTitle>
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
                <p className="text-muted-foreground">
                    No learning area attendance summaries for this term yet. Mark attendance to see
                    the subject breakdown here.
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
                    margin={{ top: 5, right: 30, left: 80, bottom: 5 }}
                >
                    <CartesianGrid strokeDasharray="3 3" horizontal={false} />
                    <XAxis type="number" />
                    <YAxis
                        type="category"
                        dataKey="learning_area_name"
                        width={130}
                        tickLine={false}
                        axisLine={false}
                    />
                    <ChartTooltip content={<ChartTooltipContent />} />
                    <Bar
                        dataKey="periods_present"
                        fill="var(--color-periods_present)"
                        radius={[0, 4, 4, 0]}
                    />
                    <Bar
                        dataKey="periods_absent"
                        fill="var(--color-periods_absent)"
                        radius={[0, 4, 4, 0]}
                    />
                    <Bar
                        dataKey="periods_excused"
                        fill="var(--color-periods_excused)"
                        radius={[0, 4, 4, 0]}
                    />
                </BarChart>
            </ChartContainer>
        </section>
    );
}
