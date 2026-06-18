"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import * as attendanceApi from "@/features/cbc/api/attendance";
import { getApiErrorMessage } from "@/lib/api/auth";
import type { AttendanceStatus } from "@/features/cbc/types";

// ─── Query Keys ───────────────────────────────────────────────────────────

export const cbcAttendanceKeys = {
    periods: (classId: string, date: string) =>
        ["cbc", "attendance", "periods", classId, date] as const,
    periodSummaries: (classId: string, from: string, to: string) =>
        ["cbc", "attendance", "summaries", classId, from, to] as const,
    periodDetail: (periodId: string) => ["cbc", "attendance", "periodDetail", periodId] as const,
    logs: (periodId: string) => ["cbc", "attendance", "logs", periodId] as const,
    students: (classId: string, termId: string) =>
        ["cbc", "attendance", "students", classId, termId] as const,
    teacherToday: (teacherId: string) => ["cbc", "attendance", "teacherToday", teacherId] as const,
    heatmap: (classId: string, termId: string) =>
        ["cbc", "attendance", "heatmap", classId, termId] as const,
    gaps: (classId: string, from: string, to: string) =>
        ["cbc", "attendance", "gaps", classId, from, to] as const,
} as const;

// ─── Hooks ────────────────────────────────────────────────────────────────

/** Fetch attendance periods for a class on a given date. */
export function useCbcAttendancePeriods(classId: string, date: string) {
    return useQuery({
        queryKey: cbcAttendanceKeys.periods(classId, date),
        queryFn: () => attendanceApi.fetchCbcAttendancePeriods(classId, date),
        staleTime: 15_000,
        refetchOnWindowFocus: false,
        enabled: !!classId && !!date,
    });
}

/** Fetch attendance period summaries for a date range (list view). */
export function useCbcAttendancePeriodSummaries(classId: string, from: string, to: string) {
    return useQuery({
        queryKey: cbcAttendanceKeys.periodSummaries(classId, from, to),
        queryFn: () => attendanceApi.fetchCbcAttendancePeriodSummaries(classId, from, to),
        staleTime: 10_000,
        refetchOnWindowFocus: false,
        enabled: !!classId && !!from && !!to,
    });
}

/** Fetch detailed period info. */
export function useCbcAttendancePeriodDetail(periodId: string | null) {
    return useQuery({
        queryKey: cbcAttendanceKeys.periodDetail(periodId ?? ""),
        queryFn: () => attendanceApi.fetchCbcAttendancePeriodDetail(periodId!),
        enabled: !!periodId,
        staleTime: 10_000,
    });
}

/** Create an attendance period. */
export function useCreateCbcAttendancePeriod(classId: string) {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ cbcLearningAreaId, date }: { cbcLearningAreaId: string; date: string }) =>
            attendanceApi.createCbcAttendancePeriod(classId, cbcLearningAreaId, date),
        onSuccess: async () => {
            await queryClient.invalidateQueries({
                queryKey: ["cbc", "attendance", "periods", classId],
            });
            await queryClient.invalidateQueries({
                queryKey: ["cbc", "attendance", "summaries", classId],
            });
            await queryClient.invalidateQueries({
                queryKey: ["cbc", "attendance", "heatmap", classId],
            });
            await queryClient.invalidateQueries({
                queryKey: ["cbc", "attendance", "gaps", classId],
            });
        },
        onError: (err: unknown) => {
            toast.error("Failed to start attendance", {
                description: getApiErrorMessage(err),
            });
        },
    });
}

/** Fetch attendance logs for a period (with recorder details). */
export function useCbcAttendanceLogs(periodId: string | null) {
    return useQuery({
        queryKey: cbcAttendanceKeys.logs(periodId ?? ""),
        queryFn: () => attendanceApi.fetchCbcAttendanceLogs(periodId!),
        enabled: !!periodId,
        staleTime: 10_000,
        refetchOnWindowFocus: false,
    });
}

/** Fetch enrolled students for attendance grid. */
export function useCbcClassStudents(classId: string | null, academicTermId: string | null) {
    return useQuery({
        queryKey: cbcAttendanceKeys.students(classId ?? "", academicTermId ?? ""),
        queryFn: () => attendanceApi.fetchClassStudents(classId!, academicTermId!),
        enabled: !!classId && !!academicTermId,
        staleTime: 60_000,
    });
}

/** Save a single attendance mark (optimistic). */
export function useSaveAttendanceMark(periodId: string) {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({
            studentId,
            status,
            remarks,
        }: {
            studentId: string;
            status: AttendanceStatus;
            remarks?: string;
        }) => attendanceApi.saveAttendanceMark(periodId, studentId, status, remarks),
        onSuccess: async () => {
            await queryClient.invalidateQueries({
                queryKey: cbcAttendanceKeys.logs(periodId),
            });
            await queryClient.invalidateQueries({
                queryKey: ["cbc", "attendance", "summaries"],
            });
        },
        onError: (err: unknown) => {
            toast.error("Failed to save attendance", {
                description: getApiErrorMessage(err),
            });
        },
    });
}

/** Batch save all marks for a period. */
export function useSaveAttendanceBatch(periodId: string) {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (
            marks: Array<{
                student_id: string;
                status: AttendanceStatus;
                remarks?: string;
            }>
        ) => attendanceApi.saveAttendanceBatch(periodId, marks),
        onSuccess: async () => {
            await queryClient.invalidateQueries({
                queryKey: cbcAttendanceKeys.logs(periodId),
            });
            await queryClient.invalidateQueries({
                queryKey: ["cbc", "attendance", "summaries"],
            });
            toast.success("Attendance saved");
        },
        onError: (err: unknown) => {
            toast.error("Failed to save attendance", {
                description: getApiErrorMessage(err),
            });
        },
    });
}

/** Mark all remaining unmarked students as Present. */
export function useMarkRemainingAsPresent(periodId: string) {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (studentIds: string[]) =>
            attendanceApi.markRemainingAsPresent(periodId, studentIds),
        onSuccess: async () => {
            await queryClient.invalidateQueries({
                queryKey: cbcAttendanceKeys.logs(periodId),
            });
            await queryClient.invalidateQueries({
                queryKey: ["cbc", "attendance", "summaries"],
            });
            toast.success("Marked remaining students as Present");
        },
        onError: (err: unknown) => {
            toast.error("Failed to mark remaining", {
                description: getApiErrorMessage(err),
            });
        },
    });
}

/** Fetch today's slots for a teacher (dashboard). */
export function useTeacherTodaySlots(teacherId: string | null) {
    return useQuery({
        queryKey: cbcAttendanceKeys.teacherToday(teacherId ?? ""),
        queryFn: () => attendanceApi.fetchTeacherTodaySlots(teacherId!),
        enabled: !!teacherId,
        staleTime: 30_000,
        refetchOnWindowFocus: true,
    });
}

/** Fetch attendance heatmap data for a term. */
export function useCbcAttendanceHeatmap(classId: string, termId: string) {
    return useQuery({
        queryKey: cbcAttendanceKeys.heatmap(classId, termId),
        queryFn: () => attendanceApi.fetchCbcAttendanceHeatmap(classId, termId),
        staleTime: 30_000,
        enabled: !!classId && !!termId,
    });
}

/** Fetch attendance gaps for a date range. */
export function useCbcAttendanceGaps(classId: string, from: string, to: string) {
    return useQuery({
        queryKey: cbcAttendanceKeys.gaps(classId, from, to),
        queryFn: () => attendanceApi.fetchCbcAttendanceGaps(classId, from, to),
        staleTime: 15_000,
        enabled: !!classId && !!from && !!to,
    });
}
