"use client";

import React from "react";
import { TrendingUp, TrendingDown } from "lucide-react";
import { AnimatePresence, motion } from "framer-motion";
import {
    Combobox,
    ComboboxContent,
    ComboboxEmpty,
    ComboboxInput,
    ComboboxItem,
    ComboboxList,
} from "@/components/ui/combobox";
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
import { SvgNumberTicker } from "@/components/shared/svg-ticker";

function TrendIndicator({ percentageChange }: { percentageChange: number }) {
    return (
        // mode="wait" ensures the exiting icon finishes animating out BEFORE the new icon animates in
        <AnimatePresence mode="wait">
            {percentageChange > 0 ? (
                <motion.div
                    key="trending-up" // Unique key is REQUIRED for Framer Motion to detect the switch
                    initial={{ opacity: 0, scale: 0.5, y: 5 }}
                    animate={{ opacity: 1, scale: 1, y: 0 }}
                    exit={{ opacity: 0, scale: 0.5, y: -5 }}
                    transition={{ duration: 0.3 }}
                >
                    <TrendingUp size={16} className="text-teal-500" />
                </motion.div>
            ) : (
                <motion.div
                    key="trending-down" // Unique key is REQUIRED for Framer Motion to detect the switch
                    initial={{ opacity: 0, scale: 0.5, y: -5 }}
                    animate={{ opacity: 1, scale: 1, y: 0 }}
                    exit={{ opacity: 0, scale: 0.5, y: 5 }}
                    transition={{ duration: 0.3 }}
                >
                    <TrendingDown size={16} className="text-rose-500" />
                </motion.div>
            )}
        </AnimatePresence>
    );
}

const data = {
    academic_year: "2026",
    data: [
        // --- TERM 1 ---
        {
            class_name: "All",
            term_name: "Term 1",
            term_number: 1,
            academic_year: "2026",
            present_percentage: 92.0,
            absent_percentage: 4.0,
            excused_percentage: 2.0,
            late_percentage: 2.0,
            days_in_term: 60,
            total_enrolled_avg: 150,
        },
        {
            class_name: "Grade 7 North",
            term_name: "Term 1",
            term_number: 1,
            academic_year: "2026",
            present_percentage: 90.0,
            absent_percentage: 6.0,
            excused_percentage: 2.0,
            late_percentage: 2.0,
            days_in_term: 60,
            total_enrolled_avg: 50,
        },
        {
            class_name: "Grade 7 East",
            term_name: "Term 1",
            term_number: 1,
            academic_year: "2026",
            present_percentage: 80.0,
            absent_percentage: 16.0,
            excused_percentage: 12.0,
            late_percentage: 2.0,
            days_in_term: 60,
            total_enrolled_avg: 50,
        },

        // --- TERM 2 ---
        {
            class_name: "All",
            term_name: "Term 2",
            term_number: 2,
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
            term_number: 2,
            academic_year: "2026",
            present_percentage: 91.0,
            absent_percentage: 5.5,
            excused_percentage: 1.5,
            late_percentage: 2.0,
            days_in_term: 60,
            total_enrolled_avg: 50,
        },
        {
            class_name: "Grade 7 East",
            term_name: "Term 2",
            term_number: 2,
            academic_year: "2026",
            present_percentage: 70.0,
            absent_percentage: 6.0,
            excused_percentage: 12.0,
            late_percentage: 12.0,
            days_in_term: 60,
            total_enrolled_avg: 50,
        },

        // --- TERM 3 ---
        {
            class_name: "All",
            term_name: "Term 3",
            term_number: 3,
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
            term_number: 3,
            academic_year: "2026",
            present_percentage: 93.2,
            absent_percentage: 3.88,
            excused_percentage: 0.97,
            late_percentage: 1.94,
            days_in_term: 65,
            total_enrolled_avg: 50,
        },
        {
            class_name: "Grade 7 East",
            term_name: "Term 3",
            term_number: 3,
            academic_year: "2026",
            present_percentage: 61,
            absent_percentage: 19,
            excused_percentage: 8,
            late_percentage: 12,
            days_in_term: 65,
            total_enrolled_avg: 50,
        },
    ],
};

const chartConfig = {
    absent_percentage: {
        label: "Absent",
    },
    present_percentage: {
        label: "Present",
    },
    excused_percentage: {
        label: "Excused",
    },
    late_percentage: {
        label: "Late",
    },
} satisfies ChartConfig;

export function AttendanceSummary() {
    const [hasMounted, setHasMounted] = React.useState(false);
    const [summaryGroup, setSummaryGroup] = React.useState("All");

    const mappedData = React.useMemo(() => {
        if (!data?.data?.length) return null;

        const sorted = data.data.sort((a, b) => a.term_number - b.term_number);

        return sorted.reduce((acc, item) => {
            if (!acc.has(item.class_name)) {
                acc.set(item.class_name, []);
            }
            acc.get(item.class_name).push(item);
            return acc;
        }, new Map());
    }, [data]);

    const mappedKeys = React.useMemo(() => mappedData?.keys?.()?.toArray?.() || [], [mappedData]);

    const selectedData = React.useMemo(
        () => mappedData?.get(summaryGroup) || [],
        [mappedData, summaryGroup]
    );

    const latestTermIndex = selectedData.length - 1;

    const latestPresentPercentage = selectedData[latestTermIndex]?.present_percentage ?? 0;

    const previousPresentPercentage =
        latestTermIndex > 0 ? (selectedData[latestTermIndex - 1]?.present_percentage ?? 0) : 0;

    const percentageChange = latestPresentPercentage - previousPresentPercentage;

    React.useEffect(() => {
        // eslint-disable-next-line react-hooks/set-state-in-effect
        setHasMounted(true);
    }, []);

    if (!hasMounted) {
        return null;
    }

    return (
        <Card className="flex flex-col">
            <CardHeader className="flex items-center justify-between pb-0">
                <CardTitle>Attendance {data?.academic_year}</CardTitle>
                <CardAction className="text-xs">
                    <Combobox
                        id="summary-group-combobox"
                        items={mappedKeys}
                        value={summaryGroup}
                        onValueChange={(val) => setSummaryGroup(val || "")}
                    >
                        <ComboboxInput
                            placeholder="Select a framework"
                            className="max-w-34 text-xs"
                        />
                        <ComboboxContent>
                            <ComboboxEmpty>No items found.</ComboboxEmpty>
                            <ComboboxList>
                                {(item) => (
                                    <ComboboxItem key={item} value={item} className="text-xs">
                                        {item}
                                    </ComboboxItem>
                                )}
                            </ComboboxList>
                        </ComboboxContent>
                    </Combobox>
                </CardAction>
            </CardHeader>
            <CardContent className="flex flex-1 items-center pb-0">
                <ChartContainer
                    config={chartConfig}
                    className="relative top-10 mx-auto h-56 w-full max-w-62.5"
                >
                    <RadialBarChart
                        data={selectedData}
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
                            dataKey="absent_percentage"
                            stackId="a"
                            cornerRadius={0}
                            className="stroke-transparent stroke-2"
                        >
                            {selectedData.map((_: unknown, index: number) => (
                                <Cell
                                    key={`absent_percentage-${index}`}
                                    fill="var(--destructive)"
                                    fillOpacity={index === latestTermIndex ? 1 : 0.25}
                                />
                            ))}
                        </RadialBar>
                        <RadialBar
                            dataKey="excused_percentage"
                            stackId="a"
                            cornerRadius={0}
                            className="stroke-transparent stroke-2"
                        >
                            {selectedData.map((_: unknown, index: number) => (
                                <Cell
                                    key={`excused_percentage-${index}`}
                                    fill="var(--chart-1)"
                                    fillOpacity={index === latestTermIndex ? 1 : 0.25}
                                />
                            ))}
                        </RadialBar>
                        <RadialBar
                            dataKey="late_percentage"
                            stackId="a"
                            cornerRadius={0}
                            className="stroke-transparent stroke-2"
                        >
                            {selectedData.map((_: unknown, index: number) => (
                                <Cell
                                    key={`late_percentage-${index}`}
                                    fill="var(--chart-2)"
                                    fillOpacity={index === latestTermIndex ? 1 : 0.25}
                                />
                            ))}
                        </RadialBar>
                        <RadialBar
                            dataKey="present_percentage"
                            stackId="a"
                            cornerRadius={0}
                            className="stroke-transparent stroke-2"
                        >
                            {selectedData.map((_: unknown, index: number) => (
                                <Cell
                                    key={`present_percentage-${index}`}
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
                                            payload?.[0]?.payload?.term_name
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
                                                <SvgNumberTicker
                                                    value={latestPresentPercentage}
                                                    x={viewBox.cx}
                                                    y={(viewBox.cy || 0) - 16}
                                                    className="fill-foreground text-2xl font-bold"
                                                />

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
            <CardFooter className="flex-col gap-2 text-xs">
                <div className="flex items-center gap-2 leading-none font-medium">
                    Trending {percentageChange > 0 ? "up" : "down"} by{" "}
                    {Math.abs(percentageChange).toFixed(1)}% this term{" "}
                    <TrendIndicator percentageChange={percentageChange} />
                </div>
                <div className="text-muted-foreground leading-none">
                    Showing attendance distribution across terms
                </div>
            </CardFooter>
        </Card>
    );
}
