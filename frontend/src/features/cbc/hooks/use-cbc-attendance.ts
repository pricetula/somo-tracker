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
    logs: (periodId: string) => ["cbc", "attendance", "logs", periodId] as const,
    students: (classId: string, termId: string) =>
        ["cbc", "attendance", "students", classId, termId] as const,
    teacherToday: (teacherId: string) => ["cbc", "attendance", "teacherToday", teacherId] as const,
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
        },
        onError: (err: unknown) => {
            toast.error("Failed to start attendance", {
                description: getApiErrorMessage(err),
            });
        },
    });
}

/** Fetch attendance logs for a period. */
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
            toast.success("Attendance saved");
        },
        onError: (err: unknown) => {
            toast.error("Failed to save attendance", {
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
