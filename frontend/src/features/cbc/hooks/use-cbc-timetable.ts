"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import * as timetableApi from "@/features/cbc/api/timetable";
import { getApiErrorMessage } from "@/lib/api/auth";
import type {
    CbcTimetableSlotCreatePayload,
    CbcTimetableSlotUpdatePayload,
    DuplicateDayPayload,
    CopyFromClassPayload,
} from "@/features/cbc/types";

// ─── Query Keys ───────────────────────────────────────────────────────────

export const cbcTimetableKeys = {
    slots: (classId: string) => ["cbc", "timetable", "slots", classId] as const,
    learningAreas: (gradeId: string) => ["cbc", "learningAreas", gradeId] as const,
    teachers: (schoolId: string) => ["cbc", "teachers", schoolId] as const,
    classTeachers: (classId: string, learningAreaId?: string) =>
        ["cbc", "classTeachers", classId, learningAreaId] as const,
    roomAutocomplete: (query: string) => ["cbc", "roomAutocomplete", query] as const,
    slotAttendanceCount: (slotId: string) => ["cbc", "slotAttendanceCount", slotId] as const,
    operatingDays: (schoolId: string) => ["cbc", "operatingDays", schoolId] as const,
} as const;

// ─── Hooks ────────────────────────────────────────────────────────────────

/** Fetch all timetable slots for a class. */
export function useCbcTimetableSlots(classId: string) {
    return useQuery({
        queryKey: cbcTimetableKeys.slots(classId),
        queryFn: () => timetableApi.fetchCbcTimetableSlots(classId),
        staleTime: 30_000,
        refetchOnWindowFocus: false,
        retry: 1,
        enabled: !!classId,
    });
}

/** Create a new timetable slot. */
export function useCreateCbcTimetableSlot(classId: string) {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (payload: CbcTimetableSlotCreatePayload) =>
            timetableApi.createCbcTimetableSlot(classId, payload),
        onSuccess: async () => {
            await queryClient.invalidateQueries({ queryKey: cbcTimetableKeys.slots(classId) });
            toast.success("Slot added", { description: "Timetable slot created successfully." });
        },
        onError: (err: unknown) => {
            toast.error("Failed to create slot", { description: getApiErrorMessage(err) });
        },
    });
}

/** Update an existing timetable slot. */
export function useUpdateCbcTimetableSlot(classId: string) {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (payload: CbcTimetableSlotUpdatePayload) =>
            timetableApi.updateCbcTimetableSlot(payload),
        onSuccess: async () => {
            await queryClient.invalidateQueries({ queryKey: cbcTimetableKeys.slots(classId) });
            toast.success("Slot updated");
        },
        onError: (err: unknown) => {
            const msg = getApiErrorMessage(err);
            // If the error is from an EXCLUDE constraint, surface it human-readable.
            if (msg && (msg.includes("excl_cbc_timetable") || msg.includes("overlapping"))) {
                toast.error("Time conflict", {
                    description:
                        "This slot overlaps with another — check the highlighted conflict.",
                });
            } else {
                toast.error("Failed to update slot", { description: msg });
            }
        },
    });
}

/** Delete a timetable slot. */
export function useDeleteCbcTimetableSlot(classId: string) {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (slotId: string) => timetableApi.deleteCbcTimetableSlot(slotId),
        onSuccess: async () => {
            await queryClient.invalidateQueries({ queryKey: cbcTimetableKeys.slots(classId) });
            toast.success("Slot removed");
        },
        onError: (err: unknown) => {
            toast.error("Failed to delete slot", { description: getApiErrorMessage(err) });
        },
    });
}

/** Fetch attendance count for a slot (for delete confirmation). */
export function useSlotAttendanceCount(slotId: string | null) {
    return useQuery({
        queryKey: cbcTimetableKeys.slotAttendanceCount(slotId ?? ""),
        queryFn: () => timetableApi.fetchSlotAttendanceCount(slotId!),
        enabled: !!slotId,
        staleTime: 60_000,
    });
}

/** Conflict pre-check — fires when all required fields are set. */
export function useSlotConflictCheck(
    teacherId: string | null,
    dayOfWeek: number | null,
    startTime: string | null,
    endTime: string | null,
    academicYearId: string | null,
    schoolId: string | null,
    roomIdentifier: string | null,
    slotId: string | null,
    excludeClassId?: string
) {
    const enabled =
        !!teacherId && !!dayOfWeek && !!startTime && !!endTime && !!academicYearId && !!schoolId;

    return useQuery({
        queryKey: [
            "cbc",
            "conflictCheck",
            teacherId,
            dayOfWeek,
            startTime,
            endTime,
            academicYearId,
            schoolId,
            roomIdentifier,
            slotId,
            excludeClassId,
        ],
        queryFn: () =>
            timetableApi.checkSlotConflicts(
                slotId,
                teacherId!,
                dayOfWeek!,
                startTime!,
                endTime!,
                academicYearId!,
                schoolId!,
                roomIdentifier,
                excludeClassId
            ),
        enabled,
        staleTime: 5_000, // re-check quickly as user edits
        refetchOnWindowFocus: false,
        retry: false,
    });
}

/** Fetch learning areas for a grade. */
export function useCbcLearningAreas(gradeId: string | null) {
    return useQuery({
        queryKey: cbcTimetableKeys.learningAreas(gradeId ?? ""),
        queryFn: () => timetableApi.fetchCbcLearningAreas(gradeId!),
        enabled: !!gradeId,
        staleTime: 300_000,
    });
}

/** Fetch all teachers at a school. */
export function useCbcTeachers(schoolId: string | null) {
    return useQuery({
        queryKey: cbcTimetableKeys.teachers(schoolId ?? ""),
        queryFn: () => timetableApi.fetchCbcTeachers(schoolId!),
        enabled: !!schoolId,
        staleTime: 60_000,
    });
}

/** Fetch teachers scoped to a class + learning area. */
export function useCbcClassTeachers(classId: string | null, learningAreaId: string | null) {
    return useQuery({
        queryKey: cbcTimetableKeys.classTeachers(classId ?? "", learningAreaId ?? undefined),
        queryFn: () => timetableApi.fetchCbcClassTeachers(classId!, learningAreaId ?? undefined),
        enabled: !!classId && !!learningAreaId,
        staleTime: 60_000,
    });
}

/** Room autocomplete. */
export function useRoomAutocomplete(query: string) {
    return useQuery({
        queryKey: cbcTimetableKeys.roomAutocomplete(query),
        queryFn: () => timetableApi.fetchRoomAutocomplete(query),
        enabled: query.length >= 2,
        staleTime: 30_000,
    });
}

/** Duplicate day. */
export function useDuplicateDay(classId: string) {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (payload: DuplicateDayPayload) => timetableApi.duplicateDay(payload),
        onSuccess: async (result) => {
            await queryClient.invalidateQueries({ queryKey: cbcTimetableKeys.slots(classId) });
            const skipped = result.skipped.length;
            if (skipped > 0) {
                toast.success(`Duplicated with ${skipped} skip(s)`, {
                    description: `${result.total_copied} slots copied. ${skipped} slot(s) were skipped due to conflicts.`,
                });
            } else {
                toast.success("Day duplicated", {
                    description: `${result.total_copied} slots copied to target day(s).`,
                });
            }
        },
        onError: (err: unknown) => {
            toast.error("Failed to duplicate day", { description: getApiErrorMessage(err) });
        },
    });
}

/** Copy from another class. */
export function useCopyTimetableFromClass(classId: string) {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (payload: CopyFromClassPayload) => timetableApi.copyTimetableFromClass(payload),
        onSuccess: async (result) => {
            await queryClient.invalidateQueries({ queryKey: cbcTimetableKeys.slots(classId) });
            const skipped = result.skipped.length;
            if (skipped > 0) {
                toast.success(`Copied with ${skipped} skip(s)`, {
                    description: `${result.total_copied} slots copied. ${skipped} slot(s) were skipped due to conflicts.`,
                });
            } else {
                toast.success("Timetable copied", {
                    description: `${result.total_copied} slots copied from the source class.`,
                });
            }
        },
        onError: (err: unknown) => {
            toast.error("Failed to copy timetable", { description: getApiErrorMessage(err) });
        },
    });
}

/** Fetch operating days. */
export function useOperatingDays(schoolId: string | null) {
    return useQuery({
        queryKey: cbcTimetableKeys.operatingDays(schoolId ?? ""),
        queryFn: () => timetableApi.fetchOperatingDays(schoolId!),
        enabled: !!schoolId,
        staleTime: 300_000,
        placeholderData: [
            { value: 1, label: "Monday", short_label: "Mon" },
            { value: 2, label: "Tuesday", short_label: "Tue" },
            { value: 3, label: "Wednesday", short_label: "Wed" },
            { value: 4, label: "Thursday", short_label: "Thu" },
            { value: 5, label: "Friday", short_label: "Fri" },
        ],
    });
}
