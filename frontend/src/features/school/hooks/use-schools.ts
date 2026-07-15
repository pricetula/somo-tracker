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

/** Switch the user's active school. Invalidates the me query to refresh cookies. */
export function useSetActiveSchool() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (schoolId: string) => setActiveSchool(schoolId),
        onSuccess: () => {
            // Invalidate the me query so it re-fetches with the updated
            // somo_school_id cookie and refreshes the active school info.
            queryClient.invalidateQueries({ queryKey: authKeys.me });
            queryClient.invalidateQueries({ queryKey: schoolKeys.list() });
            toast.success("School switched");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}

/** Update a school's details. */
export function useUpdateSchool() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ id, payload }: { id: string; payload: { name?: string; code?: string } }) =>
            updateSchool(id, payload),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: schoolKeys.all });
            queryClient.invalidateQueries({ queryKey: authKeys.me });
            toast.success("School updated");
        },
        onError: (err) => toast.error(getErrorMessage(err)),
    });
}

/** Delete a school. */
export function useDeleteSchool() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (id: string) => deleteSchool(id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: schoolKeys.all });
            queryClient.invalidateQueries({ queryKey: authKeys.me });
            toast.success("School deleted");
        },
        onError: (err) => toast.error(getErrorMessage(err)),
    });
}

/** Create a new school. Invalidates both the schools list and the me query. */
export function useCreateSchool() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: CreateSchoolPayload) => createSchool(data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: schoolKeys.all });
            queryClient.invalidateQueries({ queryKey: authKeys.me });
            toast.success("School created");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}
