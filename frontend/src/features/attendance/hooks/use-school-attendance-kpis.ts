"use client";

import { useQuery } from "@tanstack/react-query";
import { getSchoolAttendanceKPIs, SchoolAttendanceKPI } from "@/lib/api/attendance";
import { STALE_TIMES } from "@/lib/query-config";

// ─── Query keys ───────────────────────────────────────────────────────────

export const schoolAttendanceKPIsKeys = {
    get: (date: string) => ["schoolAttendanceKPIs", date] as const,
};

// ─── Hooks ────────────────────────────────────────────────────────────────

/**
 * Fetch macro-level school attendance KPIs for the active school on a date
 * (School Attendance Command Center). Attendance changes as teachers mark
 * throughout the day, so the cache is treated as frequently-updated data.
 */
export function useSchoolAttendanceKPIs(date: string) {
    return useQuery<SchoolAttendanceKPI, Error>({
        queryKey: schoolAttendanceKPIsKeys.get(date),
        queryFn: () => getSchoolAttendanceKPIs(date),
        staleTime: STALE_TIMES.FREQUENT,
        enabled: Boolean(date),
    });
}
