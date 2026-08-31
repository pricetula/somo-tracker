"use client";

import { useQuery } from "@tanstack/react-query";
import {
    getLearningAreaAttendanceBreakdowns,
    LearningAreaAttendanceBreakdownList,
} from "@/lib/api/attendance";
import { STALE_TIMES } from "@/lib/query-config";

// ─── Query keys ───────────────────────────────────────────────────────────

export const learningAreaAttendanceBreakdownKeys = {
    all: ["learningAreaAttendanceBreakdowns"] as const,
};

// ─── Hooks ────────────────────────────────────────────────────────────────

/**
 * Fetch per-learning-area attendance counts for the current active term.
 */
export function useLearningAreaAttendanceBreakdowns() {
    return useQuery<LearningAreaAttendanceBreakdownList, Error>({
        queryKey: learningAreaAttendanceBreakdownKeys.all,
        queryFn: () => getLearningAreaAttendanceBreakdowns(),
        staleTime: STALE_TIMES.FREQUENT,
    });
}
