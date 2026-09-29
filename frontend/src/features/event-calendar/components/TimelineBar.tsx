"use client";

import { eachDayOfInterval, format, differenceInDays, startOfYear, endOfYear } from "date-fns";
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { eventApi } from "../services/event-api";
import { Button } from "@/components/ui/button";
import { EventCreateDialog } from "./EventCreateDialog";

export function TimelineBar({ year }: { year: number }) {
    const [open, setOpen] = useState(false);
    const start = startOfYear(new Date(year, 0, 1));
    const end = endOfYear(new Date(year, 11, 31));
    const days = eachDayOfInterval({ start, end });

    const { data: events = [], isLoading } = useQuery({
        queryKey: ["events", "year", year],
        queryFn: async () => {
            const startStr = format(start, "yyyy-MM-dd");
            const endStr = format(end, "yyyy-MM-dd");
            return eventApi.list(startStr, endStr);
        },
        staleTime: 60_000,
    });

    const cellDays = 3;
    const rows = events.map((ev) => {
        const startOffsetDays = Math.max(0, differenceInDays(new Date(ev.start_date), start));
        const durationDays = Math.max(
            1,
            differenceInDays(new Date(ev.end_date), new Date(ev.start_date)) + 1
        );
        return {
            ...ev,
            startOffset: Math.floor(startOffsetDays / cellDays),
            duration: Math.max(1, Math.ceil(durationDays / cellDays)),
        };
    });

    const typeColor: Record<string, string> = {
        SPORTS: "bg-blue-500",
        EXAM: "bg-red-500",
        ADMISSION: "bg-green-500",
        MEETING: "bg-amber-500",
    };
    const getColor = (type: string) => typeColor[type?.toUpperCase()] || "bg-gray-500";

    const dayWidth = 24;
    const headerHeight = 28;
    const rowHeight = 32;

    return (
        <>
            <div className="w-full overflow-hidden rounded-lg border">
                <div className="flex">
                    <div className="bg-muted/30 w-60 shrink-0 border-r">
                        <div className="text-muted-foreground flex h-7 items-center justify-between border-b px-2 text-xs font-medium">
                            Events
                            <Button
                                size="sm"
                                variant="default"
                                onClick={() => setOpen(true)}
                                className="h-6 text-[10px]"
                            >
                                + Add
                            </Button>
                        </div>
                        {rows.map((ev) => (
                            <div
                                key={ev.id}
                                className="flex h-8 items-center truncate border-b px-2 text-sm"
                            >
                                {ev.title}
                                <span className="text-muted-foreground ml-2 text-[10px]">
                                    {ev.event_type}
                                </span>
                            </div>
                        ))}
                        {isLoading && (
                            <div className="text-muted-foreground p-2 text-xs">Loading…</div>
                        )}
                        {!isLoading && rows.length === 0 && (
                            <div className="text-muted-foreground p-2 text-xs">No events</div>
                        )}
                    </div>

                    <div className="flex-1 overflow-x-auto">
                        <div
                            style={{
                                width: Math.ceil(days.length / cellDays) * dayWidth,
                                minWidth: "100%",
                            }}
                        >
                            <div className="flex" style={{ height: headerHeight }}>
                                {Array.from({ length: 12 }).map((_, i) => {
                                    const monthStart = new Date(year, i, 1);
                                    const monthDays = new Date(year, i + 1, 0).getDate();
                                    const monthWidth = Math.ceil(monthDays / cellDays) * dayWidth;
                                    return (
                                        <div
                                            key={i}
                                            className="bg-muted/30 flex items-center border-r border-b px-1 text-[10px] font-medium"
                                            style={{ width: monthWidth }}
                                        >
                                            {format(monthStart, "MMM")}
                                        </div>
                                    );
                                })}
                            </div>

                            <div className="relative">
                                {rows.map((ev, i) => {
                                    const left = ev.startOffset * dayWidth;
                                    const width = ev.duration * dayWidth;
                                    const color = getColor(ev.event_type);
                                    return (
                                        <div
                                            key={ev.id}
                                            className="absolute"
                                            style={{
                                                top: i * rowHeight,
                                                height: rowHeight,
                                                left,
                                                width: Math.max(width, dayWidth),
                                            }}
                                        >
                                            <div
                                                title={`${ev.title} (${ev.event_type}) • ${ev.start_date} → ${ev.end_date}`}
                                                className={`mt-2 rounded-lg border ${color} cursor-pointer truncate border-2 border-white/5 px-1 text-[10px] text-white hover:border-white/50`}
                                            >
                                                {color}-{ev.title}
                                            </div>
                                        </div>
                                    );
                                })}

                                <div style={{ height: rows.length * rowHeight, minHeight: 200 }}>
                                    {rows.map((row, i) => (
                                        <div
                                            key={i}
                                            style={{ height: rowHeight }}
                                            className="border-b border-dashed"
                                        />
                                    ))}
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
            <EventCreateDialog open={open} onOpenChange={setOpen} />
        </>
    );
}
