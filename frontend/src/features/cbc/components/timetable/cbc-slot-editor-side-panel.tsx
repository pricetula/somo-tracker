"use client";

import * as React from "react";
import { X, AlertTriangle, Loader2, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import {
    Select,
    SelectTrigger,
    SelectValue,
    SelectContent,
    SelectItem,
} from "@/components/ui/select";

import {
    useCbcLearningAreas,
    useCbcTeachers,
    useCbcClassTeachers,
    useSlotConflictCheck,
    useCreateCbcTimetableSlot,
    useUpdateCbcTimetableSlot,
    useDeleteCbcTimetableSlot,
    useSlotAttendanceCount,
} from "@/features/cbc/hooks/use-cbc-timetable";
import type { CbcTimetableSlot, OperatingDay } from "@/features/cbc/types";

// ─── Props ────────────────────────────────────────────────────────────────

interface CbcSlotEditorSidePanelProps {
    classId: string;
    schoolId: string;
    academicYearId: string;
    gradeId: string;
    operatingDays: OperatingDay[];
    existingSlot: CbcTimetableSlot | null;
    defaultDay?: number;
    defaultStartTime?: string;
    hasClassTeachers: boolean;
    onSaved: () => void;
    onClose: () => void;
}

// ─── Constants ────────────────────────────────────────────────────────────

const DEFAULT_PERIOD_MINUTES = 40;
const TIME_INCREMENTS = 5;

// ─── Helpers ──────────────────────────────────────────────────────────────

function minutesToTime(minutes: number): string {
    const h = Math.floor(minutes / 60);
    const m = minutes % 60;
    return `${String(h).padStart(2, "0")}:${String(m).padStart(2, "0")}`;
}

function timeToMinutes(time: string): number {
    const [h, m] = time.split(":").map(Number);
    return h * 60 + m;
}

function generateTimeOptions(): string[] {
    const options: string[] = [];
    for (let m = 0; m < 24 * 60; m += TIME_INCREMENTS) {
        options.push(minutesToTime(m));
    }
    return options;
}

const TIME_OPTIONS = generateTimeOptions();

// ─── Component ────────────────────────────────────────────────────────────

export function CbcSlotEditorSidePanel({
    classId,
    schoolId,
    academicYearId,
    gradeId,
    operatingDays,
    existingSlot,
    defaultDay,
    defaultStartTime,
    hasClassTeachers,
    onSaved,
    onClose,
}: CbcSlotEditorSidePanelProps) {
    const isEditing = !!existingSlot;

    // ── Form State ─────────────────────────────────────────────────────
    const [learningAreaId, setLearningAreaId] = React.useState<string | "none">(
        existingSlot?.cbc_learning_area_id ?? "none"
    );
    const [teacherId, setTeacherId] = React.useState(existingSlot?.teacher_id ?? "");
    const [dayOfWeek, setDayOfWeek] = React.useState(existingSlot?.day_of_week ?? defaultDay ?? 1);
    const [startTime, setStartTime] = React.useState(
        existingSlot?.start_time ?? defaultStartTime ?? "08:00"
    );
    const [endTime, setEndTime] = React.useState(
        existingSlot?.end_time ?? minutesToTime(timeToMinutes(startTime) + DEFAULT_PERIOD_MINUTES)
    );
    const [roomIdentifier, setRoomIdentifier] = React.useState(existingSlot?.room_identifier ?? "");
    const [showAllTeachers, setShowAllTeachers] = React.useState(!hasClassTeachers);

    // ── Delete confirm ─────────────────────────────────────────────────
    const [showDeleteConfirm, setShowDeleteConfirm] = React.useState(false);
    const { data: attendanceCount } = useSlotAttendanceCount(isEditing ? existingSlot.id : null);
    const { mutateAsync: deleteSlot, isPending: isDeleting } = useDeleteCbcTimetableSlot(classId);

    // ── Data ───────────────────────────────────────────────────────────
    const resolvedLearningAreaId = learningAreaId === "none" ? null : learningAreaId;

    const { data: learningAreas = [] } = useCbcLearningAreas(gradeId);

    const { data: allTeachers = [] } = useCbcTeachers(schoolId);

    const { data: scopedTeachers = [] } = useCbcClassTeachers(
        showAllTeachers ? null : classId,
        resolvedLearningAreaId
    );

    // ── Conflict check ────────────────────────────────────────────────
    const { data: conflicts = [], isLoading: conflictsLoading } = useSlotConflictCheck(
        teacherId || null,
        dayOfWeek,
        startTime,
        endTime,
        academicYearId,
        schoolId,
        roomIdentifier || null,
        existingSlot?.id ?? null,
        classId // exclude self
    );

    const teacherConflicts = conflicts.filter((c) => c.type === "teacher");
    const roomConflicts = conflicts.filter((c) => c.type === "room");

    // ── Mutations ─────────────────────────────────────────────────────
    const { mutateAsync: createSlot, isPending: isCreating } = useCreateCbcTimetableSlot(classId);
    const { mutateAsync: updateSlot, isPending: isUpdating } = useUpdateCbcTimetableSlot(classId);

    const isSaving = isCreating || isUpdating;

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();

        if (isEditing && existingSlot) {
            await updateSlot({
                id: existingSlot.id,
                teacher_id: teacherId,
                cbc_learning_area_id: resolvedLearningAreaId,
                room_identifier: roomIdentifier || null,
                day_of_week: dayOfWeek,
                start_time: startTime,
                end_time: endTime,
            });
        } else {
            await createSlot({
                class_id: classId,
                teacher_id: teacherId,
                cbc_learning_area_id: resolvedLearningAreaId,
                room_identifier: roomIdentifier || null,
                day_of_week: dayOfWeek,
                start_time: startTime,
                end_time: endTime,
            });
        }

        onSaved();
    };

    const handleDelete = async () => {
        if (!existingSlot) return;
        await deleteSlot(existingSlot.id);
        onSaved();
    };

    // ── Teacher list (scoped or all) ──────────────────────────────────
    const teacherList = showAllTeachers ? allTeachers : scopedTeachers;

    // ── Render ────────────────────────────────────────────────────────
    return (
        <div className="fixed inset-y-0 right-0 z-50 flex w-full max-w-md flex-col border-l bg-white shadow-lg">
            {/* ── Header ─────────────────────────────────────────────── */}
            <div className="flex items-center justify-between border-b px-4 py-3">
                <h3 className="text-base font-medium">{isEditing ? "Edit slot" : "Add slot"}</h3>
                <button
                    onClick={onClose}
                    className="text-muted-foreground hover:text-foreground rounded-md p-1 transition-colors"
                    aria-label="Close panel"
                >
                    <X className="size-4" />
                </button>
            </div>

            {/* ── Form ───────────────────────────────────────────────── */}
            <form
                onSubmit={handleSubmit}
                className="flex flex-1 flex-col gap-5 overflow-y-auto px-4 py-4"
            >
                {/* 1. Learning area */}
                <fieldset>
                    <Label htmlFor="learning-area" className="mb-1.5 block text-sm font-medium">
                        Learning area
                    </Label>
                    <Select
                        value={learningAreaId}
                        onValueChange={(val) => {
                            setLearningAreaId(val);
                            // If switching learning area, reset teacher scope
                            if (!showAllTeachers) setShowAllTeachers(false);
                        }}
                    >
                        <SelectTrigger id="learning-area" className="w-full">
                            <SelectValue placeholder="Select learning area" />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectItem value="none">
                                No learning area (break / assembly / free period)
                            </SelectItem>
                            {learningAreas.map((la) => (
                                <SelectItem key={la.id} value={la.id}>
                                    {la.name} ({la.code})
                                </SelectItem>
                            ))}
                        </SelectContent>
                    </Select>
                </fieldset>

                {/* 2. Teacher */}
                <fieldset>
                    <div className="mb-1.5 flex items-center justify-between">
                        <Label htmlFor="teacher" className="text-sm font-medium">
                            Teacher
                        </Label>
                        {!showAllTeachers && (
                            <button
                                type="button"
                                onClick={() => setShowAllTeachers(true)}
                                className="text-xs text-teal-600 underline-offset-2 hover:text-teal-700 hover:underline"
                            >
                                Show all teachers
                            </button>
                        )}
                        {showAllTeachers && hasClassTeachers && (
                            <button
                                type="button"
                                onClick={() => setShowAllTeachers(false)}
                                className="text-muted-foreground hover:text-foreground text-xs underline-offset-2 hover:underline"
                            >
                                Show suggested only
                            </button>
                        )}
                    </div>
                    <Select value={teacherId} onValueChange={setTeacherId}>
                        <SelectTrigger id="teacher" className="w-full">
                            <SelectValue
                                placeholder={
                                    showAllTeachers
                                        ? "Select teacher..."
                                        : "Select teacher (suggested)..."
                                }
                            />
                        </SelectTrigger>
                        <SelectContent>
                            {teacherList.length === 0 && (
                                <div className="text-muted-foreground px-2 py-4 text-center text-xs">
                                    {showAllTeachers
                                        ? "No teachers found at this school"
                                        : "No teachers assigned for this class yet"}
                                </div>
                            )}
                            {teacherList.map((t) => (
                                <SelectItem key={t.id} value={t.id}>
                                    {t.name}
                                </SelectItem>
                            ))}
                        </SelectContent>
                    </Select>

                    {/* Teacher conflict warning */}
                    {teacherConflicts.length > 0 && (
                        <div className="mt-2 flex items-start gap-1.5 rounded-md border border-red-200 bg-red-50 px-2.5 py-2 text-xs text-red-700">
                            <AlertTriangle className="mt-0.5 size-3.5 shrink-0" />
                            <ul className="list-inside list-disc space-y-0.5">
                                {teacherConflicts.map((c, i) => (
                                    <li key={i}>
                                        {c.entity} is already teaching {c.class_name} at this time
                                    </li>
                                ))}
                            </ul>
                        </div>
                    )}
                </fieldset>

                {/* 3. Day of week */}
                <fieldset>
                    <Label className="mb-1.5 block text-sm font-medium">Day of week</Label>
                    <div className="flex gap-1">
                        {operatingDays.map((day) => (
                            <button
                                key={day.value}
                                type="button"
                                onClick={() => setDayOfWeek(day.value)}
                                className={`flex-1 rounded-md px-2 py-1.5 text-xs font-medium transition-colors ${
                                    dayOfWeek === day.value
                                        ? "bg-teal-500 text-white"
                                        : "bg-secondary text-secondary-foreground hover:bg-teal-100"
                                }`}
                                aria-pressed={dayOfWeek === day.value}
                            >
                                {day.short_label}
                            </button>
                        ))}
                    </div>
                </fieldset>

                {/* 4. Start / End time */}
                <fieldset>
                    <Label className="mb-1.5 block text-sm font-medium">Time</Label>
                    <div className="flex items-center gap-2">
                        <div className="flex-1">
                            <Select value={startTime} onValueChange={setStartTime}>
                                <SelectTrigger>
                                    <SelectValue placeholder="Start" />
                                </SelectTrigger>
                                <SelectContent className="max-h-48">
                                    {TIME_OPTIONS.map((t) => (
                                        <SelectItem key={t} value={t}>
                                            {t}
                                        </SelectItem>
                                    ))}
                                </SelectContent>
                            </Select>
                        </div>
                        <span className="text-muted-foreground text-sm">to</span>
                        <div className="flex-1">
                            <Select value={endTime} onValueChange={setEndTime}>
                                <SelectTrigger>
                                    <SelectValue placeholder="End" />
                                </SelectTrigger>
                                <SelectContent className="max-h-48">
                                    {TIME_OPTIONS.filter(
                                        (t) => timeToMinutes(t) > timeToMinutes(startTime)
                                    ).map((t) => (
                                        <SelectItem key={t} value={t}>
                                            {t}
                                        </SelectItem>
                                    ))}
                                </SelectContent>
                            </Select>
                        </div>
                    </div>
                </fieldset>

                {/* 5. Room */}
                <fieldset>
                    <Label htmlFor="room" className="mb-1.5 block text-sm font-medium">
                        Room
                    </Label>
                    <Input
                        id="room"
                        type="text"
                        placeholder="e.g. Room 4, Lab B"
                        value={roomIdentifier}
                        onChange={(e) => setRoomIdentifier(e.target.value)}
                        className="w-full"
                    />

                    {/* Room conflict warning */}
                    {roomConflicts.length > 0 && (
                        <div className="mt-2 flex items-start gap-1.5 rounded-md border border-red-200 bg-red-50 px-2.5 py-2 text-xs text-red-700">
                            <AlertTriangle className="mt-0.5 size-3.5 shrink-0" />
                            <ul className="list-inside list-disc space-y-0.5">
                                {roomConflicts.map((c, i) => (
                                    <li key={i}>
                                        {c.entity} is in use by {c.class_name} at this time
                                    </li>
                                ))}
                            </ul>
                        </div>
                    )}
                </fieldset>

                {/* ── Conflict loading indicator ──────────────────────── */}
                {conflictsLoading && (
                    <div className="text-muted-foreground flex items-center gap-1.5 text-xs">
                        <Loader2 className="size-3 animate-spin" />
                        Checking for conflicts...
                    </div>
                )}

                {/* ── Spacer ──────────────────────────────────────────── */}
                <div className="flex-1" />

                {/* ── Actions ─────────────────────────────────────────── */}
                <Separator />

                {isEditing && (
                    <div className="mb-2">
                        {!showDeleteConfirm ? (
                            <Button
                                type="button"
                                variant="ghost"
                                size="sm"
                                className="text-destructive hover:text-destructive flex items-center gap-1.5 px-0"
                                onClick={() => setShowDeleteConfirm(true)}
                            >
                                <Trash2 className="size-3.5" />
                                Delete slot
                            </Button>
                        ) : (
                            <div className="rounded-md border border-red-200 bg-red-50 px-3 py-2">
                                <p className="mb-2 text-xs text-red-700">
                                    {attendanceCount && attendanceCount.count > 0
                                        ? `This slot has ${attendanceCount.count} attendance record(s) linked to it. Deleting it won't delete those records, but you won't be able to take attendance against this slot anymore.`
                                        : "This slot has no attendance records. It will be removed immediately."}
                                </p>
                                <div className="flex items-center gap-2">
                                    <Button
                                        type="button"
                                        variant="destructive"
                                        size="sm"
                                        onClick={handleDelete}
                                        disabled={isDeleting}
                                    >
                                        {isDeleting ? "Deleting..." : "Delete"}
                                    </Button>
                                    <Button
                                        type="button"
                                        variant="outline"
                                        size="sm"
                                        onClick={() => setShowDeleteConfirm(false)}
                                    >
                                        Cancel
                                    </Button>
                                </div>
                            </div>
                        )}
                    </div>
                )}

                <div className="flex items-center gap-2">
                    <Button type="button" variant="outline" onClick={onClose} className="flex-1">
                        Cancel
                    </Button>
                    <Button
                        type="submit"
                        disabled={isSaving || !teacherId}
                        className="flex-1 bg-teal-600 text-white hover:bg-teal-700"
                    >
                        {isSaving ? (
                            <>
                                <Loader2 className="mr-1.5 size-3.5 animate-spin" />
                                Saving...
                            </>
                        ) : isEditing ? (
                            "Update slot"
                        ) : (
                            "Add slot"
                        )}
                    </Button>
                </div>
            </form>
        </div>
    );
}
