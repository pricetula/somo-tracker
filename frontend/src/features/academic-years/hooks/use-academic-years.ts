/**
 * TanStack Query hooks for academic years and terms management.
 *
 * Provides queries for listing years/terms and mutations for all CRUD operations.
 * Query keys are scoped to this feature — independent of the combobox hooks
 * in the academic-terms feature.
 */

"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { useRouter } from "next/navigation";

import {
    listAcademicYears,
    listTerms,
    createAcademicYear,
    updateAcademicYear,
    setCurrentYear,
    deleteAcademicYear,
    createTerm,
    updateTerm,
} from "@/lib/api/academic-terms";
import { getErrorMessage } from "@/lib/errors";
import type {
    CreateAcademicYearPayload,
    UpdateAcademicYearPayload,
    CreateTermPayload,
    UpdateTermPayload,
    AcademicYear,
} from "@/lib/api/academic-terms";

// ─── Query keys ───────────────────────────────────────────────────────────

export const academicYearKeys = {
    all: ["academic-years-manage"] as const,
    lists: () => [...academicYearKeys.all, "list"] as const,
    detail: (id: string) => [...academicYearKeys.all, "detail", id] as const,
};

export const academicTermKeys = {
    all: ["academic-terms-manage"] as const,
    list: (yearId?: string) => [...academicTermKeys.all, "list", yearId] as const,
};

// ─── Hooks — Queries ──────────────────────────────────────────────────────

/** Fetch all academic years for the active school. */
export function useAcademicYearsManage() {
    return useQuery({
        queryKey: academicYearKeys.lists(),
        queryFn: () => listAcademicYears(),
        staleTime: 2 * 60 * 1000,
        placeholderData: (prev) => prev,
    });
}

/**
 * Fetch all academic years as a Record<id, AcademicYear> for O(1) lookups.
 *
 * Shares the same cache entry as useAcademicYearsManage — no extra network
 * request. Useful for cross-references from other entities.
 */
export function useAcademicYearMap() {
    return useQuery({
        queryKey: academicYearKeys.lists(),
        queryFn: () => listAcademicYears(),
        staleTime: 2 * 60 * 1000,
        placeholderData: (prev) => prev,
        select: (data: { items: AcademicYear[] }): Record<string, AcademicYear> =>
            Object.fromEntries(data.items.map((y) => [y.id, y])),
    });
}

/** Derive a single academic year (with terms) from the list query. */
export function useAcademicYearDetail(id: string | undefined) {
    const { data, ...rest } = useAcademicYearsManage();

    const year = data?.items?.find((y) => y.id === id) ?? null;

    return {
        ...rest,
        data: year as AcademicYear | null,
    };
}

/** Fetch terms, optionally filtered by academic year. */
export function useTermsManage(academicYearId?: string) {
    return useQuery({
        queryKey: academicTermKeys.list(academicYearId),
        queryFn: () => listTerms(academicYearId ? { academic_year_id: academicYearId } : {}),
        staleTime: 2 * 60 * 1000,
        placeholderData: (prev) => prev,
    });
}

// ─── Hooks — Mutations ────────────────────────────────────────────────────

/** Create a new academic year. */
export function useCreateAcademicYear() {
    const queryClient = useQueryClient();
    const router = useRouter();

    return useMutation({
        mutationFn: (payload: CreateAcademicYearPayload) => createAcademicYear(payload),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: academicYearKeys.all });
            toast.success("Academic year created");
            router.push("/academic-years");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

/** Update an existing academic year (optimistic locking via version). */
export function useUpdateAcademicYear() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ id, payload }: { id: string; payload: UpdateAcademicYearPayload }) =>
            updateAcademicYear(id, payload),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: academicYearKeys.all });
            toast.success("Academic year updated");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

/** Set an academic year as the current year. */
export function useSetCurrentYear() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (id: string) => setCurrentYear(id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: academicYearKeys.all });
            toast.success("Current academic year updated");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

/** Delete an academic year and its cascade-deleted terms. */
export function useDeleteAcademicYear() {
    const queryClient = useQueryClient();
    const router = useRouter();

    return useMutation({
        mutationFn: (id: string) => deleteAcademicYear(id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: academicYearKeys.all });
            toast.success("Academic year deleted");
            router.push("/academic-years");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

/** Create a new term within an academic year. */
export function useCreateTerm() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (payload: CreateTermPayload) => createTerm(payload),
        onSuccess: (_data, variables) => {
            queryClient.invalidateQueries({ queryKey: academicYearKeys.all });
            queryClient.invalidateQueries({
                queryKey: academicTermKeys.list(variables.academic_year_id),
            });
            toast.success("Term created");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

/** Update an existing term (optimistic locking via version). */
export function useUpdateTerm() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ id, payload }: { id: string; payload: UpdateTermPayload }) =>
            updateTerm(id, payload),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: academicYearKeys.all });
            queryClient.invalidateQueries({ queryKey: academicTermKeys.all });
            toast.success("Term updated");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}
