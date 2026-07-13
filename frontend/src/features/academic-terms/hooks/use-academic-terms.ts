/**
 * useAcademicYears — TanStack Query hook for fetching academic years.
 *
 * Stable query key + generous staleTime ensures N simultaneously mounted
 * instances produce exactly ONE network request.
 */

"use client";

import { useQuery } from "@tanstack/react-query";

import { listAcademicYears } from "@/lib/api/academic-terms";

// ─── Query keys ───────────────────────────────────────────────────────────

export const academicYearKeys = {
    all: ["academic-years"] as const,
    list: () => [...academicYearKeys.all, "list"] as const,
};

// ─── Hook ─────────────────────────────────────────────────────────────────

export function useAcademicYears() {
    return useQuery({
        queryKey: academicYearKeys.list(),
        queryFn: () => listAcademicYears(),
        staleTime: 5 * 60 * 1000,
        placeholderData: (prev) => prev,
    });
}
