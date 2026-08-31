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
import type { TeacherDeliveryBreakdownItem } from "@/lib/api/teacher-delivery";
import { useTeacherDeliveryBreakdown } from "@/features/teacher-delivery/hooks/use-teacher-delivery-breakdown";
import { getErrorMessage } from "@/lib/errors";

const chartConfig = {
    marked_slots: {
        label: "Marked Slots",
        color: "#10B981",
    },
    missed_slots: {
        label: "Missed Slots",
        color: "#EF4444",
    },
} satisfies ChartConfig;

/**
 * Shadcn UI grouped BarChart for the School Administrator dashboard.
 *
 * Compares Marked vs. Missed slot counts per teacher (grouped horizontal
 * bars). Missed slots render in red and the backend orders teachers by missed
 * slot count descending, so chronic non-compliant teachers — the skipped
 * roll-call and delivery negligence watch list — surface at the top. Hover
 * tooltips show exact counts alongside the teacher name and TSC number.
 *
 * Backed by GET /api/v1/teacher-delivery-summaries/breakdown.
 */
export function TeacherComplianceChart() {
    const { data, isLoading, isError, error } = useTeacherDeliveryBreakdown();

    const loadFailed = isError;

    const heading = (
        <h3 className="text-foreground text-lg font-medium">
            Teacher delivery: marked vs. missed slots
        </h3>
    );

    if (isLoading) {
        return (
            <section className="space-y-4">
                {heading}
                <Skeleton className="h-96 w-full" />
            </section>
        );
    }

    if (loadFailed) {
        return (
            <section className="space-y-4">
                {heading}
                <Alert variant="destructive">
                    <AlertTitle>Unable to load teacher delivery breakdown</AlertTitle>
                    <AlertDescription>{getErrorMessage(error)}</AlertDescription>
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
                    No delivery summaries for this term yet. Mark attendance or skip lessons to see
                    the teacher delivery breakdown here.
                </p>
            </section>
        );
    }

    return (
        <section className="space-y-4">
            {heading}
            <ChartContainer config={chartConfig} className="h-96 w-full">
                <BarChart
                    data={items}
                    layout="vertical"
                    margin={{ top: 5, right: 30, left: 80, bottom: 5 }}
                >
                    <CartesianGrid strokeDasharray="3 3" horizontal={false} />
                    <XAxis type="number" />
                    <YAxis
                        type="category"
                        dataKey="teacher_name"
                        width={120}
                        tickLine={false}
                        axisLine={false}
                    />
                    <ChartTooltip
                        content={
                            <ChartTooltipContent
                                labelKey="teacher_name"
                                labelFormatter={(value, payload) => {
                                    const row = payload?.[0]?.payload as
                                        | TeacherDeliveryBreakdownItem
                                        | undefined;
                                    const tsc = row?.tsc_number ? ` · TSC ${row.tsc_number}` : "";
                                    return `${value}${tsc}`;
                                }}
                            />
                        }
                    />
                    <Bar
                        dataKey="marked_slots"
                        fill="var(--color-marked_slots)"
                        radius={[0, 4, 4, 0]}
                    />
                    <Bar
                        dataKey="missed_slots"
                        fill="var(--color-missed_slots)"
                        radius={[0, 4, 4, 0]}
                    />
                </BarChart>
            </ChartContainer>
        </section>
    );
}
