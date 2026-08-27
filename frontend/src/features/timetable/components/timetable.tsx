"use client";

import Link from "next/link";
import { PlusIcon } from "lucide-react";
import { formatDateString } from "@/lib/utils/date";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { ClassCombobox } from "@/features/classes/components/class-combobox";
import { useTimetableView, type TimetableViewResult } from "../hooks";
import { AllocationBlock } from "./allocation-block";
import { useState } from "react";

const formatDateStringOptions = {
    inputFormat: "HH:mm:ss.SSSSSS",
    outputFormat: "HH:mm",
};

export function TimeTable() {
    const [classId, setClassId] = useState("");
    const { data, isLoading } = useTimetableView();

    if (isLoading) {
        return (
            <div className="space-y-4">
                <div className="flex items-center justify-between">
                    <h1 className="text-lg font-medium">Timetable</h1>
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

    if (!days.length) {
        return (
            <Link href="/timetable/new">
                <Button variant="default" size="sm">
                    <PlusIcon className="mr-1.5 size-3.5" />
                    New Timetable
                </Button>
            </Link>
        );
    }

    return (
        <div className="space-y-4">
            <div className="flex items-center justify-between">
                <h1 className="text-lg font-medium">Timetable</h1>
                <ClassCombobox
                    value={classId}
                    onChange={(c) => {
                        if (Array.isArray(c)) {
                            setClassId(c[0]);
                            return;
                        }
                        setClassId(c);
                    }}
                />
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
                                            {formatDateString(
                                                period.start_time,
                                                formatDateStringOptions
                                            )}
                                            -
                                            {formatDateString(
                                                period.end_time,
                                                formatDateStringOptions
                                            )}
                                        </div>
                                    </td>

                                    {days.map((day) => {
                                        const allocation = period.allocationByDay[day.day_of_week];

                                        if (period.is_break) {
                                            return (
                                                <td
                                                    key={day.day_of_week}
                                                    className="bg-row-disabled border-r"
                                                >
                                                    <AllocationBlock
                                                        allocation={allocation}
                                                        isBreak={period.is_break}
                                                        classId={classId}
                                                    />
                                                </td>
                                            );
                                        }

                                        // const cell = dayCells[0] ?? null;

                                        return (
                                            <td key={day.day_of_week} className="border-r p-4">
                                                <AllocationBlock
                                                    allocation={allocation}
                                                    isBreak={false}
                                                    dayOfWeek={day.day_of_week}
                                                    blockId={period.blockId}
                                                    classId={classId}
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
