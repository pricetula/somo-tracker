"use client";

import { useQuery } from "@tanstack/react-query";
import { getLowestAttendanceStudents, LowestAttendanceStudent } from "@/lib/api/attendance";
import { STALE_TIMES } from "@/lib/query-config";

// ─── Query keys ───────────────────────────────────────────────────────────

export const lowestAttendanceStudentsKeys = {
    get: (limit?: number) => ["lowestAttendanceStudents", limit ?? 5] as const,
};

// ─── Hooks ────────────────────────────────────────────────────────────────

/**
 * Fetch the N students with the lowest attendance percentage for the current week.
 * Attendance changes as teachers mark throughout the week, so the cache is treated
 * as frequently-updated data.
 */
export function useLowestAttendanceStudents(limit?: number) {
    return useQuery<LowestAttendanceStudent[], Error>({
        queryKey: lowestAttendanceStudentsKeys.get(limit),
        queryFn: () => getLowestAttendanceStudents(limit),
        staleTime: STALE_TIMES.FREQUENT,
    });
}
