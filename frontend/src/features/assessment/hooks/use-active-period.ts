/**
 * Hook for fetching the active (current) academic year and term.
 *
 * Uses the backend /api/v1/academic-years endpoint which returns
 * years with an `is_current` boolean flag, and /api/v1/academic-terms
 * for terms within that year.
 */

"use client";

import { useQuery } from "@tanstack/react-query";

import { listAcademicYears, listTerms } from "@/lib/api/academic-terms";
import type { AcademicYear, AcademicTerm } from "@/lib/api/academic-terms";

// ─── Query keys ───────────────────────────────────────────────────────────

const academicKeys = {
    all: ["academic"] as const,
    years: () => [...academicKeys.all, "years"] as const,
    terms: (yearId: string) => [...academicKeys.all, "terms", yearId] as const,
};

// ─── Hooks ────────────────────────────────────────────────────────────────

/**
 * Fetch all academic years for the active school and find the current one.
 */
export function useActiveAcademicYear() {
    return useQuery({
        queryKey: academicKeys.years(),
        queryFn: async (): Promise<AcademicYear | null> => {
            const { data: years } = await listAcademicYears();
            return years.find((y) => y.is_current) ?? years[0] ?? null;
        },
        staleTime: 30_000, // cache for 30s
    });
}

/**
 * Fetch terms for a given academic year and find the current one.
 */
export function useActiveTerm(academicYearId: string | undefined) {
    return useQuery({
        queryKey: academicKeys.terms(academicYearId ?? ""),
        queryFn: async (): Promise<AcademicTerm | null> => {
            if (!academicYearId) return null;
            const { data: terms } = await listTerms({ academic_year_id: academicYearId });
            return terms.find((t) => t.is_current) ?? terms[0] ?? null;
        },
        enabled: !!academicYearId,
        staleTime: 30_000,
    });
}
