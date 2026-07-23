/**
 * useAttendance — TanStack Query hooks for attendance operations.
 */

"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import {
    getSlotRoster,
    bulkMarkAttendance,
    getAdminDashboard,
    getStudentHistory,
    updateAttendanceRecord,
    getChildAttendanceSummary,
    computeAttendanceSummaries,
    skipSession,
    unskipSession,
    getSession,
    type BulkAttendancePayload,
    type AttendanceStatus,
    type SkipSessionPayload,
} from "@/lib/api/attendance";
import { getErrorMessage } from "@/lib/errors";

// ─── Query keys ───────────────────────────────────────────────────────────

export const attendanceKeys = {
    all: ["attendance"] as const,
    roster: (slotId: string, date?: string) =>
        [...attendanceKeys.all, "roster", slotId, date] as const,
    dashboard: (date?: string) => [...attendanceKeys.all, "dashboard", date] as const,
    studentHistory: (studentId: string) => [...attendanceKeys.all, "history", studentId] as const,
    childSummary: (studentId: string, termId: string) =>
        [...attendanceKeys.all, "child", studentId, termId] as const,
};

// ─── Hooks ────────────────────────────────────────────────────────────────

/** Fetch the class roster with existing marks for a given slot and date. */
export function useSlotRoster(timetableSlotId: string, date?: string) {
    return useQuery({
        queryKey: attendanceKeys.roster(timetableSlotId, date),
        queryFn: () => getSlotRoster(timetableSlotId, date),
        enabled: !!timetableSlotId,
        staleTime: 30_000,
    });
}

/** Bulk mark attendance for a slot. */
export function useBulkMarkAttendance() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (payload: BulkAttendancePayload) => bulkMarkAttendance(payload),
        onSuccess: (data) => {
            void queryClient.invalidateQueries({ queryKey: attendanceKeys.all });
            toast.success(`Attendance saved — ${data.count} students marked`);
        },
        onError: (err) => {
            toast.error("Couldn't save attendance", {
                description: getErrorMessage(err),
                action: {
                    label: "Retry",
                    onClick: () => {
                        /* retry handled by mutation state */
                    },
                },
            });
        },
    });
}

/** Get admin dashboard — completion status per class. */
export function useAdminDashboard(date?: string) {
    return useQuery({
        queryKey: attendanceKeys.dashboard(date),
        queryFn: () => getAdminDashboard(date),
        staleTime: 30_000,
    });
}

/** Get student attendance history. */
export function useStudentHistory(
    studentId: string,
    filters?: { term_id?: string; start_date?: string; end_date?: string }
) {
    return useQuery({
        queryKey: [...attendanceKeys.studentHistory(studentId), filters],
        queryFn: () => getStudentHistory(studentId, filters),
        enabled: !!studentId,
    });
}

/** Update a single attendance record (admin correction). */
export function useUpdateAttendanceRecord() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({
            recordId,
            status,
            note,
        }: {
            recordId: string;
            status: AttendanceStatus;
            note?: string | null;
        }) => updateAttendanceRecord(recordId, { status, note }),
        onSuccess: () => {
            void queryClient.invalidateQueries({ queryKey: attendanceKeys.all });
            toast.success("Attendance record updated");
        },
        onError: (err) => {
            toast.error("Failed to update record", {
                description: getErrorMessage(err),
            });
        },
    });
}

/** Get parent-facing attendance summary for a child. */
export function useChildAttendanceSummary(studentId: string, termId: string) {
    return useQuery({
        queryKey: attendanceKeys.childSummary(studentId, termId),
        queryFn: () => getChildAttendanceSummary(studentId, termId),
        enabled: !!studentId && !!termId,
        staleTime: 60_000,
    });
}

// ─── Session Hooks ───────────────────────────────────────────────────────

export const attendanceSessionKeys = {
    all: ["attendance-sessions"] as const,
    bySlot: (slotId: string, date: string) => [...attendanceSessionKeys.all, slotId, date] as const,
};

/** Fetch the session status for a slot + date. */
export function useSession(slotId: string, date?: string) {
    return useQuery({
        queryKey: attendanceSessionKeys.bySlot(slotId, date ?? ""),
        queryFn: () => getSession(slotId, date ?? ""),
        enabled: !!slotId && !!date,
        staleTime: 30_000,
    });
}

/** Mark a session as skipped. */
export function useSkipSession() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (payload: SkipSessionPayload) => skipSession(payload),
        onSuccess: () => {
            void queryClient.invalidateQueries({ queryKey: attendanceKeys.all });
            void queryClient.invalidateQueries({ queryKey: attendanceSessionKeys.all });
            toast.success("Session marked as skipped. Attendance records have been removed.");
        },
        onError: (err) => {
            toast.error("Failed to skip session", {
                description: getErrorMessage(err),
            });
        },
    });
}

/** Unskip a session (re-open it). */
export function useUnskipSession() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (payload: { timetable_slot_id: string; date: string }) =>
            unskipSession(payload),
        onSuccess: () => {
            void queryClient.invalidateQueries({ queryKey: attendanceKeys.all });
            void queryClient.invalidateQueries({ queryKey: attendanceSessionKeys.all });
            toast.success("Session re-opened. You can now mark attendance.");
        },
        onError: (err) => {
            toast.error("Failed to unskip session", {
                description: getErrorMessage(err),
            });
        },
    });
}

/** Trigger recomputation of attendance term summaries. */
export function useComputeAttendanceSummaries() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (termId: string) => computeAttendanceSummaries(termId),
        onSuccess: (data) => {
            void queryClient.invalidateQueries({ queryKey: attendanceKeys.all });
            toast.success(`Attendance summaries computed — ${data.count} records updated`);
        },
        onError: (err) => {
            toast.error("Failed to compute summaries", {
                description: getErrorMessage(err),
            });
        },
    });
}
