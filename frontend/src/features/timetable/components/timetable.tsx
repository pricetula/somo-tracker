"use client";

import React from "react";
import { toast } from "sonner";
import { Trash } from "lucide-react";
import { formatDateString } from "@/lib/utils/date";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { ClassCombobox } from "@/features/classes/components/class-combobox";
import { useTimetableView, useBulkDeleteBlocks, type TimetableViewResult } from "../hooks";
import { AllocationBlock } from "./allocation-block";

export function TimeTable({ trackId, teacherId }: { trackId?: string; teacherId?: string } = {}) {
    const [classId, setClassId] = React.useState("");
    const { data, isLoading } = useTimetableView(trackId, classId, teacherId);
    const {
        mutate: deleteBlockMutate,
        isError: isErrorDeleteBlock,
        error: errorDeleteBlock,
    } = useBulkDeleteBlocks();

    React.useEffect(() => {
        if (isErrorDeleteBlock && errorDeleteBlock) {
            toast.error(errorDeleteBlock.message ?? "Failed to delete time block");
        }
    }, [isErrorDeleteBlock, errorDeleteBlock]);

    const handleDeleteBlock = React.useCallback(
        (blockName: string) => {
            const viewData = data as TimetableViewResult | undefined;
            if (!viewData) return;
            const ids: string[] = [];
            for (const row of viewData.rows ?? []) {
                if (row.period_name === blockName) {
                    for (const id of Object.values(row.blockIdByDay)) {
                        if (id) ids.push(id);
                    }
                }
            }
            if (ids.length === 0) {
                toast.warning("No blocks found with that name");
                return;
            }
            deleteBlockMutate(ids, {
                onSuccess: () => toast.success("Time block deleted"),
            });
        },
        [data, deleteBlockMutate]
    );

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
            <div className="flex h-48 items-center justify-center space-y-4">
                <p className="text-muted-foreground">No timetable data available.</p>
            </div>
        );

    const { days = [], rows = [] } = data as TimetableViewResult;

    if (!days.length) {
        return (
            <div className="flex h-48 items-center justify-center space-y-4">
                <p className="text-muted-foreground">No timetable data available.</p>
            </div>
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
                                        <section className="space-y-2">
                                            <header className="flex items-center justify-between">
                                                <h4>{period.period_name}</h4>
                                                <Button
                                                    size="xs"
                                                    variant="outline"
                                                    onClick={() =>
                                                        handleDeleteBlock(period.period_name)
                                                    }
                                                >
                                                    <Trash />
                                                </Button>
                                            </header>
                                            <div className="text-muted-foreground text-[10px]">
                                                {formatDateString(period.start_time)}-
                                                {formatDateString(period.end_time)}
                                            </div>
                                        </section>
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
                                            <td
                                                key={day.day_of_week}
                                                className="border-r p-4 align-top"
                                            >
                                                <AllocationBlock
                                                    allocation={allocation}
                                                    isBreak={false}
                                                    dayOfWeek={day.day_of_week}
                                                    blockId={period.blockIdByDay[day.day_of_week]}
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
