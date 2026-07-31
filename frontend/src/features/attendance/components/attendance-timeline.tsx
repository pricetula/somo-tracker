"use client";

import { format, isSameDay, isAfter, isBefore, parse } from "date-fns";
import { ClassCombobox } from "@/features/classes/components/class-combobox";
import { DatePicker } from "@/components/ui/date-picker";
import { useAcademicYears } from "@/features/academic-terms/hooks/use-academic-terms";
import { useEnrichedSlotList } from "@/features/timetable-structure/hooks/use-timetable-structure";
import { Skeleton } from "@/components/ui/skeleton";
import { type EnrichedSlot } from "@/lib/api/timetable-structure";
import * as React from "react";

type SlotStatus = "active" | "missed" | "completed" | "upcoming" | "break";
function getSlotStatus(slot: EnrichedSlot, now: Date, s: string): SlotStatus {
    if (slot.is_break) return "break";

    // session_status = "SUBMITTED" or "SKIPPED" means attendance was taken
    if (slot.session_status) return "completed";

    const selected = new Date(s);
    const start = parse(slot.start_time, "HH:mm", now);
    const end = parse(slot.end_time, "HH:mm", now);

    if (isSameDay(selected, now) && isAfter(now, start) && isBefore(now, end)) return "active";
    else if ((isSameDay(selected, now) && isAfter(start, now)) || isAfter(selected, now))
        return "upcoming";

    return "missed";
}
function todayDateString(): string {
    return format(new Date(), "yyyy-MM-dd");
}

import { TimelineItem } from "./timeline-item";

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
