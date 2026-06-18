"use client";

import * as React from "react";

import { cn } from "@/lib/utils";
import type { CbcTimetableSlot } from "@/features/cbc/types";

// ─── Props ────────────────────────────────────────────────────────────────

interface CbcSlotBlockProps {
    slot: CbcTimetableSlot;
    teacherName: string;
    top: number;
    height: number;
    isDragOverlay?: boolean;
    isConflict?: boolean;
    onClick: () => void;
    onDragStart: (e: React.DragEvent) => void;
    onDragEnd: (e: React.DragEvent) => void;
}

// ─── Helpers ──────────────────────────────────────────────────────────────

const LEARNING_AREA_COLORS = [
    "border-l-teal-500",
    "border-l-cyan-500",
    "border-l-emerald-500",
    "border-l-green-500",
    "border-l-teal-400",
    "border-l-cyan-400",
];

function hashColor(id: string): string {
    let hash = 0;
    for (let i = 0; i < id.length; i++) {
        hash = id.charCodeAt(i) + ((hash << 5) - hash);
    }
    return LEARNING_AREA_COLORS[Math.abs(hash) % LEARNING_AREA_COLORS.length];
}

// ─── Component ────────────────────────────────────────────────────────────

export const CbcSlotBlock = React.forwardRef<HTMLDivElement, CbcSlotBlockProps>(
    (
        {
            slot,
            teacherName,
            top,
            height,
            isDragOverlay = false,
            isConflict = false,
            onClick,
            onDragStart,
            onDragEnd,
        },
        ref
    ) => {
        const colorClass = hashColor(slot.cbc_learning_area_id ?? slot.id);

        return (
            <div
                ref={ref}
                data-slot-block
                draggable
                className={cn(
                    "group absolute right-1 left-1 cursor-grab rounded-md border border-teal-200 px-2 py-1 text-xs transition-shadow",
                    "hover:shadow-sm active:cursor-grabbing",
                    "border-l-[3px]",
                    colorClass,
                    isDragOverlay && "opacity-50 shadow-lg",
                    isConflict && "border-red-400 bg-red-50",
                    !slot.cbc_learning_area_id && "border-dashed border-gray-300 bg-gray-50"
                )}
                style={{
                    top,
                    height: Math.max(height, 40),
                    minHeight: 40,
                }}
                onClick={(e) => {
                    e.stopPropagation();
                    onClick();
                }}
                onDragStart={onDragStart}
                onDragEnd={onDragEnd}
                role="button"
                tabIndex={0}
                aria-label={`${teacherName}, ${slot.start_time}–${slot.end_time}`}
                onKeyDown={(e) => {
                    if (e.key === "Enter" || e.key === " ") {
                        e.preventDefault();
                        onClick();
                    }
                }}
            >
                {/* Learning area / label */}
                <div className="truncate leading-tight font-medium">
                    {slot.cbc_learning_area_id ? (
                        // Learning area name would be resolved via context
                        <span className="text-teal-800">Lesson</span>
                    ) : (
                        <span className="text-muted-foreground italic">Break / Assembly</span>
                    )}
                </div>

                {/* Teacher + Room */}
                <div className="text-muted-foreground mt-0.5 truncate leading-tight">
                    {teacherName}
                    {slot.room_identifier && ` · ${slot.room_identifier}`}
                </div>

                {/* Time */}
                <div className="text-muted-foreground mt-0.5 text-[10px] leading-tight">
                    {slot.start_time}–{slot.end_time}
                </div>

                {/* Resize handle */}
                <div
                    className="absolute right-0 bottom-0 left-0 h-2 cursor-s-resize opacity-0 transition-opacity group-hover:opacity-100"
                    onMouseDown={(e) => {
                        e.stopPropagation();
                        e.preventDefault();
                        // Resize logic handled at the grid level
                    }}
                >
                    <div className="mx-auto h-0.5 w-6 rounded-full bg-teal-300" />
                </div>
            </div>
        );
    }
);

CbcSlotBlock.displayName = "CbcSlotBlock";
