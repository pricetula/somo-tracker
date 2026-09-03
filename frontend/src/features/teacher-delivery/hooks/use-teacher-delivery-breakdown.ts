"use client";

import { useQuery } from "@tanstack/react-query";
import {
    getTeacherDeliveryBreakdown,
    TeacherDeliveryBreakdownList,
} from "@/lib/api/teacher-delivery";
import { STALE_TIMES } from "@/lib/query-config";

// ─── Query keys ───────────────────────────────────────────────────────────

export const teacherDeliveryKeys = {
    breakdown: ["teacherDeliveryBreakdown"] as const,
};

// ─── Hooks ────────────────────────────────────────────────────────────────

/**
 * Fetch per-teacher Marked vs. Missed slot counts for the current active term.
 */
export function useTeacherDeliveryBreakdown() {
    return useQuery<TeacherDeliveryBreakdownList, Error>({
        queryKey: teacherDeliveryKeys.breakdown,
        queryFn: () => getTeacherDeliveryBreakdown(),
        staleTime: STALE_TIMES.FREQUENT,
    });
}
