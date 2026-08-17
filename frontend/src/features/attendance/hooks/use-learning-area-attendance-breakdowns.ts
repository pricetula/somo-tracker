"use client";

import { useQuery } from "@tanstack/react-query";
import { getCurrentYearAndTerm } from "@/lib/api/academic-terms";
import {
    getLearningAreaAttendanceBreakdowns,
    LearningAreaAttendanceBreakdownList,
} from "@/lib/api/attendance";
import { STALE_TIMES } from "@/lib/query-config";

// ─── Query keys ───────────────────────────────────────────────────────────

export const learningAreaAttendanceBreakdownKeys = {
    currentTerm: ["currentAcademicTermId"] as const,
    get: (termId: string | undefined) => ["learningAreaAttendanceBreakdowns", termId] as const,
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
        queryKey: learningAreaAttendanceBreakdownKeys.currentTerm,
        queryFn: async () => {
            const current = await getCurrentYearAndTerm();
            return current.academic_term_id;
        },
        enabled,
        staleTime: STALE_TIMES.REFERENCE_DATA,
    });
}

/**
 * Fetch per-learning-area Present/Absent/Excused period counts for a school
 * term (School Administrator dashboard grouped bar chart).
 *
 * The query stays disabled until a term id is available; the backend orders
 * items by absent period count descending so subjects with the highest
 * truancy / absenteeism surface first (hotspot watch).
 */
export function useLearningAreaAttendanceBreakdowns(termId?: string) {
    return useQuery<LearningAreaAttendanceBreakdownList, Error>({
        queryKey: learningAreaAttendanceBreakdownKeys.get(termId),
        queryFn: () => getLearningAreaAttendanceBreakdowns(termId as string),
        enabled: Boolean(termId),
        staleTime: STALE_TIMES.FREQUENT,
    });
}
