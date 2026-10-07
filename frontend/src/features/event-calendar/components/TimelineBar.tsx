"use client";

import { eachDayOfInterval, format, differenceInDays, startOfYear, endOfYear } from "date-fns";
import { useState, useEffect, useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import { eventApi } from "../services/event-api";
import { Button } from "@/components/ui/button";
import { EventCreateDialog } from "./EventCreateDialog";
import { useDeleteEvent } from "../hooks/useDeleteEvent";
import { X } from "lucide-react";
import type { Event } from "../types/event.types";

export function TimelineBar({ year }: { year: number }) {
    const [open, setOpen] = useState(false);
    const [editingEvent, setEditingEvent] = useState<Event | null>(null);
    const start = startOfYear(new Date(year, 0, 1));
    const end = endOfYear(new Date(year, 11, 31));
    const days = eachDayOfInterval({ start, end });
    const deleteEvent = useDeleteEvent();

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
    const rows = (events ?? []).map((ev) => {
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

    const dayWidth = 24;
    const scrollContainerRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        if (!scrollContainerRef.current) return;
        const today = new Date(year, new Date().getMonth(), new Date().getDate());
        const start = startOfYear(new Date(year, 0, 1));
        const offsetDays = Math.max(0, differenceInDays(today, start));
        const cellOffset = Math.floor(offsetDays / cellDays);
        const scrollTo = Math.max(0, cellOffset * dayWidth - 120);
        scrollContainerRef.current.scrollLeft = scrollTo;
    }, [year, days.length]);

    const typeColor: Record<string, string> = {
        SPORTS: "bg-blue-500",
        EXAM: "bg-red-500",
        ADMISSION: "bg-green-500",
        MEETING: "bg-amber-500",
    };
    const getColor = (type: string) => typeColor[type?.toUpperCase()] || "bg-gray-500";

    const headerHeight = 40;
    const rowHeight = 40;

    return (
        <>
            <div className="w-full overflow-hidden rounded-lg border">
                <div className="flex">
                    <div className="bg-muted/30 w-60 shrink-0 border-r">
                        <div className="text-muted-foreground flex h-10 items-center justify-between border-b px-2 text-xs font-medium">
                            Events
                            <Button
                                size="sm"
                                variant="outline"
                                onClick={() => {
                                    setEditingEvent(null);
                                    setOpen(true);
                                }}
                                className="h-6 text-[10px]"
                            >
                                + Add
                            </Button>
                        </div>
                        {rows.map((ev) => (
                            <div
                                key={ev.id}
                                className="flex h-10 items-center truncate border-b px-2 text-sm"
                            >
                                <div className="min-w-0 flex-1 truncate">
                                    {ev.title}
                                    <span className="text-muted-foreground ml-2 text-[10px]">
                                        {ev.event_type}
                                    </span>
                                </div>
                                <Button
                                    size="icon"
                                    variant="ghost"
                                    className="text-muted-foreground hover:bg-destructive hover:text-destructive-foreground hover:dark:bg-destructive/50 ml-1 h-5 w-5 shrink-0"
                                    onClick={() => deleteEvent.mutate(ev.id)}
                                    title="Delete event"
                                >
                                    <X className="h-3 w-3" />
                                </Button>
                            </div>
                        ))}
                        {isLoading && (
                            <div className="text-muted-foreground p-2 text-xs">Loading…</div>
                        )}
                        {!isLoading && rows.length === 0 && (
                            <div className="text-muted-foreground p-2 text-xs">No events</div>
                        )}
                    </div>

                    <div ref={scrollContainerRef} className="flex-1 overflow-x-auto">
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
                                            className="bg-muted/30 flex h-10 items-center border-r border-b px-1 text-[10px] font-medium"
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
                                            <button
                                                title={`${ev.title} (${ev.event_type}) • ${ev.start_date} → ${ev.end_date}`}
                                                className={`mt-2 h-6 w-full rounded-xl border text-left align-middle ${color} cursor-pointer truncate border-2 border-white/5 px-2 text-white hover:border-white/50`}
                                                onClick={() => {
                                                    setEditingEvent(ev);
                                                    setOpen(true);
                                                }}
                                            >
                                                {ev.title}
                                            </button>
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
            <EventCreateDialog open={open} onOpenChange={setOpen} event={editingEvent} />
        </>
    );
}
