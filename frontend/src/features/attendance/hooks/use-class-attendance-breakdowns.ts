"use client";

import { useQuery } from "@tanstack/react-query";
import { ClassAttendanceBreakdownList, getClassAttendanceBreakdowns } from "@/lib/api/attendance";
import { STALE_TIMES } from "@/lib/query-config";

// ─── Query keys ───────────────────────────────────────────────────────────

export const classAttendanceBreakdownsKeys = {
    all: ["classAttendanceBreakdowns"] as const,
};

// ─── Hooks ────────────────────────────────────────────────────────────────

/**
 * Fetch per-class Present/Late/Absent counts for the current active term
 * (School Administrator dashboard grouped bar chart).
 * Backend resolves the current term server-side.
 */
export function useClassAttendanceBreakdowns() {
    return useQuery<ClassAttendanceBreakdownList, Error>({
        queryKey: classAttendanceBreakdownsKeys.all,
        queryFn: () => getClassAttendanceBreakdowns(),
        staleTime: STALE_TIMES.FREQUENT,
    });
}
