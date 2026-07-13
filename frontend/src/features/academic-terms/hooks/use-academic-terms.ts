/**
 * TanStack Query hooks for academic years and terms.
 *
 * Stable query keys + generous staleTime ensures N simultaneously mounted
 * instances produce exactly ONE network request.
 */

"use client";

import { useQuery } from "@tanstack/react-query";

import { listAcademicYears, listTerms } from "@/lib/api/academic-terms";

// ─── Query keys ───────────────────────────────────────────────────────────

export const academicYearKeys = {
    all: ["academic-years"] as const,
    list: () => [...academicYearKeys.all, "list"] as const,
};

export const academicTermKeys = {
    all: ["academic-terms"] as const,
    list: () => [...academicTermKeys.all, "list"] as const,
};

// ─── Hooks ────────────────────────────────────────────────────────────────

export function useAcademicYears() {
    return useQuery({
        queryKey: academicYearKeys.list(),
        queryFn: () => listAcademicYears(),
        staleTime: 5 * 60 * 1000,
        placeholderData: (prev) => prev,
    });
}

/** Fetch academic terms for the active school. */
export function useAcademicTerms() {
    return useQuery({
        queryKey: academicTermKeys.list(),
        queryFn: () => listTerms(),
        staleTime: 5 * 60 * 1000,
        placeholderData: (prev) => prev,
    });
}
