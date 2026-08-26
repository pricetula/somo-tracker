"use client";

import Link from "next/link";
import { PlusIcon } from "lucide-react";
import { useTimetableView, type TimetableViewResult } from "../hooks";
import { AllocationBlock } from "./allocation-block";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";

export function TimeTable() {
    const { data, isLoading } = useTimetableView();

    if (isLoading) {
        return (
            <div className="space-y-4">
                <div className="flex items-center justify-between">
                    <h1 className="text-lg font-medium">Timetable</h1>
                    <Link href="/timetable/new">
                        <Button variant="default" size="sm">
                            <PlusIcon className="mr-1.5 size-3.5" />
                            New Timetable
                        </Button>
                    </Link>
                </div>
                <article className="relative max-h-150 w-full overflow-auto rounded-md border text-xs">
                    <table className="w-full">
                        <thead className="border-b tracking-wider">
                            <tr>
                                <th
                                    scope="col"
                                    className="bg-background sticky top-0 left-0 z-30 min-w-50 border-r pl-4 text-left md:w-34"
                                >
                                    <Skeleton className="h-4 w-24" />
                                </th>
                                {Array.from({ length: 5 }).map((_, i) => (
                                    <th
                                        key={i}
                                        scope="col"
                                        className="bg-background sticky top-0 z-20 min-w-50 border-r py-2 pl-4 text-left"
                                    >
                                        <Skeleton className="h-4 w-20" />
                                    </th>
                                ))}
                            </tr>
                        </thead>
                        <tbody className="divide-y">
                            {Array.from({ length: 6 }).map((_, rowIdx) => (
                                <tr key={rowIdx}>
                                    <td
                                        scope="row"
                                        className="bg-background sticky left-0 z-10 border-r p-4 align-top whitespace-nowrap"
                                    >
                                        <Skeleton className="mb-1 h-4 w-28" />
                                        <Skeleton className="h-3 w-16" />
                                    </td>
                                    {Array.from({ length: 5 }).map((_, cellIdx) => (
                                        <td key={cellIdx} className="border-r p-4">
                                            <Skeleton className="h-10 w-full" />
                                        </td>
                                    ))}
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </article>
            </div>
        );
    }

    if (!data)
        return (
            <div className="space-y-4">
                <div className="flex items-center justify-between">
                    <h1 className="text-lg font-medium">Timetable</h1>
                    <Link href="/timetable/new">
                        <Button variant="default" size="sm">
                            <PlusIcon className="mr-1.5 size-3.5" />
                            New Timetable
                        </Button>
                    </Link>
                </div>
                <p className="text-muted-foreground">No timetable data available.</p>
            </div>
        );

    const { days = [], rows = [] } = data as TimetableViewResult;

    return (
        <div className="space-y-4">
            <div className="flex items-center justify-between">
                <h1 className="text-lg font-medium">Timetable</h1>
                <Link href="/timetable/new">
                    <Button variant="default" size="sm">
                        <PlusIcon className="mr-1.5 size-3.5" />
                        New Timetable
                    </Button>
                </Link>
            </div>
            <article className="relative max-h-150 w-full overflow-auto rounded-md border text-xs">
                <table className="w-full">
                    <thead className="border-b tracking-wider">
                        <tr>
                            <th
                                scope="col"
                                className="bg-background sticky top-0 left-0 z-30 min-w-50 border-r pl-4 text-left md:w-34"
                            >
                                Time Slots
                            </th>
                            {days.map((day) => (
                                <th
                                    key={day.day_of_week}
                                    scope="col"
                                    className="bg-background sticky top-0 z-20 min-w-50 border-r py-2 pl-4 text-left"
                                >
                                    {day.name}
                                </th>
                            ))}
                        </tr>
                    </thead>
                    <tbody className="divide-y">
                        {rows.map((period) => {
                            return (
                                <tr key={period.period_name}>
                                    <td
                                        scope="row"
                                        className="bg-background sticky left-0 z-10 border-r p-4 align-top whitespace-nowrap"
                                    >
                                        <div className="font-medium">{period.period_name}</div>
                                        <div className="text-muted-foreground text-[10px]">
                                            {period.start_time} - {period.end_time}
                                        </div>
                                    </td>

                                    {days.map((day) => {
                                        const allocation = period.allocationByDay[day.day_of_week];

                                        if (!allocation) {
                                            return (
                                                <td
                                                    key={day.day_of_week}
                                                    className="bg-row-disabled border-r pl-4"
                                                />
                                            );
                                        }

                                        if (period.is_break) {
                                            return (
                                                <td
                                                    key={day.day_of_week}
                                                    className="bg-row-disabled border-r"
                                                >
                                                    <AllocationBlock
                                                        allocation={allocation}
                                                        isBreak={period.is_break}
                                                    />
                                                </td>
                                            );
                                        }

                                        // const cell = dayCells[0] ?? null;

                                        return (
                                            <td key={day.day_of_week} className="border-r">
                                                <AllocationBlock
                                                    allocation={allocation}
                                                    isBreak={period.is_break}
                                                />
                                            </td>
                                        );
                                    })}
                                </tr>
                            );
                        })}
                    </tbody>
                </table>
            </article>
        </div>
    );
}
