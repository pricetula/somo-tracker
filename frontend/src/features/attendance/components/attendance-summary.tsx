"use client";

import { TrendingUp } from "lucide-react";
import { Label, PolarRadiusAxis, RadialBar, RadialBarChart, PolarAngleAxis, Cell } from "recharts";

import {
    Card,
    CardContent,
    CardAction,
    CardFooter,
    CardHeader,
    CardTitle,
} from "@/components/ui/card";
import {
    ChartContainer,
    ChartTooltip,
    ChartTooltipContent,
    type ChartConfig,
} from "@/components/ui/chart";

const apiData = {
    academic_year: "2026",
    data: [
        // --- TERM 1 ---
        {
            class_name: "All",
            term_name: "Term 1",
            academic_year: "2026",
            present_percentage: 90.0,
            absent_percentage: 6.0,
            excused_percentage: 2.0,
            late_percentage: 2.0,
            days_in_term: 60,
            total_enrolled_avg: 150,
        },
        {
            class_name: "Grade 7 North",
            term_name: "Term 1",
            academic_year: "2026",
            present_percentage: 90.0,
            absent_percentage: 6.0,
            excused_percentage: 2.0,
            late_percentage: 2.0,
            days_in_term: 60,
            total_enrolled_avg: 50,
        },

        // --- TERM 2 ---
        {
            class_name: "All",
            term_name: "Term 2",
            academic_year: "2026",
            present_percentage: 89.5,
            absent_percentage: 6.5,
            excused_percentage: 2.0,
            late_percentage: 2.0,
            days_in_term: 60,
            total_enrolled_avg: 150,
        },
        {
            class_name: "Grade 7 North",
            term_name: "Term 2",
            academic_year: "2026",
            present_percentage: 91.0,
            absent_percentage: 5.5,
            excused_percentage: 1.5,
            late_percentage: 2.0,
            days_in_term: 60,
            total_enrolled_avg: 50,
        },

        // --- TERM 3 ---
        {
            class_name: "All",
            term_name: "Term 3",
            academic_year: "2026",
            present_percentage: 91.96,
            absent_percentage: 4.85,
            excused_percentage: 1.62,
            late_percentage: 1.94,
            days_in_term: 65,
            total_enrolled_avg: 150,
        },
        {
            class_name: "Grade 7 North",
            term_name: "Term 3",
            academic_year: "2026",
            present_percentage: 93.2,
            absent_percentage: 3.88,
            excused_percentage: 0.97,
            late_percentage: 1.94,
            days_in_term: 65,
            total_enrolled_avg: 50,
        },
    ],
};

const chartConfig = {
    absent: {
        label: "Absent",
    },
    present: {
        label: "Present",
    },
    excused: {
        label: "Excused",
    },
} satisfies ChartConfig;

export function AttendanceSummary() {
    const latestTermIndex = chartData.length - 1;
    const currentPresentPercent = chartData[latestTermIndex]?.present;

    return (
        <Card className="flex flex-col">
            <CardHeader className="items-center pb-0">
                <CardTitle>Attendance distribution</CardTitle>
                <CardAction>ss</CardAction>
            </CardHeader>
            <CardContent className="flex flex-1 items-center pb-0">
                <ChartContainer
                    config={chartConfig}
                    className="relative top-10 mx-auto h-56 w-full max-w-62.5"
                >
                    <RadialBarChart
                        data={chartData}
                        endAngle={180}
                        innerRadius={80}
                        outerRadius={110}
                    >
                        <PolarAngleAxis
                            type="number"
                            domain={[0, 100]}
                            angleAxisId={0}
                            tick={false}
                            tickLine={false}
                            axisLine={false}
                        />
                        <RadialBar
                            dataKey="absent"
                            stackId="a"
                            cornerRadius={5}
                            className="stroke-transparent stroke-2"
                        >
                            {chartData.map((_, index) => (
                                <Cell
                                    key={`absent-${index}`}
                                    fill="var(--destructive)"
                                    fillOpacity={index === latestTermIndex ? 1 : 0.25}
                                />
                            ))}
                        </RadialBar>
                        <RadialBar
                            dataKey="excused"
                            stackId="a"
                            cornerRadius={5}
                            className="stroke-transparent stroke-2"
                        >
                            {chartData.map((_, index) => (
                                <Cell
                                    key={`excused-${index}`}
                                    fill="var(--chart-1)"
                                    fillOpacity={index === latestTermIndex ? 1 : 0.25}
                                />
                            ))}
                        </RadialBar>
                        <RadialBar
                            dataKey="present"
                            stackId="a"
                            cornerRadius={5}
                            className="stroke-transparent stroke-2"
                        >
                            {chartData.map((_, index) => (
                                <Cell
                                    key={`present-${index}`}
                                    fill="var(--chart-3)"
                                    fillOpacity={index === latestTermIndex ? 1 : 0.25}
                                />
                            ))}
                        </RadialBar>

                        <ChartTooltip
                            cursor={false}
                            content={({ active, payload }) => {
                                return (
                                    <ChartTooltipContent
                                        active={active}
                                        payload={payload}
                                        labelFormatter={(_, payload) =>
                                            payload?.[0]?.payload?.academicTermName
                                        }
                                    />
                                );
                            }}
                        />

                        <PolarRadiusAxis tick={false} tickLine={false} axisLine={false}>
                            <Label
                                content={({ viewBox }) => {
                                    if (viewBox && "cx" in viewBox && "cy" in viewBox) {
                                        return (
                                            <text x={viewBox.cx} y={viewBox.cy} textAnchor="middle">
                                                <tspan
                                                    x={viewBox.cx}
                                                    y={(viewBox.cy || 0) - 16}
                                                    className="fill-foreground text-2xl font-bold"
                                                >
                                                    {currentPresentPercent + "%"}
                                                </tspan>
                                                <tspan
                                                    x={viewBox.cx}
                                                    y={(viewBox.cy || 0) + 4}
                                                    className="fill-muted-foreground"
                                                >
                                                    Present
                                                </tspan>
                                            </text>
                                        );
                                    }
                                }}
                            />
                        </PolarRadiusAxis>
                    </RadialBarChart>
                </ChartContainer>
            </CardContent>
            <CardFooter className="flex-col gap-2 text-sm">
                <div className="flex items-center gap-2 leading-none font-medium">
                    Trending up by 5.2% this month <TrendingUp className="h-4 w-4" />
                </div>
                <div className="text-muted-foreground leading-none">
                    Showing total visitors for the last 6 months
                </div>
            </CardFooter>
        </Card>
    );
}
