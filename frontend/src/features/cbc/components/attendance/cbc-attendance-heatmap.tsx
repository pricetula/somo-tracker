"use client";

import * as React from "react";
import { format, startOfMonth, endOfMonth, eachDayOfInterval, isSameDay } from "date-fns";
import { cn } from "@/lib/utils";
import type { AttendanceHeatmapDay } from "@/features/cbc/types";

// ─── Props ────────────────────────────────────────────────────────────────

interface CbcAttendanceHeatmapProps {
    /** Heatmap data for the term (one entry per day with attendance). */
    data: AttendanceHeatmapDay[];
    /** Selected date to highlight. */
    selectedDate: string;
    /** Called when a day cell is clicked. */
    onSelectDate: (date: string) => void;
    /** The current academic term's start and end dates for rendering context. */
    termStart: string;
    termEnd: string;
}

// ─── Color helpers ────────────────────────────────────────────────────────

function getCellColor(rate: number | null, hasPeriods: boolean): string {
    // Empty/gray = no periods recorded at all
    if (!hasPeriods) {
        return "bg-gray-100 text-gray-400";
    }
    // null rate with periods = 0% present (all absent)
    if (rate === null) {
        return "bg-red-100 text-red-800";
    }
    if (rate >= 0.9) return "bg-green-600 text-white";
    if (rate >= 0.75) return "bg-green-400 text-white";
    if (rate >= 0.5) return "bg-amber-300 text-amber-900";
    if (rate >= 0.25) return "bg-orange-300 text-orange-900";
    return "bg-red-300 text-red-900";
}

// ─── Component ────────────────────────────────────────────────────────────

export function CbcAttendanceHeatmap({
    data,
    selectedDate,
    onSelectDate,
    termStart,
    termEnd,
}: CbcAttendanceHeatmapProps) {
    // ── Build day grid for the current month ─────────────────────────
    const today = React.useMemo(() => new Date(), []);
    const selected = React.useMemo(() => new Date(selectedDate), [selectedDate]);
    // ── Month navigation ─────────────────────────────────────────────
    const [currentMonthOffset, setCurrentMonthOffset] = React.useState(0);
    const effectiveMonth = React.useMemo(
        () => new Date(selected.getFullYear(), selected.getMonth() + currentMonthOffset, 1),
        [selected, currentMonthOffset]
    );

    const canGoPrev = effectiveMonth >= startOfMonth(new Date(termStart));
    const canGoNext = effectiveMonth < startOfMonth(new Date(termEnd));

    const navigateMonth = (delta: number) => {
        setCurrentMonthOffset((prev) => prev + delta);
    };

    // Regenerate days when month changes
    const monthDays = React.useMemo(() => {
        const mStart = startOfMonth(effectiveMonth);
        const mEnd = endOfMonth(effectiveMonth);
        const allDays = eachDayOfInterval({ start: mStart, end: mEnd });
        const sDay = mStart.getDay();
        const pad = sDay === 0 ? 6 : sDay - 1;
        const padded: (Date | null)[] = [...Array.from({ length: pad }, () => null), ...allDays];
        return padded;
    }, [effectiveMonth]);

    const dayLabels = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"];

    // ── Find heatmap entry for a date ─────────────────────────────────
    const findEntry = (date: Date): AttendanceHeatmapDay | undefined => {
        const key = format(date, "yyyy-MM-dd");
        return data.find((d) => d.date === key);
    };

    return (
        <div className="space-y-2">
            {/* Month header */}
            <div className="flex items-center justify-between">
                <h3 className="text-sm font-medium text-gray-700">
                    {format(effectiveMonth, "MMMM yyyy")}
                </h3>
                <div className="flex items-center gap-1">
                    <button
                        type="button"
                        onClick={() => navigateMonth(-1)}
                        disabled={!canGoPrev}
                        className="text-muted-foreground hover:text-foreground inline-flex size-6 items-center justify-center rounded text-xs disabled:opacity-30"
                        aria-label="Previous month"
                    >
                        ◀
                    </button>
                    <button
                        type="button"
                        onClick={() => navigateMonth(1)}
                        disabled={!canGoNext}
                        className="text-muted-foreground hover:text-foreground inline-flex size-6 items-center justify-center rounded text-xs disabled:opacity-30"
                        aria-label="Next month"
                    >
                        ▶
                    </button>
                </div>
            </div>

            {/* Day-of-week labels */}
            <div className="grid grid-cols-7 gap-1">
                {dayLabels.map((label) => (
                    <div
                        key={label}
                        className="text-muted-foreground text-center text-[10px] font-medium"
                    >
                        {label}
                    </div>
                ))}
            </div>

            {/* Day grid */}
            <div className="grid grid-cols-7 gap-1">
                {monthDays.map((day, i) => {
                    if (!day) {
                        return <div key={`empty-${i}`} className="size-8" />;
                    }

                    const entry = findEntry(day);
                    const hasPeriods = entry ? entry.period_count > 0 : false;
                    const rate = entry?.present_rate ?? null;
                    const isSelected = isSameDay(day, selected);
                    const isToday = isSameDay(day, today);
                    const dateKey = format(day, "yyyy-MM-dd");

                    // Future dates (beyond today) are disabled
                    const isFuture = day > today;

                    return (
                        <button
                            key={dateKey}
                            type="button"
                            onClick={() => {
                                if (!isFuture) {
                                    onSelectDate(dateKey);
                                }
                            }}
                            disabled={isFuture}
                            className={cn(
                                "relative flex size-8 items-center justify-center rounded text-[11px] font-medium transition-colors",
                                isFuture && "cursor-default opacity-30",
                                !isFuture && "cursor-pointer",
                                hasPeriods ? getCellColor(rate, true) : "bg-gray-50 text-gray-300",
                                isSelected && "ring-2 ring-teal-500 ring-offset-1",
                                isToday && !isSelected && "ring-1 ring-gray-300"
                            )}
                            title={
                                hasPeriods
                                    ? `${format(day, "MMM d")}: ${rate !== null ? Math.round(rate * 100) : 0}% present`
                                    : `${format(day, "MMM d")}: No attendance recorded`
                            }
                        >
                            {day.getDate()}
                        </button>
                    );
                })}
            </div>

            {/* Legend */}
            <div className="flex items-center gap-3 text-[10px] text-gray-500">
                <span>No record</span>
                <span className="inline-block size-3 rounded bg-gray-50" />
                <span>Low</span>
                <span className="inline-block size-3 rounded bg-red-300" />
                <span className="inline-block size-3 rounded bg-amber-300" />
                <span className="inline-block size-3 rounded bg-green-400" />
                <span>High</span>
                <span className="inline-block size-3 rounded bg-green-600" />
            </div>
        </div>
    );
}
