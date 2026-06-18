"use client";

import * as React from "react";

import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import {
    Select,
    SelectTrigger,
    SelectValue,
    SelectContent,
    SelectItem,
} from "@/components/ui/select";

import { CbcTimetableGrid } from "./cbc-timetable-grid";
import { CbcSlotEditorSidePanel } from "./cbc-slot-editor-side-panel";
import { CbcBulkActions } from "./cbc-bulk-actions";
import {
    useCbcTimetableSlots,
    useCbcTeachers,
    useOperatingDays,
} from "@/features/cbc/hooks/use-cbc-timetable";
import type { CbcTimetableSlot } from "@/features/cbc/types";

// ─── Props ────────────────────────────────────────────────────────────────

interface CbcTimetablePageProps {
    classId: string;
    schoolId: string;
    academicYearId: string;
    gradeId: string;
    className: string;
    hasClassTeachers: boolean;
}

// ─── Constants ────────────────────────────────────────────────────────────

// ─── Component ────────────────────────────────────────────────────────────

export function CbcTimetablePage({
    classId,
    schoolId,
    academicYearId,
    gradeId,
    className,
    hasClassTeachers,
}: CbcTimetablePageProps) {
    // ── State ────────────────────────────────────────────────────────────
    const [editingSlot, setEditingSlot] = React.useState<CbcTimetableSlot | null>(null);
    const [isAddingSlot, setIsAddingSlot] = React.useState(false);
    const [addingDay, setAddingDay] = React.useState<number>(1);
    const [addingTime, setAddingTime] = React.useState<string | undefined>(undefined);
    const [teacherFilter, setTeacherFilter] = React.useState<string>("");
    const [selectedAcademicYear, setSelectedAcademicYear] = React.useState(academicYearId);

    // ── Data ─────────────────────────────────────────────────────────────
    const {
        data: slots = [],
        isLoading: slotsLoading,
        error: slotsError,
    } = useCbcTimetableSlots(classId);

    const { data: operatingDays = [] } = useOperatingDays(schoolId);

    const { data: allTeachers = [] } = useCbcTeachers(schoolId);

    // ── Filtered slots by teacher search ─────────────────────────────────
    const filteredSlots = React.useMemo(() => {
        if (!teacherFilter) return slots;
        const lower = teacherFilter.toLowerCase();
        return slots.filter((slot) => {
            const teacher = allTeachers.find((t) => t.id === slot.teacher_id);
            return teacher?.name.toLowerCase().includes(lower);
        });
    }, [slots, teacherFilter, allTeachers]);

    // ── Slots grouped by day ────────────────────────────────────────────
    const slotsByDay = React.useMemo(() => {
        const map = new Map<number, CbcTimetableSlot[]>();
        for (const day of operatingDays) {
            map.set(day.value, []);
        }
        for (const slot of filteredSlots) {
            const existing = map.get(slot.day_of_week) ?? [];
            existing.push(slot);
            map.set(slot.day_of_week, existing);
        }
        return map;
    }, [filteredSlots, operatingDays]);

    // ── Slot count per day ──────────────────────────────────────────────
    const dayCounts = React.useMemo(() => {
        const counts: Record<number, number> = {};
        for (const slot of slots) {
            counts[slot.day_of_week] = (counts[slot.day_of_week] ?? 0) + 1;
        }
        return counts;
    }, [slots]);

    // ── Handlers ─────────────────────────────────────────────────────────
    const handleSlotClick = React.useCallback((slot: CbcTimetableSlot) => {
        setEditingSlot(slot);
        setIsAddingSlot(false);
    }, []);

    const handleEmptyCellClick = React.useCallback((day: number, inferredTime?: string) => {
        setAddingDay(day);
        setAddingTime(inferredTime);
        setEditingSlot(null);
        setIsAddingSlot(true);
    }, []);

    const handleClosePanel = React.useCallback(() => {
        setEditingSlot(null);
        setIsAddingSlot(false);
        setAddingTime(undefined);
    }, []);

    const handleSlotSaved = React.useCallback(() => {
        handleClosePanel();
    }, [handleClosePanel]);

    // ── Render ───────────────────────────────────────────────────────────
    return (
        <div className="flex flex-1 flex-col gap-4">
            {/* ── Header ───────────────────────────────────────────── */}

            {/* No current academic year banner */}
            {!academicYearId && (
                <div
                    role="alert"
                    className="rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800"
                >
                    No current academic year set. Set one in School Settings before building a
                    timetable.
                </div>
            )}

            <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                    <h2 className="text-lg font-medium tracking-tight">{className} — timetable</h2>
                    {!slotsLoading && slots.length > 0 && (
                        <span className="border-border/40 text-muted-foreground inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium">
                            {slots.length} slots
                        </span>
                    )}
                </div>

                <div className="flex items-center gap-2">
                    {/* Academic year selector */}
                    <Select value={selectedAcademicYear} onValueChange={setSelectedAcademicYear}>
                        <SelectTrigger className="h-8 w-44 text-xs">
                            <SelectValue placeholder="Academic year" />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectItem value={academicYearId}>Current year</SelectItem>
                        </SelectContent>
                    </Select>

                    {/* Teacher search filter */}
                    <div className="relative">
                        <input
                            type="text"
                            placeholder="Filter by teacher..."
                            value={teacherFilter}
                            onChange={(e) => setTeacherFilter(e.target.value)}
                            className="border-input text-foreground placeholder:text-muted-foreground h-8 w-48 rounded-md border bg-transparent px-2.5 py-1 text-xs outline-none focus:border-teal-500 focus:ring-1 focus:ring-teal-500"
                            aria-label="Filter timetable by teacher name"
                        />
                    </div>

                    {/* Bulk actions */}
                    <CbcBulkActions
                        classId={classId}
                        academicYearId={academicYearId}
                        operatingDays={operatingDays}
                        slots={slots}
                    />
                </div>
            </div>

            {/* ── No teachers hint ───────────────────────────────────── */}
            {!hasClassTeachers && slots.length === 0 && (
                <p className="text-muted-foreground text-xs">
                    No teachers assigned to this class yet — add them in Class Settings to speed
                    this up next time.
                </p>
            )}

            {/* ── Loading state ──────────────────────────────────────── */}
            {slotsLoading && (
                <div className="flex flex-col gap-2">
                    <Skeleton className="h-8 w-full" />
                    <Skeleton className="h-64 w-full" />
                </div>
            )}

            {/* ── Error state ────────────────────────────────────────── */}
            {slotsError && !slotsLoading && (
                <Card className="flex items-center justify-center p-8">
                    <div className="text-center">
                        <p className="text-destructive text-sm font-medium">
                            Failed to load timetable
                        </p>
                        <p className="text-muted-foreground mt-1 text-xs">
                            {(slotsError as Error).message}
                        </p>
                        <Button
                            variant="outline"
                            size="sm"
                            className="mt-3"
                            onClick={() => window.location.reload()}
                        >
                            Retry
                        </Button>
                    </div>
                </Card>
            )}

            {/* ── Empty state ────────────────────────────────────────── */}
            {!slotsLoading && !slotsError && slots.length === 0 && (
                <Card className="flex items-center justify-center p-8">
                    <div className="text-center">
                        <h3 className="text-lg font-medium">No timetable slots yet</h3>
                        <p className="text-muted-foreground mt-1 text-sm">
                            Click a time slot in the grid to add your first lesson.
                        </p>
                    </div>
                </Card>
            )}

            {/* ── Grid ───────────────────────────────────────────────── */}
            {!slotsLoading && !slotsError && (
                <CbcTimetableGrid
                    slotsByDay={slotsByDay}
                    operatingDays={operatingDays}
                    dayCounts={dayCounts}
                    allTeachers={allTeachers}
                    onSlotClick={handleSlotClick}
                    onEmptyCellClick={handleEmptyCellClick}
                    onDragEnd={handleDragEnd}
                />
            )}

            {/* ── Side Panel (Add / Edit) ────────────────────────────── */}
            {(isAddingSlot || editingSlot) && (
                <CbcSlotEditorSidePanel
                    classId={classId}
                    schoolId={schoolId}
                    academicYearId={academicYearId}
                    gradeId={gradeId}
                    operatingDays={operatingDays}
                    existingSlot={editingSlot}
                    defaultDay={isAddingSlot ? addingDay : undefined}
                    defaultStartTime={addingTime}
                    hasClassTeachers={hasClassTeachers}
                    onSaved={handleSlotSaved}
                    onClose={handleClosePanel}
                />
            )}
        </div>
    );
}
