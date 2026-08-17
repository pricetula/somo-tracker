"use client";

import { useQuery } from "@tanstack/react-query";
import { getCurrentYearAndTerm } from "@/lib/api/academic-terms";
import {
    getTeacherDeliveryBreakdown,
    TeacherDeliveryBreakdownList,
} from "@/lib/api/teacher-delivery";
import { STALE_TIMES } from "@/lib/query-config";

// ─── Query keys ───────────────────────────────────────────────────────────

export const teacherDeliveryKeys = {
    currentTerm: ["currentAcademicTermId"] as const,
    breakdown: (termId: string | undefined) => ["teacherDeliveryBreakdown", termId] as const,
};

// ─── Hooks ────────────────────────────────────────────────────────────────

/**
 * Resolve the active academic term id for the current school.
 *
 * Backed by GET /api/v1/academic-years/current. Used as a zero-config
 * fallback so the dashboard chart renders for the active term without the
 * caller having to plumb term selection through props.
 */
export function useCurrentTermId(enabled = true) {
    return useQuery<string | undefined, Error>({
        queryKey: teacherDeliveryKeys.currentTerm,
        queryFn: async () => {
            const current = await getCurrentYearAndTerm();
            return current.academic_term_id;
        },
        enabled,
        staleTime: STALE_TIMES.REFERENCE_DATA,
    });
}

/**
 * Fetch per-teacher Marked vs. Missed slot counts for a school term
 * (School Administrator dashboard grouped bar chart).
 *
 * The query stays disabled until a term id is available; the backend orders
 * items by missed slot count descending so chronic non-compliant teachers
 * surface first (compliance watch).
 */
export function useTeacherDeliveryBreakdown(termId?: string) {
    return useQuery<TeacherDeliveryBreakdownList, Error>({
        queryKey: teacherDeliveryKeys.breakdown(termId),
        queryFn: () => getTeacherDeliveryBreakdown(termId as string),
        enabled: Boolean(termId),
        staleTime: STALE_TIMES.FREQUENT,
    });
}
