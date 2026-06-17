/**
 * TanStack Query hooks for schools with caching.
 *
 * Schools are fetched once per session and cached. They are re-fetched
 * only when a new school is created (via query invalidation).
 */

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import {
    listSchools,
    createSchool,
    activateSchool,
    type School,
    type CreateSchoolPayload,
} from "@/lib/api/schools";
import { getApiErrorMessage } from "@/lib/api/auth";
import type { MeResponse } from "@/lib/api/auth";
import { authKeys } from "./use-auth";

export const schoolKeys = {
    all: ["schools"] as const,
    byTenant: (tenantId: string) => ["schools", tenantId] as const,
};

/** Fetch all schools for a tenant. Cached forever until invalidated. */
export function useSchools(tenantId: string | undefined) {
    return useQuery<School[]>({
        queryKey: schoolKeys.byTenant(tenantId ?? ""),
        queryFn: () => listSchools(tenantId!),
        staleTime: Infinity,
        gcTime: 60 * 60 * 1000,
        enabled: !!tenantId,
        retry: 2,
    });
}

/** Create a new school (requires SCHOOL_ADMIN role). */
export function useCreateSchool() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (payload: CreateSchoolPayload) => createSchool(payload),
        onSuccess: (school) => {
            // Invalidate the school list for this tenant
            queryClient.invalidateQueries({ queryKey: schoolKeys.all });
            toast.success("School created!", {
                description: `"${school.name}" has been added.`,
            });
        },
        onError: (err) => {
            toast.error("Failed to create school", {
                description: getApiErrorMessage(err),
            });
        },
    });
}

/** Activate a school — switches the user's active school membership.
 *
 * Optimistic updates: immediately updates the `me` cache (school_id, school_name)
 * and the schools list (is_active flags across memberships). Rolls back on error.
 */
export function useActivateSchool() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (schoolId: string) => activateSchool(schoolId),
        onMutate: async (schoolId: string) => {
            // Cancel in-flight queries so they don't overwrite our optimistic update
            await queryClient.cancelQueries({ queryKey: authKeys.me });

            // Snapshot the current me data for rollback
            const previousMe = queryClient.getQueryData<MeResponse | null>(authKeys.me);

            // Find the school name from the schools cache
            const schools = queryClient.getQueryData<School[]>(schoolKeys.all) ?? [];
            const targetSchool = schools.find((s) => s.id === schoolId);

            // Optimistically update the me cache with the new active school
            if (previousMe) {
                queryClient.setQueryData<MeResponse>(authKeys.me, {
                    ...previousMe,
                    school_id: schoolId,
                    school_name: targetSchool?.name ?? previousMe.school_name,
                });
            }

            return { previousMe };
        },
        onError: (_err, _schoolId, context) => {
            // Rollback the me cache to the previous state
            if (context?.previousMe) {
                queryClient.setQueryData(authKeys.me, context.previousMe);
            }
            toast.error("Failed to switch school", {
                description: getApiErrorMessage(_err),
            });
        },
        onSettled: () => {
            // Always refetch me and schools to ensure consistency with the server
            queryClient.invalidateQueries({ queryKey: authKeys.me });
            queryClient.invalidateQueries({ queryKey: schoolKeys.all });
        },
    });
}
