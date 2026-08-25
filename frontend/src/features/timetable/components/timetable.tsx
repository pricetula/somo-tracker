"use client";

import { useTimetableView, type TimetableViewResult } from "../hooks";
import { AllocationBlock } from "./allocation-block";

export function TimeTable() {
    const { data, isLoading } = useTimetableView();

    if (isLoading) {
        return (
            <article className="relative max-h-150 w-full overflow-auto rounded-md border text-xs">
                <div className="text-muted-foreground flex h-64 items-center justify-center">
                    Loading timetable…
                </div>
            </article>
        );
    }

    const { days = [], rows = [] } = data as TimetableViewResult;
    console.log(";;;;;", rows);
    return (
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
    );
}
