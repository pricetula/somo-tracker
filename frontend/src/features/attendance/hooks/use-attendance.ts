/**
 * React Query hooks for the Attendance feature.
 */
"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import {
    batchMarkAttendance,
    createSession,
    getClassTermSummary,
    getSession,
    getSessionsForClassDate,
    getStudentTermSummary,
    listRecords,
    listRecordsByClassDate,
    listRecordsBySlot,
    listRecordsByStudent,
    listSessions,
    refreshSummaries,
    updateRecord,
    updateSession,
} from "@/lib/api/attendance";
import { getErrorMessage } from "@/lib/errors";
import type {
    BatchMarkPayload,
    BatchMarkResult,
    CreateSessionPayload,
    UpdateRecordPayload,
    UpdateSessionPayload,
} from "@/features/attendance/types";

// ─── Query Keys ───────────────────────────────────────────────────────────

export const attendanceKeys = {
    all: ["attendance"] as const,
    sessions: {
        all: ["attendance", "sessions"] as const,
        list: (params?: Record<string, unknown>) =>
            ["attendance", "sessions", "list", params] as const,
        detail: (id: string) => ["attendance", "sessions", id] as const,
        byClassDate: (classId: string, date: string) =>
            ["attendance", "sessions", "class", classId, date] as const,
    },
    records: {
        all: ["attendance", "records"] as const,
        list: (params?: Record<string, unknown>) =>
            ["attendance", "records", "list", params] as const,
        bySlot: (slotId: string, date: string) =>
            ["attendance", "records", "slot", slotId, date] as const,
        byStudent: (studentId: string, termId?: string) =>
            ["attendance", "records", "student", studentId, termId] as const,
        byClassDate: (classId: string, date: string) =>
            ["attendance", "records", "class", classId, date] as const,
    },
    summaries: {
        all: ["attendance", "summaries"] as const,
        byStudent: (studentId: string, termId?: string) =>
            ["attendance", "summaries", "student", studentId, termId] as const,
        byClass: (classId: string, termId?: string) =>
            ["attendance", "summaries", "class", classId, termId] as const,
    },
};

// ─── Sessions Queries ─────────────────────────────────────────────────────

/** List attendance sessions with optional filters. */
export function useAttendanceSessions(params?: {
    timetable_slot_id?: string;
    date?: string;
    status?: string;
    class_id?: string;
}) {
    return useQuery({
        queryKey: attendanceKeys.sessions.list(params as Record<string, unknown> | undefined),
        queryFn: () => listSessions(params),
    });
}

/** Get a single attendance session by ID. */
export function useAttendanceSession(id: string) {
    return useQuery({
        queryKey: attendanceKeys.sessions.detail(id),
        queryFn: () => getSession(id),
        enabled: !!id,
    });
}

/** Get sessions for a class on a specific date. */
export function useSessionsForClassDate(classId: string, date: string) {
    return useQuery({
        queryKey: attendanceKeys.sessions.byClassDate(classId, date),
        queryFn: () => getSessionsForClassDate(classId, date),
        enabled: !!classId && !!date,
    });
}

// ─── Records Queries ──────────────────────────────────────────────────────

/** List attendance records with filters. */
export function useAttendanceRecords(params?: {
    timetable_slot_id?: string;
    date?: string;
    student_id?: string;
    class_id?: string;
    academic_term_id?: string;
    status?: string;
}) {
    return useQuery({
        queryKey: attendanceKeys.records.list(params as Record<string, unknown> | undefined),
        queryFn: () => listRecords(params),
    });
}

/** List records by timetable slot and date. */
export function useRecordsBySlot(timetableSlotId: string, date: string) {
    return useQuery({
        queryKey: attendanceKeys.records.bySlot(timetableSlotId, date),
        queryFn: () => listRecordsBySlot(timetableSlotId, date),
        enabled: !!timetableSlotId && !!date,
    });
}

/** List records for a student in a term. */
export function useRecordsByStudent(studentId: string, termId?: string) {
    return useQuery({
        queryKey: attendanceKeys.records.byStudent(studentId, termId),
        queryFn: () => listRecordsByStudent(studentId, termId),
        enabled: !!studentId,
    });
}

/** List records for a class on a date. */
export function useRecordsByClassDate(classId: string, date: string) {
    return useQuery({
        queryKey: attendanceKeys.records.byClassDate(classId, date),
        queryFn: () => listRecordsByClassDate(classId, date),
        enabled: !!classId && !!date,
    });
}

// ─── Summaries Queries ────────────────────────────────────────────────────

/** Get attendance summaries for a student in a term. */
export function useStudentTermSummary(studentId: string, termId?: string) {
    return useQuery({
        queryKey: attendanceKeys.summaries.byStudent(studentId, termId),
        queryFn: () => getStudentTermSummary(studentId, termId),
        enabled: !!studentId,
    });
}

/** Get attendance summaries for a class in a term. */
export function useClassTermSummary(classId: string, termId?: string) {
    return useQuery({
        queryKey: attendanceKeys.summaries.byClass(classId, termId),
        queryFn: () => getClassTermSummary(classId, termId),
        enabled: !!classId,
    });
}

// ─── Sessions Mutations ───────────────────────────────────────────────────

/** Create a new attendance session. */
export function useCreateSession() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: (data: CreateSessionPayload) => createSession(data),
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
        onSettled: () => {
            queryClient.invalidateQueries({ queryKey: attendanceKeys.sessions.all });
        },
    });
}

/** Update an attendance session. */
export function useUpdateSession(id: string) {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: (data: UpdateSessionPayload) => updateSession(id, data),
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
        onSettled: () => {
            queryClient.invalidateQueries({ queryKey: attendanceKeys.sessions.all });
            queryClient.invalidateQueries({ queryKey: attendanceKeys.sessions.detail(id) });
        },
    });
}

// ─── Records Mutations ────────────────────────────────────────────────────

/** Batch mark attendance for multiple students. */
export function useBatchMarkAttendance(termId?: string) {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: (data: BatchMarkPayload) => batchMarkAttendance(data, termId),
        onSuccess: (result: BatchMarkResult) => {
            toast.success(`Attendance saved: ${result.created} created, ${result.updated} updated`);
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
        onSettled: () => {
            queryClient.invalidateQueries({ queryKey: attendanceKeys.records.all });
            queryClient.invalidateQueries({ queryKey: attendanceKeys.summaries.all });
        },
    });
}

/** Update a single attendance record. */
export function useUpdateRecord() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: ({ id, data }: { id: string; data: UpdateRecordPayload }) =>
            updateRecord(id, data),
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
        onSettled: () => {
            queryClient.invalidateQueries({ queryKey: attendanceKeys.records.all });
            queryClient.invalidateQueries({ queryKey: attendanceKeys.summaries.all });
        },
    });
}

// ─── Summaries Mutations ──────────────────────────────────────────────────

/** Refresh attendance summaries for a term. */
export function useRefreshSummaries() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: (termId: string) => refreshSummaries(termId),
        onSuccess: () => {
            toast.success("Attendance summaries refreshed");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
        onSettled: () => {
            queryClient.invalidateQueries({ queryKey: attendanceKeys.summaries.all });
        },
    });
}
