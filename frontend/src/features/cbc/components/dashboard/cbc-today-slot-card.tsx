"use client";

import * as React from "react";
import { Clock, CheckCircle2, AlertCircle } from "lucide-react";

import { cn } from "@/lib/utils";
import type {
    DashboardSlotCard as DashboardSlotCardType,
    AttendanceStatus,
    AttendanceStudentRow,
    OfflineAttendanceEntry,
} from "@/features/cbc/types";
import { CbcInlineRegister } from "./cbc-inline-register";

// ─── Status config for the card badge ──────────────────────────────────────

interface BadgeConfig {
    icon: React.ReactNode;
    label: string;
    className: string;
    leftBorder: string;
}

function getBadgeConfig(status: DashboardSlotCardType["status"]): BadgeConfig {
    switch (status) {
        case "ongoing":
            return {
                icon: <Clock className="size-3.5" />,
                label: "Ongoing",
                className: "bg-amber-50 text-amber-700 border-amber-200",
                leftBorder: "border-l-amber-400",
            };
        case "done":
            return {
                icon: <CheckCircle2 className="size-3.5" />,
                label: "Done",
                className: "bg-green-50 text-green-700 border-green-200",
                leftBorder: "border-l-green-500",
            };
        case "incomplete":
            return {
                icon: <AlertCircle className="size-3.5" />,
                label: "Incomplete",
                className: "bg-amber-50 text-amber-700 border-amber-200",
                leftBorder: "border-l-amber-400",
            };
        case "upcoming":
            return {
                icon: <Clock className="size-3.5" />,
                label: "Upcoming",
                className: "bg-gray-50 text-gray-400 border-gray-200",
                leftBorder: "border-l-gray-200",
            };
        case "past_not_taken":
            return {
                icon: <AlertCircle className="size-3.5" />,
                label: "Past — not taken",
                className: "bg-red-50 text-red-700 border-red-200",
                leftBorder: "border-l-red-500",
            };
    }
}

// ─── Props ────────────────────────────────────────────────────────────────

interface CbcTodaySlotCardProps {
    slot: DashboardSlotCardType;
    /** Students for the register (fetched per-slot when expanded). */
    students: AttendanceStudentRow[];
    /** Whether student data is being loaded. */
    studentsLoading: boolean;
    /** Period ID for the register (null if no attendance started). */
    attendancePeriodId: string | null;
    localQueue: OfflineAttendanceEntry[];
    savingStudentIds: Set<string>;
    onSelectStatus: (studentId: string, status: AttendanceStatus, periodId: string) => void;
    onMarkAllPresent: () => void;
    onStartAttendance: () => void;
    /** Whether this card should be visually prominent (the "Ongoing" slot). */
    isPrimary: boolean;
}

// ─── Component ────────────────────────────────────────────────────────────

export function CbcTodaySlotCard({
    slot,
    students,
    studentsLoading,
    attendancePeriodId,
    localQueue,
    savingStudentIds,
    onSelectStatus,
    onMarkAllPresent,
    onStartAttendance,
    isPrimary,
}: CbcTodaySlotCardProps) {
    const [expanded, setExpanded] = React.useState(isPrimary);
    const badge = getBadgeConfig(slot.status);

    const allMarked = students.length > 0 && students.every((s) => s.status !== null);

    // Auto-collapse when all marked
    const prevAllMarked = React.useRef(allMarked);
    React.useEffect(() => {
        if (allMarked && !prevAllMarked.current && expanded) {
            // Brief delay to show the success state
            const timer = setTimeout(() => {
                setExpanded(false);
            }, 800);
            return () => clearTimeout(timer);
        }
        prevAllMarked.current = allMarked;
    }, [allMarked, expanded]);

    // ── Card is not actionable yet (upcoming) ────────────────────────
    const isUpcoming = slot.status === "upcoming";

    return (
        <div
            className={cn(
                "overflow-hidden rounded-lg border transition-all",
                badge.leftBorder,
                "border-l-4",
                isPrimary && "shadow-sm",
                isUpcoming && "opacity-60"
            )}
        >
            {/* Card header — tap to expand/collapse */}
            <button
                type="button"
                onClick={() => {
                    if (!isUpcoming) setExpanded((prev) => !prev);
                }}
                disabled={isUpcoming}
                className={cn(
                    "flex w-full items-center gap-3 px-4 py-3 text-left transition-colors",
                    !isUpcoming && "hover:bg-gray-50",
                    expanded && "border-b bg-gray-50/50"
                )}
            >
                {/* Time range */}
                <div className="shrink-0 text-center">
                    <p className="text-xs font-medium">{slot.start_time}</p>
                    <p className="text-muted-foreground text-[10px]">{slot.end_time}</p>
                </div>

                {/* Details */}
                <div className="min-w-0 flex-1">
                    <p className="truncate text-sm font-medium">{slot.learning_area_name}</p>
                    <p className="text-muted-foreground truncate text-xs">
                        {slot.class_name} · {slot.start_time}–{slot.end_time}
                    </p>
                </div>

                {/* Status badge */}
                <span
                    className={cn(
                        "inline-flex items-center gap-1 rounded-full border px-2.5 py-1 text-[10px] font-medium whitespace-nowrap",
                        badge.className
                    )}
                >
                    {badge.icon}
                    {badge.label}
                </span>

                {/* Expand indicator */}
                {!isUpcoming && (
                    <span className="text-muted-foreground text-xs transition-transform">
                        {expanded ? "▲" : "▼"}
                    </span>
                )}
            </button>

            {/* Expanded register area */}
            {expanded && (
                <div>
                    {/* Substitute note */}
                    {!slot.is_usual_teacher && (
                        <div className="bg-blue-50 px-4 py-1.5">
                            <p className="text-[10px] text-blue-600">
                                You&apos;re not the usual teacher for this class — covering today?
                            </p>
                        </div>
                    )}

                    {/* No attendance period started yet */}
                    {!attendancePeriodId && slot.status !== "done" && (
                        <div className="flex items-center justify-center border-b px-4 py-3">
                            <button
                                type="button"
                                onClick={onStartAttendance}
                                className="inline-flex items-center gap-1.5 rounded-md border bg-white px-3 py-1.5 text-xs font-medium text-teal-600 transition-colors hover:bg-teal-50"
                            >
                                <Clock className="size-3.5" />
                                Take attendance now
                            </button>
                        </div>
                    )}

                    {/* Inline register */}
                    {attendancePeriodId && (
                        <div>
                            {studentsLoading ? (
                                <div className="space-y-1 px-4 py-2">
                                    {Array.from({ length: 4 }).map((_, i) => (
                                        <div
                                            key={i}
                                            className="bg-muted h-8 animate-pulse rounded"
                                        />
                                    ))}
                                </div>
                            ) : (
                                <CbcInlineRegister
                                    students={students}
                                    periodId={attendancePeriodId}
                                    localQueue={localQueue}
                                    savingStudentIds={savingStudentIds}
                                    onSelectStatus={onSelectStatus}
                                    onMarkAllPresent={onMarkAllPresent}
                                    allMarked={allMarked}
                                />
                            )}
                        </div>
                    )}
                </div>
            )}
        </div>
    );
}
