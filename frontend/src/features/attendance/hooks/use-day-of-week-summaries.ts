"use client";

import { useQuery } from "@tanstack/react-query";
import { DayOfWeekSummaries, getDayOfWeekSummaries } from "@/lib/api/attendance";
import { STALE_TIMES } from "@/lib/query-config";

// ─── Query keys ───────────────────────────────────────────────────────────

export const dayOfWeekSummariesKeys = {
    get: (classId: string | undefined) => ["dayOfWeekSummaries", classId] as const,
};

// ─── Hooks ────────────────────────────────────────────────────────────────

/**
 * Fetch attendance exceptions (absent/late/excused) aggregated by weekday for
 * the current academic year.
 *
 * The query stays enabled with a `class_id` of `undefined`, in which case the
 * backend aggregates across all classes in the tenant ("All" rollup). The
 * backend orders items Monday → Friday ascending, which the weekday
 * stacked-bar chart renders left → right.
 */
export function useDayOfWeekSummaries(classId?: string) {
    return useQuery<DayOfWeekSummaries, Error>({
        queryKey: dayOfWeekSummariesKeys.get(classId),
        queryFn: () => getDayOfWeekSummaries(classId),
        staleTime: STALE_TIMES.FREQUENT,
    });
}
