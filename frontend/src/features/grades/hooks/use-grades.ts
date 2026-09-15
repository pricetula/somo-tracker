/**
 * Grades feature — hooks for grades query.
 */

"use client";

import { useQuery } from "@tanstack/react-query";
import { listGrades } from "../services/api";
import type { GradeLevel } from "../types/grade";
import { useMeSession } from "@/features/auth/hooks/use-me-session";

export const gradesKeys = {
    list: ["grades", "list"] as const,
};

/**
 * Hook to fetch grade levels for the active school.
 */
export function useGrades() {
    const { data: me } = useMeSession();

    return useQuery<GradeLevel[], Error>({
        queryKey: gradesKeys.list,
        queryFn: async () => {
            if (!me?.active_school_id) {
                return [];
            }
            const res = await listGrades();
            return res?.grades ?? [];
        },
        enabled: !!me?.active_school_id,
        retry: 0,
    });
}
