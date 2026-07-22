/**
 * DailyCalendarHeatmap — Calendar heatmap of daily attendance rates over a term.
 *
 * Visualisation: Green → red gradient showing attendance rate per day.
 * Props: Array of { date, rate } entries.
 */
"use client";

import { cn } from "@/lib/utils";
import { format, parseISO, getDay, getDate, eachDayOfInterval } from "date-fns";

// ─── Helpers ──────────────────────────────────────────────────────────────

function heatColor(rate: number): string {
    if (rate >= 95) return "bg-emerald-500/90";
    if (rate >= 90) return "bg-emerald-500/65";
    if (rate >= 85) return "bg-emerald-500/40";
    if (rate >= 80) return "bg-amber-500/50";
    if (rate >= 70) return "bg-orange-500/50";
    return "bg-destructive/50";
}

const DAY_HEADERS = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];

// ─── Types ────────────────────────────────────────────────────────────────

export interface CalendarDayData {
    date: string; // YYYY-MM-DD
    rate: number;
}

// ─── Component ────────────────────────────────────────────────────────────

interface DailyCalendarHeatmapProps {
    data: CalendarDayData[];
    termLabel?: string;
}

export function DailyCalendarHeatmap({ data, termLabel }: DailyCalendarHeatmapProps) {
    if (!data.length) {
        return (
            <p className="text-muted-foreground py-8 text-center text-sm">
                No daily attendance data available.
            </p>
        );
    }

    const rateMap = new Map<string, number>();
    let minDate: Date | null = null;
    let maxDate: Date | null = null;

    for (const d of data) {
        rateMap.set(d.date, d.rate);
        const parsed = parseISO(d.date);
        if (!minDate || parsed < minDate) minDate = parsed;
        if (!maxDate || parsed > maxDate) maxDate = parsed;
    }

    if (!minDate || !maxDate) {
        return (
            <p className="text-muted-foreground py-8 text-center text-sm">
                No date range available.
            </p>
        );
    }

    // Split into months
    const months: {
        label: string;
        days: { day: number; dateStr: string; rate: number | null }[];
    }[] = [];
    const range = eachDayOfInterval({ start: minDate, end: maxDate });
    const monthMap = new Map<string, (typeof months)[0]>();

    for (const day of range) {
        const monthKey = format(day, "yyyy-MM");
        const dateStr = format(day, "yyyy-MM-dd");
        if (!monthMap.has(monthKey)) {
            monthMap.set(monthKey, {
                label: format(day, "MMM yyyy"),
                days: [],
            });
        }
        monthMap.get(monthKey)!.days.push({
            day: getDate(day),
            dateStr,
            rate: rateMap.get(dateStr) ?? null,
        });
    }

    for (const [, m] of monthMap) {
        months.push(m);
    }

    return (
        <div className="space-y-2">
            <div className="flex items-center justify-between">
                <p className="text-foreground text-sm font-medium">Daily Attendance Calendar</p>
                {termLabel && <p className="text-muted-foreground text-xs">{termLabel}</p>}
            </div>

            <div className="flex flex-wrap gap-6">
                {months.map((month) => {
                    const firstDay = getDay(parseISO(`${month.days[0].dateStr}`));

                    return (
                        <div key={month.label} className="space-y-1">
                            <p className="text-muted-foreground text-xs font-medium">
                                {month.label}
                            </p>
                            <div className="grid grid-cols-7 gap-1">
                                {DAY_HEADERS.map((h) => (
                                    <div
                                        key={h}
                                        className="text-muted-foreground h-6 w-8 text-center text-[10px]"
                                    >
                                        {h[0]}
                                    </div>
                                ))}
                                {/* Empty cells before first day */}
                                {Array.from({ length: firstDay }).map((_, i) => (
                                    <div key={`empty-${i}`} className="h-6 w-8" />
                                ))}
                                {month.days.map((day) => (
                                    <div
                                        key={day.dateStr}
                                        className={cn(
                                            "flex h-6 w-8 items-center justify-center rounded text-[10px]",
                                            day.rate !== null ? heatColor(day.rate) : "bg-muted/20"
                                        )}
                                        title={
                                            day.rate !== null
                                                ? `${day.dateStr}: ${day.rate.toFixed(1)}%`
                                                : `${day.dateStr}: No data`
                                        }
                                    >
                                        <span
                                            className={cn(
                                                "tabular-nums",
                                                day.rate !== null && day.rate < 70
                                                    ? "text-destructive-foreground"
                                                    : "text-foreground"
                                            )}
                                        >
                                            {day.day}
                                        </span>
                                    </div>
                                ))}
                            </div>
                        </div>
                    );
                })}
            </div>

            {/* Legend */}
            <div className="flex items-center gap-3 pt-1">
                <span className="text-muted-foreground text-xs">Rate:</span>
                {[
                    { label: "≥95%", cls: "bg-emerald-500/90" },
                    { label: "≥90%", cls: "bg-emerald-500/65" },
                    { label: "≥85%", cls: "bg-emerald-500/40" },
                    { label: "≥80%", cls: "bg-amber-500/50" },
                    { label: "≥70%", cls: "bg-orange-500/50" },
                    { label: "<70%", cls: "bg-destructive/50" },
                ].map((entry) => (
                    <div key={entry.label} className="flex items-center gap-1">
                        <div className={cn("h-3 w-3 rounded", entry.cls)} />
                        <span className="text-muted-foreground text-xs">{entry.label}</span>
                    </div>
                ))}
            </div>
        </div>
    );
}

// ─── Skeleton ─────────────────────────────────────────────────────────────

export function DailyCalendarHeatmapSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-56 animate-pulse rounded" />
            <div className="bg-muted h-48 w-full animate-pulse rounded" />
        </div>
    );
}
