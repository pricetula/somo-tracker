"use client";

import { useQuery } from "@tanstack/react-query";
import { listAcademicYears, AcademicYear } from "@/lib/api/academic-terms";

// ─── Query keys ───────────────────────────────────────────────────────────

export const academicYearsKeys = {
    list: ["academicYears"] as const,
};

// ─── Hooks ────────────────────────────────────────────────────────────────

/** Fetch the list of academic years for the active school. */
export function useAcademicYears() {
    return useQuery<AcademicYear[], Error>({
        queryKey: academicYearsKeys.list,
        queryFn: async () => {
            const response = await listAcademicYears();
            return response?.data || [];
        },
        // Optionally, you can adjust staleTime, retry, etc.
        staleTime: 60 * 1000, // 1 minute
    });
}
