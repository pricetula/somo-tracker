/**
 * TanStack Query hooks for the School feature.
 *
 * Covers listing schools and creating a new school.
 */

"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import {
    listSchools,
    createSchool,
    updateSchool,
    deleteSchool,
    setActiveSchool,
} from "@/lib/api/schools";
import { getErrorMessage } from "@/lib/errors";
import { authKeys } from "@/hooks/use-auth";
import type { ListSchoolsResponse, CreateSchoolPayload } from "../types";

// ─── Query keys ───────────────────────────────────────────────────────────

export const schoolKeys = {
    all: ["schools"] as const,
    list: () => [...schoolKeys.all, "list"] as const,
};

// ─── Hooks: Schools List ─────────────────────────────────────────────────

/** Fetch the list of schools available in the current tenant. */
export function useSchools(opts: { enabled?: boolean } = {}) {
    const { enabled = true } = opts;

    return useQuery<ListSchoolsResponse>({
        queryKey: schoolKeys.list(),
        queryFn: () => listSchools(),
        placeholderData: (prev) => prev,
        enabled,
    });
}

// ─── Mutations ────────────────────────────────────────────────────────────

/** Switch the user's active school with optimistic update. Invalidates the me query to refresh cookies. */
export function useSetActiveSchool() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (schoolId: string) => setActiveSchool(schoolId),
        onMutate: async () => {
            await queryClient.cancelQueries({ queryKey: authKeys.me });
            await queryClient.cancelQueries({ queryKey: schoolKeys.all });
            const prevAuth = queryClient.getQueriesData({
                queryKey: authKeys.me,
            });
            const prevSchool = queryClient.getQueriesData({
                queryKey: schoolKeys.all,
            });
            return { previousQueries: [...prevAuth, ...prevSchool] };
        },
        onError: (err, _schoolId, context) => {
            if (context?.previousQueries) {
                for (const [key, data] of context.previousQueries) {
                    queryClient.setQueryData(key, data);
                }
            }
            toast.error(getErrorMessage(err));
        },
        onSettled: () => {
            queryClient.invalidateQueries({ queryKey: authKeys.me });
            queryClient.invalidateQueries({ queryKey: schoolKeys.all });
        },
    });
}

/** Update a school's details with optimistic update. */
export function useUpdateSchool() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ id, payload }: { id: string; payload: { name?: string; code?: string } }) =>
            updateSchool(id, payload),
        onMutate: async ({ id, payload }) => {
            await queryClient.cancelQueries({ queryKey: schoolKeys.all });
            const previousQueries = queryClient.getQueriesData<ListSchoolsResponse>({
                queryKey: schoolKeys.all,
            });

            queryClient.setQueriesData<ListSchoolsResponse>({ queryKey: schoolKeys.all }, (old) => {
                if (!old) return old;
                return {
                    ...old,
                    items: old.items.map((s) => (s.id === id ? { ...s, ...payload } : s)),
                };
            });

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
            queryClient.invalidateQueries({ queryKey: schoolKeys.all });
            queryClient.invalidateQueries({ queryKey: authKeys.me });
        },
    });
}

/** Delete a school with optimistic removal. */
export function useDeleteSchool() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (id: string) => deleteSchool(id),
        onMutate: async (id) => {
            await queryClient.cancelQueries({ queryKey: schoolKeys.all });
            const previousQueries = queryClient.getQueriesData<ListSchoolsResponse>({
                queryKey: schoolKeys.all,
            });

            queryClient.setQueriesData<ListSchoolsResponse>({ queryKey: schoolKeys.all }, (old) => {
                if (!old) return old;
                return {
                    ...old,
                    items: old.items.filter((item) => item.id !== id),
                };
            });

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
            queryClient.invalidateQueries({ queryKey: schoolKeys.all });
            queryClient.invalidateQueries({ queryKey: authKeys.me });
        },
    });
}

/** Create a new school with optimistic update. Invalidates both the schools list and the me query. */
export function useCreateSchool() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: CreateSchoolPayload) => createSchool(data),
        onMutate: async () => {
            await queryClient.cancelQueries({ queryKey: schoolKeys.all });
            const previousQueries = queryClient.getQueriesData({
                queryKey: schoolKeys.all,
            });
            return { previousQueries };
        },
        onError: (err, _data, context) => {
            if (context?.previousQueries) {
                for (const [key, data] of context.previousQueries) {
                    queryClient.setQueryData(key, data);
                }
            }
            toast.error(getErrorMessage(err));
        },
        onSettled: () => {
            queryClient.invalidateQueries({ queryKey: schoolKeys.all });
            queryClient.invalidateQueries({ queryKey: authKeys.me });
        },
    });
}
