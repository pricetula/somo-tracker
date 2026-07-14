/**
 * useReports — TanStack Query hooks for term report operations.
 */

"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import {
    getTermReport,
    generateTermReport,
    generateClassTermReports,
    publishTermReport,
    listTermReports,
} from "@/lib/api/reports";
import { getErrorMessage } from "@/lib/errors";

// ─── Query keys ───────────────────────────────────────────────────────────

export const reportKeys = {
    all: ["reports"] as const,
    termReport: (termId: string, studentId: string) =>
        [...reportKeys.all, "term", termId, studentId] as const,
    termList: (termId: string, classId?: string) =>
        [...reportKeys.all, "list", termId, classId] as const,
};

// ─── Hooks ────────────────────────────────────────────────────────────────

/** Get a compiled term report (parent read-only). */
export function useTermReport(termId: string, studentId: string) {
    return useQuery({
        queryKey: reportKeys.termReport(termId, studentId),
        queryFn: () => getTermReport(termId, studentId),
        enabled: !!termId && !!studentId,
        staleTime: 60_000,
    });
}

/** List all reports for a term. */
export function useTermReportList(termId: string, classId?: string) {
    return useQuery({
        queryKey: reportKeys.termList(termId, classId),
        queryFn: () => listTermReports(termId, classId),
        enabled: !!termId,
    });
}

/** Generate a single student's term report. */
export function useGenerateTermReport() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ termId, studentId }: { termId: string; studentId: string }) =>
            generateTermReport(termId, studentId),
        onSuccess: (data) => {
            void queryClient.invalidateQueries({ queryKey: reportKeys.all });
            toast.success(`Report generated (${data.status})`);
        },
        onError: (err) => {
            toast.error("Failed to generate report", {
                description: getErrorMessage(err),
            });
        },
    });
}

/** Generate reports for all students in a class. */
export function useGenerateClassReports() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ termId, classId }: { termId: string; classId: string }) =>
            generateClassTermReports(termId, classId),
        onSuccess: (data) => {
            void queryClient.invalidateQueries({ queryKey: reportKeys.all });
            toast.success(`${data.count} reports generated`);
        },
        onError: (err) => {
            toast.error("Failed to generate reports", {
                description: getErrorMessage(err),
            });
        },
    });
}

/** Publish a term report (make visible to parents). */
export function usePublishTermReport() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (reportId: string) => publishTermReport(reportId),
        onSuccess: () => {
            void queryClient.invalidateQueries({ queryKey: reportKeys.all });
            toast.success("Report published");
        },
        onError: (err) => {
            toast.error("Failed to publish report", {
                description: getErrorMessage(err),
            });
        },
    });
}
