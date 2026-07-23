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
import { STALE_TIMES } from "@/lib/query-config";
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
        staleTime: STALE_TIMES.STANDARD,
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
        staleTime: STALE_TIMES.STANDARD,
        placeholderData: (prev) => prev,
        select: (data: { items: AcademicYear[] }): Record<string, AcademicYear> =>
            data?.items?.reduce?.(
                (acc, item) => {
                    acc[item.id] = item;
                    return acc;
                },
                {} as Record<string, AcademicYear>
            ) ?? {},
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
        staleTime: STALE_TIMES.STANDARD,
        placeholderData: (prev) => prev,
    });
}

// ─── Hooks — Mutations ────────────────────────────────────────────────────

/** Create a new academic year with optimistic update. */
export function useCreateAcademicYear() {
    const queryClient = useQueryClient();
    const router = useRouter();

    return useMutation({
        mutationFn: (payload: CreateAcademicYearPayload) => createAcademicYear(payload),
        onMutate: async () => {
            await queryClient.cancelQueries({ queryKey: academicYearKeys.all });
            const previousQueries = queryClient.getQueriesData({
                queryKey: academicYearKeys.all,
            });
            return { previousQueries };
        },
        onError: (err, _payload, context) => {
            if (context?.previousQueries) {
                for (const [key, data] of context.previousQueries) {
                    queryClient.setQueryData(key, data);
                }
            }
            toast.error(getErrorMessage(err));
        },
        onSettled: () => {
            queryClient.invalidateQueries({ queryKey: academicYearKeys.all });
        },
        onSuccess: () => {
            router.push("/academic-years");
        },
    });
}

/** Update an existing academic year with optimistic update. */
export function useUpdateAcademicYear() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ id, payload }: { id: string; payload: UpdateAcademicYearPayload }) =>
            updateAcademicYear(id, payload),
        onMutate: async ({ id, payload }) => {
            await queryClient.cancelQueries({ queryKey: academicYearKeys.all });
            const previousQueries = queryClient.getQueriesData<{ items: { id: string }[] }>({
                queryKey: academicYearKeys.all,
            });

            queryClient.setQueriesData<{ items: { id: string }[] }>(
                { queryKey: academicYearKeys.all },
                (old) => {
                    if (!old) return old;
                    return {
                        ...old,
                        items: old.items.map((item) =>
                            item.id === id ? { ...item, ...payload } : item
                        ),
                    };
                }
            );

            return { previousQueries };
        },
        onError: (err, _vars, context) => {
            if (context?.previousQueries) {
                for (const [key, data] of context.previousQueries) {
                    queryClient.setQueryData(key, data);
                }
            }
            toast.error(getErrorMessage(err));
        },
        onSettled: () => {
            queryClient.invalidateQueries({ queryKey: academicYearKeys.all });
        },
    });
}

/** Set an academic year as the current year with optimistic update. */
export function useSetCurrentYear() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (id: string) => setCurrentYear(id),
        onMutate: async (id) => {
            await queryClient.cancelQueries({ queryKey: academicYearKeys.all });
            const previousQueries = queryClient.getQueriesData<{
                items: { id: string; is_current?: boolean }[];
            }>({
                queryKey: academicYearKeys.all,
            });

            queryClient.setQueriesData<{ items: { id: string; is_current?: boolean }[] }>(
                { queryKey: academicYearKeys.all },
                (old) => {
                    if (!old) return old;
                    return {
                        ...old,
                        items: old.items.map((item) => ({
                            ...item,
                            is_current: item.id === id,
                        })),
                    };
                }
            );

            return { previousQueries };
        },
        onError: (err, _id, context) => {
            if (context?.previousQueries) {
                for (const [key, data] of context.previousQueries) {
                    queryClient.setQueryData(key, data);
                }
            }
            toast.error(getErrorMessage(err));
        },
        onSettled: () => {
            queryClient.invalidateQueries({ queryKey: academicYearKeys.all });
        },
    });
}

/** Delete an academic year and its cascade-deleted terms with optimistic removal. */
export function useDeleteAcademicYear() {
    const queryClient = useQueryClient();
    const router = useRouter();

    return useMutation({
        mutationFn: (id: string) => deleteAcademicYear(id),
        onMutate: async (id) => {
            await queryClient.cancelQueries({ queryKey: academicYearKeys.all });
            const previousQueries = queryClient.getQueriesData<{ items: { id: string }[] }>({
                queryKey: academicYearKeys.all,
            });

            queryClient.setQueriesData<{ items: { id: string }[] }>(
                { queryKey: academicYearKeys.all },
                (old) => {
                    if (!old) return old;
                    return {
                        ...old,
                        items: old.items.filter((item) => item.id !== id),
                    };
                }
            );

            return { previousQueries };
        },
        onError: (err, _id, context) => {
            if (context?.previousQueries) {
                for (const [key, data] of context.previousQueries) {
                    queryClient.setQueryData(key, data);
                }
            }
            toast.error(getErrorMessage(err));
        },
        onSettled: (_data, _err, _id) => {
            queryClient.invalidateQueries({ queryKey: academicYearKeys.all });
        },
        onSuccess: () => {
            router.push("/academic-years");
        },
    });
}

/** Create a new term within an academic year with optimistic update. */
export function useCreateTerm() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (payload: CreateTermPayload) => createTerm(payload),
        onMutate: async () => {
            await queryClient.cancelQueries({ queryKey: academicYearKeys.all });
            await queryClient.cancelQueries({ queryKey: academicTermKeys.all });
            const prevYear = queryClient.getQueriesData({
                queryKey: academicYearKeys.all,
            });
            const prevTerm = queryClient.getQueriesData({
                queryKey: academicTermKeys.all,
            });
            return { previousQueries: [...prevYear, ...prevTerm] };
        },
        onError: (err, _payload, context) => {
            if (context?.previousQueries) {
                for (const [key, data] of context.previousQueries) {
                    queryClient.setQueryData(key, data);
                }
            }
            toast.error(getErrorMessage(err));
        },
        onSettled: () => {
            queryClient.invalidateQueries({ queryKey: academicYearKeys.all });
            queryClient.invalidateQueries({ queryKey: academicTermKeys.all });
        },
    });
}

/** Update an existing term with optimistic update. */
export function useUpdateTerm() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ id, payload }: { id: string; payload: UpdateTermPayload }) =>
            updateTerm(id, payload),
        onMutate: async ({ id, payload }) => {
            await queryClient.cancelQueries({ queryKey: academicYearKeys.all });
            await queryClient.cancelQueries({ queryKey: academicTermKeys.all });
            const prevYear = queryClient.getQueriesData({
                queryKey: academicYearKeys.all,
            });
            const prevTerm = queryClient.getQueriesData({
                queryKey: academicTermKeys.all,
            });

            queryClient.setQueriesData<{ items: { id: string }[] }>(
                { queryKey: academicTermKeys.all },
                (old) => {
                    if (!old) return old;
                    return {
                        ...old,
                        items: old.items.map((item) =>
                            item.id === id ? { ...item, ...payload } : item
                        ),
                    };
                }
            );

            return { previousQueries: [...prevYear, ...prevTerm] };
        },
        onError: (err, _vars, context) => {
            if (context?.previousQueries) {
                for (const [key, data] of context.previousQueries) {
                    queryClient.setQueryData(key, data);
                }
            }
            toast.error(getErrorMessage(err));
        },
        onSettled: () => {
            queryClient.invalidateQueries({ queryKey: academicYearKeys.all });
            queryClient.invalidateQueries({ queryKey: academicTermKeys.all });
        },
    });
}
