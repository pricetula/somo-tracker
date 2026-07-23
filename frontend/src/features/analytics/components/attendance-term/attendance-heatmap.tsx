/**
 * AttendanceHeatmap — Day × Slot attendance pattern heatmap.
 *
 * Visualisation: Attendance patterns by day of week and time slot.
 * Uses a CSS grid with colour-coded cells.
 */
"use client";

import { cn } from "@/lib/utils";
import { GraphHelp } from "@/features/analytics/components/graph-help";

// ─── Types ────────────────────────────────────────────────────────────────

export interface HeatmapCell {
    day: string; // e.g. "Mon"
    dayIndex: number; // 0-6
    slot: string; // e.g. "08:00"
    slotIndex: number;
    attendanceRate: number;
    periodsTotal: number;
}

export interface AttendanceHeatmapData {
    days: string[];
    slots: string[];
    cells: HeatmapCell[];
}

// ─── Helpers ──────────────────────────────────────────────────────────────

function cellColor(rate: number): string {
    if (rate >= 90) return "bg-emerald-500/80";
    if (rate >= 75) return "bg-emerald-500/50";
    if (rate >= 60) return "bg-amber-500/50";
    if (rate >= 40) return "bg-orange-500/50";
    return "bg-destructive/50";
}

// ─── Component ────────────────────────────────────────────────────────────

interface AttendanceHeatmapProps {
    data: AttendanceHeatmapData;
}

export function AttendanceHeatmap({ data }: AttendanceHeatmapProps) {
    const { days, slots, cells } = data;

    if (!cells.length) {
        return (
            <p className="text-muted-foreground py-8 text-center text-sm">
                No heatmap data available.
            </p>
        );
    }

    const cellMap = new Map<string, HeatmapCell>();
    for (const cell of cells) {
        cellMap.set(`${cell.dayIndex}-${cell.slotIndex}`, cell);
    }

    return (
        <div className="space-y-2">
            <p className="text-foreground text-sm font-medium">
                Attendance Pattern by Day &amp; Slot
                <GraphHelp>
                    Heatmap showing attendance patterns by day of week and time slot. Colour
                    intensity reflects attendance rate &mdash; green for high attendance, red for
                    low.
                </GraphHelp>
            </p>
            <p className="text-muted-foreground text-xs">
                Colour intensity = attendance rate. Hover for details.
            </p>

            {/* Header row — day labels */}
            <div
                className="grid gap-1"
                style={{ gridTemplateColumns: `80px repeat(${days.length}, 1fr)` }}
            >
                <div className="text-muted-foreground text-xs" />

                {days.map((day) => (
                    <div
                        key={day}
                        className="text-muted-foreground text-center text-xs font-medium"
                    >
                        {day}
                    </div>
                ))}

                {/* Data rows — one per slot */}
                {slots.map((slot, si) => (
                    <>
                        <div
                            key={`slot-${slot}`}
                            className="text-muted-foreground flex items-center text-xs"
                        >
                            {slot}
                        </div>
                        {days.map((_, di) => {
                            const cell = cellMap.get(`${di}-${si}`);
                            if (!cell) {
                                return (
                                    <div
                                        key={`${di}-${si}`}
                                        className="bg-muted/30 aspect-[3/2] rounded"
                                    />
                                );
                            }
                            return (
                                <div
                                    key={`${di}-${si}`}
                                    className={cn(
                                        "group relative aspect-[3/2] cursor-default rounded transition-colors",
                                        cellColor(cell.attendanceRate)
                                    )}
                                    title={`${cell.day} ${cell.slot}: ${cell.attendanceRate.toFixed(1)}% (${cell.periodsTotal} periods)`}
                                >
                                    <div className="bg-popover text-popover-foreground absolute inset-x-0 bottom-full z-10 mb-1 hidden rounded px-2 py-1 text-xs shadow group-hover:block">
                                        {cell.attendanceRate.toFixed(1)}%
                                    </div>
                                </div>
                            );
                        })}
                    </>
                ))}
            </div>

            {/* Legend */}
            <div className="flex items-center gap-3 pt-1">
                <span className="text-muted-foreground text-xs">Rate:</span>
                {[
                    { label: "≥90%", cls: "bg-emerald-500/80" },
                    { label: "≥75%", cls: "bg-emerald-500/50" },
                    { label: "≥60%", cls: "bg-amber-500/50" },
                    { label: "≥40%", cls: "bg-orange-500/50" },
                    { label: "<40%", cls: "bg-destructive/50" },
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

export function AttendanceHeatmapSkeleton() {
    return (
        <div className="space-y-2">
            <div className="bg-muted h-4 w-56 animate-pulse rounded" />
            <div className="bg-muted aspect-[3/1] w-full animate-pulse rounded" />
        </div>
    );
}
