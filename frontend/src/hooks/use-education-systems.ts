/**
 * TanStack Query hook for education systems with infinite cache.
 *
 * Education systems are static reference data (CBC, IGCSE, IB MYP) that
 * almost never change. We use `staleTime: Infinity` so the data is fetched
 * once per session and never re-fetched unless the user hard-refreshes or
 * the query cache is explicitly invalidated.
 */

import { useQuery } from "@tanstack/react-query";

import { listEducationSystems, type EducationSystem } from "@/lib/api/education-systems";

const EDUCATION_SYSTEMS_KEY = ["education-systems"] as const;

/** Options for useEducationSystems. */
export interface UseEducationSystemsOptions {
    /** Enable/disable the query (default: true). */
    enabled?: boolean;
}

/**
 * Returns all available education systems with infinite stale time.
 *
 * The data is cached for the entire browser session and only re-fetched
 * on page reload or manual invalidation via `queryClient.invalidateQueries`.
 */
export function useEducationSystems(opts: UseEducationSystemsOptions = {}) {
    const { enabled = true } = opts;

    return useQuery<EducationSystem[]>({
        queryKey: EDUCATION_SYSTEMS_KEY,
        queryFn: listEducationSystems,
        staleTime: Infinity, // Never re-fetch automatically
        gcTime: 60 * 60 * 1000, // Keep in garbage-collected cache for 1 hour
        retry: 2,
        enabled,
    });
}
