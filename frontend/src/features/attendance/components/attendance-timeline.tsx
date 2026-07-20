/**
 * AttendanceTimeline — scrollable timeline of timetable slots for a class on a
 * selected date. Defaults to today, with a DatePicker to browse other dates.
 *
 * Each slot shows:
 *   - Teacher name
 *   - Start / end time
 *   - Learning area / period name
 *   - Status badge: missed, active (now), upcoming, or completed
 *
 * Active slots show a "Mark Attendance" CTA to open the marking grid.
 *
 * This component makes a single API call: GET /api/v1/timetable/slots/enriched
 * with the `date` parameter. The backend filters to the matching day-of-week
 * and LEFT JOINs attendance_sessions to include session_status / skip_reason.
 */
"use client";

import * as React from "react";
import { format } from "date-fns";
import Link from "next/link";
import { Clock, PlayCircle, CheckCircle2, AlertCircle, Coffee } from "lucide-react";

import { ClassCombobox } from "@/features/classes/components/class-combobox";
import { DatePicker } from "@/components/ui/date-picker";
import { useAcademicYears } from "@/features/academic-terms/hooks/use-academic-terms";
import { useEnrichedSlotList } from "@/features/timetable-structure/hooks/use-timetable-structure";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import type { EnrichedSlot } from "@/lib/api/timetable-structure";

// ─── Helpers ──────────────────────────────────────────────────────────────

type SlotStatus = "active" | "missed" | "completed" | "upcoming" | "break";

function getSlotStatus(slot: EnrichedSlot, now: Date, selectedDate: string): SlotStatus {
    if (slot.is_break) return "break";

    // session_status = "SUBMITTED" or "SKIPPED" means attendance was taken
    if (slot.session_status) return "completed";

    const todayStr = now.toISOString().slice(0, 10);

    // Future date → all upcoming
    if (selectedDate > todayStr) return "upcoming";

    // Past date → missed (no session means attendance wasn't taken)
    if (selectedDate < todayStr) return "missed";

    // Today → calculate based on current time
    const start = new Date(`${todayStr}T${slot.start_time}`);
    const end = new Date(`${todayStr}T${slot.end_time}`);

    if (now >= start && now <= end) return "active";
    if (now > end) return "missed";
    return "upcoming";
}

function formatTime(timeStr: string) {
    // Convert "HH:MM:SS" or "HH:MM" to 12-hour format
    try {
        return format(new Date(`2000-01-01T${timeStr}`), "h:mm a");
    } catch {
        return timeStr;
    }
}

function todayDateString(): string {
    return format(new Date(), "yyyy-MM-dd");
}

// ─── Status config ────────────────────────────────────────────────────────

const statusConfig: Record<SlotStatus, { icon: React.ReactNode; label: string; color: string }> = {
    active: {
        icon: <PlayCircle className="size-4" />,
        label: "Active",
        color: "text-emerald-600",
    },
    missed: {
        icon: <AlertCircle className="size-4" />,
        label: "Missed",
        color: "text-destructive",
    },
    completed: {
        icon: <CheckCircle2 className="size-4" />,
        label: "Completed",
        color: "text-blue-600",
    },
    upcoming: {
        icon: <Clock className="size-4" />,
        label: "Upcoming",
        color: "text-amber-600",
    },
    break: {
        icon: <Coffee className="size-4" />,
        label: "Break",
        color: "text-muted-foreground",
    },
};

// ─── Timeline Item ────────────────────────────────────────────────────────

function TimelineItem({
    slot,
    status,
    date,
}: {
    slot: EnrichedSlot;
    status: SlotStatus;
    date: string;
}) {
    const cfg = statusConfig[status];

    if (status === "break") {
        return (
            <div className="bg-muted/20 text-muted-foreground flex items-center gap-3 px-4 py-2 text-sm">
                <Coffee className="size-4 shrink-0" />
                <span className="font-medium">{slot.period_name}</span>
                <span className="text-xs">
                    {formatTime(slot.start_time)} – {formatTime(slot.end_time)}
                </span>
            </div>
        );
    }

    return (
        <div
            className={`group relative border-l-2 px-4 py-3 ${
                status === "active"
                    ? "border-l-green-500 bg-green-600/5"
                    : status === "missed"
                      ? "border-l-red-500 bg-red-600/5"
                      : status === "completed"
                        ? "border-l-blue-500 bg-blue-600/5"
                        : status === "upcoming"
                          ? "border-l-amber-500 bg-amber-600/5"
                          : "bg-muted/30 border-l-transparent"
            }`}
        >
            <div className="flex items-start justify-between gap-4">
                {/* Left: slot info */}
                <div className="flex flex-col gap-0.5">
                    <div className="flex items-center gap-2">
                        <span className="text-foreground font-medium">{slot.period_name}</span>
                        <span
                            className={`flex items-center gap-1 text-xs font-medium ${cfg.color}`}
                        >
                            {cfg.icon}
                            {cfg.label}
                        </span>
                    </div>
                    <span className="text-muted-foreground text-xs">
                        {formatTime(slot.start_time)} – {formatTime(slot.end_time)}
                    </span>
                    {slot.teacher_name && (
                        <span className="text-muted-foreground text-xs">
                            {slot.teacher_name}
                            {slot.learning_area_name && ` · ${slot.learning_area_name}`}
                        </span>
                    )}
                </div>

                {/* Right: CTA for active / missed slots */}
                {(status === "active" || status === "missed") && (
                    <Button asChild variant="outline" size="sm" className="shrink-0">
                        <Link href={`/attendance/mark/${slot.id}/${date}`}>Mark Attendance</Link>
                    </Button>
                )}

                {status === "completed" && (
                    <Badge variant="secondary" className="shrink-0 text-xs">
                        Done
                    </Badge>
                )}
            </div>
        </div>
    );
}

// ─── Main Component ───────────────────────────────────────────────────────

export function AttendanceTimeline() {
    const [classId, setClassId] = React.useState<string>("");
    const [selectedDate, setSelectedDate] = React.useState<string>(todayDateString());

    // Fetch academic years to find the current one
    const { data: yearsData } = useAcademicYears();

    // Derive the academic year ID from years data (auto-select first year once loaded)
    const academicYearId = React.useMemo(() => {
        if (yearsData?.items?.length) {
            return yearsData.items[0].id;
        }
        return "";
    }, [yearsData]);

    // Single query: enriched slots for this class + date.
    // The backend filters by day-of-week matching the date and includes
    // session_status / skip_reason from the attendance_sessions table.
    const {
        data: slotsData,
        isLoading,
        isError,
    } = useEnrichedSlotList(academicYearId, classId ? { classId, date: selectedDate } : undefined);

    // Slots are already filtered to the correct day-of-week by the backend.
    const allSlots = React.useMemo(() => {
        if (!slotsData?.items) return [];
        return [...slotsData.items].sort((a, b) => a.start_time.localeCompare(b.start_time));
    }, [slotsData]);

    // Current time for status calculation
    const now = React.useMemo(() => new Date(), []);

    // Auto-select first class when classes load
    const handleClassChange = React.useCallback((val: string | string[]) => {
        setClassId(val as string);
    }, []);

    // ── Loading state ────────────────────────────────────────────────────
    if (isLoading || !academicYearId) {
        return (
            <div className="space-y-4">
                <Skeleton className="h-9 w-full max-w-xs" />
                <div className="space-y-3">
                    {Array.from({ length: 5 }).map((_, i) => (
                        <Skeleton key={i} className="h-16 w-full" />
                    ))}
                </div>
            </div>
        );
    }

    // ── Error state ──────────────────────────────────────────────────────
    if (isError) {
        return (
            <div className="bg-destructive/10 text-destructive p-4 text-sm">
                Failed to load timetable. Please try again.
            </div>
        );
    }

    // ── No slots for date ────────────────────────────────────────────────
    if (allSlots.length === 0) {
        return (
            <div className="space-y-4">
                <div className="flex flex-wrap items-center gap-3">
                    <div className="w-full max-w-xs">
                        <ClassCombobox
                            value={classId}
                            onChange={handleClassChange}
                            placeholder="Select a class..."
                            doPreselectFirstOption
                        />
                    </div>
                    <DatePicker value={selectedDate} onChange={setSelectedDate} />
                </div>
                <div className="text-muted-foreground py-12 text-center text-sm">
                    No timetable slots scheduled for{" "}
                    {format(new Date(selectedDate + "T00:00:00"), "EEEE, MMMM d, yyyy")}.
                </div>
            </div>
        );
    }

    // ── Render ───────────────────────────────────────────────────────────
    return (
        <div className="space-y-4">
            <div className="flex max-w-50 items-center gap-3">
                <ClassCombobox
                    value={classId}
                    onChange={handleClassChange}
                    placeholder="Select a class..."
                    doPreselectFirstOption
                />
                <DatePicker value={selectedDate} onChange={setSelectedDate} />
            </div>

            {/* Timeline */}
            <div className="max-h-[calc(100vh-14rem)] space-y-1 pr-2">
                {allSlots.map((slot, idx) => {
                    const status = getSlotStatus(slot, now, selectedDate);

                    // Show a "now" divider only when viewing today
                    const isToday = selectedDate === now.toISOString().slice(0, 10);
                    const showNowDivider =
                        isToday &&
                        status === "active" &&
                        (idx === 0 ||
                            getSlotStatus(allSlots[idx - 1], now, selectedDate) !== "active");

                    return (
                        <React.Fragment key={slot.id}>
                            {showNowDivider && (
                                <div className="flex items-center gap-2 py-1">
                                    <div className="h-px flex-1 bg-emerald-300" />
                                    <span className="text-xs font-medium text-emerald-600">
                                        NOW
                                    </span>
                                    <div className="h-px flex-1 bg-emerald-300" />
                                </div>
                            )}
                            <TimelineItem slot={slot} status={status} date={selectedDate} />
                        </React.Fragment>
                    );
                })}
            </div>
        </div>
    );
}
