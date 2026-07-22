/**
 * AttendanceCalendar — month-grid calendar with color-coded daily attendance
 * for a class.
 *
 * Each cell is tinted based on the class-level attendance rate for that day:
 *   🟢 High (≥95% present)      → green background tint
 *   🟡 Medium (≥50% present)    → amber background tint
 *   🔴 Low (<50% present)       → red background tint
 *   ⚪ No data                  → default
 *
 * Clicking a day with data fires `onDayClick` so the parent can navigate to a
 * detail view or filter a timeline.
 *
 * Accepts data in two formats:
 *
 * **1. Full rollup** — `dailySummaries` map of `YYYY-MM-DD` → counts:
 * ```ts
 * {
 *   "2026-07-14": { present: 22, absent: 1, late: 2, excused: 0, total: 25 },
 * }
 * ```
 *
 * **2. Rate map** — `attendanceRateMap` map of `YYYY-MM-DD` → percentage (0–100):
 * ```ts
 * {
 *   "2026-07-14": 96.0,
 * }
 * ```
 * Useful when the parent only has aggregated rate data (e.g. from analytics).
 */
"use client";

import * as React from "react";
import { format } from "date-fns";

import { Calendar } from "@/components/shared/calendar";
import { cn } from "@/lib/utils";

// ─── Types ────────────────────────────────────────────────────────────────

export interface DaySummary {
    present: number;
    absent: number;
    late: number;
    excused: number;
    total: number;
}

export interface AttendanceCalendarProps {
    /** Map of "YYYY-MM-DD" → daily rollup data (present/absent/late/excused counts). */
    dailySummaries?: Record<string, DaySummary>;
    /**
     * Map of "YYYY-MM-DD" → attendance rate (0–100).
     * Simpler alternative to `dailySummaries` when only the rate is available.
     */
    attendanceRateMap?: Record<string, number>;
    /** Called when a day with attendance data is clicked. Receives the date string. */
    onDayClick?: (dateStr: string) => void;
    /** Optional class name for the outer wrapper. */
    className?: string;
}

// ─── Attendance rate helpers ──────────────────────────────────────────────

type AttendanceTier = "high" | "medium" | "low" | "no-data";

function getDayRate(
    date: Date,
    summaries: Record<string, DaySummary>,
    rateMap: Record<string, number>
): number | null {
    const key = format(date, "yyyy-MM-dd");

    // Prefer full summary for precise rate
    const summary = summaries[key];
    if (summary && summary.total > 0) {
        return (summary.present / summary.total) * 100;
    }

    // Fall back to rate map
    const rate = rateMap[key];
    if (rate !== undefined && rate !== null) return rate;

    return null;
}

function getDayTier(
    date: Date,
    summaries: Record<string, DaySummary>,
    rateMap: Record<string, number>
): AttendanceTier {
    const rate = getDayRate(date, summaries, rateMap);
    if (rate === null) return "no-data";

    if (rate >= 95) return "high";
    if (rate >= 50) return "medium";
    return "low";
}

// ─── Tier styling ─────────────────────────────────────────────────────────

const tierIndicatorClass: Record<AttendanceTier, string> = {
    high: "bg-emerald-500",
    medium: "bg-amber-400",
    low: "bg-red-500",
    "no-data": "bg-transparent",
};

const tierCellBg: Record<AttendanceTier, string> = {
    high: "bg-emerald-50 dark:bg-emerald-950/30",
    medium: "bg-amber-50 dark:bg-amber-950/30",
    low: "bg-red-50 dark:bg-red-950/30",
    "no-data": "",
};

// ─── Component ────────────────────────────────────────────────────────────

export function AttendanceCalendar({
    dailySummaries = {},
    attendanceRateMap = {},
    onDayClick,
    className,
}: AttendanceCalendarProps) {
    // Merge both data sources into a single internal cache for lookups and
    // click detection.
    const hasData = React.useMemo(() => {
        const keys = new Set<string>();
        for (const k of Object.keys(dailySummaries)) keys.add(k);
        for (const k of Object.keys(attendanceRateMap)) keys.add(k);
        return keys;
    }, [dailySummaries, attendanceRateMap]);

    const handleDayClick = React.useCallback(
        (day: Date) => {
            const key = format(day, "yyyy-MM-dd");
            if (hasData.has(key) && onDayClick) {
                onDayClick(key);
            }
        },
        [hasData, onDayClick]
    );

    // Render a small colored bar below the day number to indicate attendance
    const renderDayContent = React.useCallback(
        (date: Date) => {
            const tier = getDayTier(date, dailySummaries, attendanceRateMap);
            if (tier === "no-data") return null;

            return (
                <span
                    className={cn("mt-0.5 h-1 w-4 shrink-0 rounded-full", tierIndicatorClass[tier])}
                    aria-hidden="true"
                />
            );
        },
        [dailySummaries, attendanceRateMap]
    );

    // Compute a set of modifiers so we can tint the cell backgrounds
    const modifierDates = React.useMemo(() => {
        const high: Date[] = [];
        const medium: Date[] = [];
        const low: Date[] = [];
        const noData: Date[] = [];

        const allKeys = new Set([
            ...Object.keys(dailySummaries),
            ...Object.keys(attendanceRateMap),
        ]);
        for (const key of allKeys) {
            const d = new Date(key + "T00:00:00");
            const tier = getDayTier(d, dailySummaries, attendanceRateMap);
            const bucket = { high, medium, low, "no-data": noData };
            bucket[tier].push(d);
        }

        return { high, medium, low };
    }, [dailySummaries, attendanceRateMap]);

    return (
        <div className={cn("w-fit", className)}>
            <Calendar
                dayContent={renderDayContent}
                onDayClick={handleDayClick}
                modifiers={modifierDates}
                modifiersClassNames={{
                    high: tierCellBg["high"],
                    medium: tierCellBg["medium"],
                    low: tierCellBg["low"],
                }}
            />

            {/* Legend */}
            <div className="text-muted-foreground mt-3 flex items-center gap-4 px-1 text-xs">
                <LegendDot color="bg-emerald-500" label="≥95%" />
                <LegendDot color="bg-amber-400" label="≥50%" />
                <LegendDot color="bg-red-500" label="<50%" />
            </div>
        </div>
    );
}

// ─── Legend Dot ───────────────────────────────────────────────────────────

function LegendDot({ color, label }: { color: string; label: string }) {
    return (
        <span className="flex items-center gap-1.5">
            <span className={cn("inline-block size-2 rounded-full", color)} aria-hidden="true" />
            {label}
        </span>
    );
}
