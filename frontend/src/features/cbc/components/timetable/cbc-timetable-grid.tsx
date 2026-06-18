"use client";

import * as React from "react";

import { CbcSlotBlock } from "./cbc-slot-block";
import type { CbcTimetableSlot, OperatingDay, TeacherOption } from "@/features/cbc/types";

// ─── Props ────────────────────────────────────────────────────────────────

interface CbcTimetableGridProps {
    slotsByDay: Map<number, CbcTimetableSlot[]>;
    operatingDays: OperatingDay[];
    dayCounts: Record<number, number>;
    allTeachers: TeacherOption[];
    onSlotClick: (slot: CbcTimetableSlot) => void;
    onEmptyCellClick: (day: number, inferredTime?: string) => void;
}

// ─── Constants ────────────────────────────────────────────────────────────

/** Minimum readable slot height in pixels. */
const MIN_SLOT_HEIGHT = 40;

/** Pixels per minute of time — default scale. */
const PX_PER_MINUTE = 1;

/** Grid start/end for a standard school day. */
const GRID_START = "06:00";
const GRID_END = "18:00";

// ─── Helpers ──────────────────────────────────────────────────────────────

function parseTime(time: string): number {
    const [h, m] = time.split(":").map(Number);
    return h * 60 + m;
}

function formatTime(minutes: number): string {
    const h = Math.floor(minutes / 60);
    const m = minutes % 60;
    return `${String(h).padStart(2, "0")}:${String(m).padStart(2, "0")}`;
}

function computeSlotTop(startTime: string): number {
    const startMin = parseTime(startTime);
    const gridStartMin = parseTime(GRID_START);
    return Math.max(0, (startMin - gridStartMin) * PX_PER_MINUTE);
}

function computeSlotHeight(startTime: string, endTime: string): number {
    const duration = parseTime(endTime) - parseTime(startTime);
    return Math.max(duration * PX_PER_MINUTE, MIN_SLOT_HEIGHT);
}

function snapToGrid(minutes: number): number {
    return Math.round(minutes / 5) * 5;
}

// ─── Component ────────────────────────────────────────────────────────────

export function CbcTimetableGrid({
    slotsByDay,
    operatingDays,
    dayCounts,
    allTeachers,
    onSlotClick,
    onEmptyCellClick,
}: CbcTimetableGridProps) {
    const gridRef = React.useRef<HTMLDivElement>(null);

    // ── Time axis ─────────────────────────────────────────────────────
    const timeSlots = React.useMemo(() => {
        const start = parseTime(GRID_START);
        const end = parseTime(GRID_END);
        const slots: string[] = [];
        for (let m = start; m < end; m += 60) {
            slots.push(formatTime(m));
        }
        return slots;
    }, []);

    const totalHeight = (parseTime(GRID_END) - parseTime(GRID_START)) * PX_PER_MINUTE;

    // ── Drag state ────────────────────────────────────────────────────
    const [dragOverDay, setDragOverDay] = React.useState<number | null>(null);
    const [draggedSlotId, setDraggedSlotId] = React.useState<string | null>(null);

    const handleDragStart = React.useCallback((e: React.DragEvent, slotId: string) => {
        setDraggedSlotId(slotId);
        e.dataTransfer.effectAllowed = "move";
        e.dataTransfer.setData("text/plain", slotId);
    }, []);

    const handleDragOver = React.useCallback((e: React.DragEvent, day: number) => {
        e.preventDefault();
        setDragOverDay(day);
    }, []);

    const handleDragLeave = React.useCallback(() => {
        setDragOverDay(null);
    }, []);

    const handleDrop = React.useCallback((e: React.DragEvent) => {
        e.preventDefault();
        setDragOverDay(null);
        setDraggedSlotId(null);
        // The parent handles the actual update
    }, []);

    const handleDragEndCleanup = React.useCallback(() => {
        setDragOverDay(null);
        setDraggedSlotId(null);
    }, []);

    // ── Empty cell click - infer time from position ──────────────────
    const handleCellClick = React.useCallback(
        (day: number, clientY: number) => {
            if (!gridRef.current) return;
            const rect = gridRef.current.getBoundingClientRect();
            const relativeY = clientY - rect.top;
            const minutes = Math.round(relativeY / PX_PER_MINUTE) + parseTime(GRID_START);
            const snapped = snapToGrid(minutes);
            const inferredTime = formatTime(snapped);
            onEmptyCellClick(day, inferredTime);
        },
        [onEmptyCellClick]
    );

    // ── Render ───────────────────────────────────────────────────────
    return (
        <div
            ref={gridRef}
            className="relative overflow-auto rounded-lg border"
            style={{ maxHeight: "75vh" }}
        >
            {/* Header row */}
            <div className="sticky top-0 z-10 flex bg-white shadow-sm">
                {/* Time gutter */}
                <div className="border-border flex-shrink-0 border-r" style={{ width: 56 }} />

                {/* Day columns */}
                {operatingDays.map((day) => {
                    const count = dayCounts[day.value] ?? 0;
                    const isDragTarget = dragOverDay === day.value;
                    return (
                        <div
                            key={day.value}
                            className={`border-border flex flex-1 items-center justify-center gap-1.5 border-r py-2 text-xs font-medium transition-colors last:border-r-0 ${
                                isDragTarget ? "bg-teal-50" : ""
                            }`}
                        >
                            <span>{day.short_label}</span>
                            <span className="text-muted-foreground">({count})</span>
                        </div>
                    );
                })}
            </div>

            {/* Body */}
            <div
                className="relative flex"
                style={{ height: totalHeight }}
                onDragLeave={handleDragLeave}
            >
                {/* Time gutter */}
                <div className="border-border flex-shrink-0 border-r" style={{ width: 56 }}>
                    {timeSlots.map((t) => (
                        <div
                            key={t}
                            className="text-muted-foreground flex items-start justify-end pr-2 text-[10px] leading-none"
                            style={{ height: 60, paddingTop: -4 }}
                        >
                            {t}
                        </div>
                    ))}
                </div>

                {/* Day columns */}
                {operatingDays.map((day) => {
                    const daySlots = slotsByDay.get(day.value) ?? [];
                    return (
                        <div
                            key={day.value}
                            className={`border-border relative flex-1 border-r last:border-r-0 ${
                                dragOverDay === day.value ? "bg-teal-50/50" : ""
                            }`}
                            onDragOver={(e) => handleDragOver(e, day.value)}
                            onDrop={(e) => handleDrop(e)}
                            onClick={(e) => {
                                // Only trigger on empty area (not on a slot block)
                                const target = e.target as HTMLElement;
                                if (target.closest("[data-slot-block]")) return;
                                handleCellClick(day.value, e.clientY);
                            }}
                        >
                            {/* Existing slots */}
                            {daySlots.map((slot) => {
                                const teacher = allTeachers.find((t) => t.id === slot.teacher_id);
                                return (
                                    <CbcSlotBlock
                                        key={slot.id}
                                        slot={slot}
                                        teacherName={teacher?.name ?? "Unknown"}
                                        top={computeSlotTop(slot.start_time)}
                                        height={computeSlotHeight(slot.start_time, slot.end_time)}
                                        isDragOverlay={draggedSlotId === slot.id}
                                        onClick={() => onSlotClick(slot)}
                                        onDragStart={(e) => handleDragStart(e, slot.id)}
                                        onDragEnd={handleDragEndCleanup}
                                    />
                                );
                            })}

                            {/* Empty zone indicator */}
                            {daySlots.length === 0 && (
                                <div
                                    className="absolute inset-0 flex items-center justify-center opacity-0 transition-opacity hover:opacity-100"
                                    style={{ pointerEvents: "none" }}
                                >
                                    <div className="border-dashed-muted-foreground/30 rounded-md border-2 border-dashed px-2 py-1">
                                        <span className="text-muted-foreground text-lg font-light">
                                            +
                                        </span>
                                    </div>
                                </div>
                            )}
                        </div>
                    );
                })}
            </div>
        </div>
    );
}
