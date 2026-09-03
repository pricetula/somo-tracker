/**
 * React Query hooks for the Attendance feature.
 */
"use client";

import { useQuery } from "@tanstack/react-query";
import { getCalendarStatus } from "@/lib/api/attendance";

// ─── Query Keys ───────────────────────────────────────────────────────────

export const attendanceKeys = {
    all: ["attendance"] as const,
    sessions: {
        all: ["attendance", "sessions"] as const,
        list: (params?: Record<string, unknown>) =>
            ["attendance", "sessions", "list", params] as const,
        detail: (id: string) => ["attendance", "sessions", id] as const,
        byClassDate: (classId: string, date: string) =>
            ["attendance", "sessions", "class", classId, date] as const,
    },
    records: {
        all: ["attendance", "records"] as const,
        list: (params?: Record<string, unknown>) =>
            ["attendance", "records", "list", params] as const,
        bySlot: (slotId: string, date: string) =>
            ["attendance", "records", "slot", slotId, date] as const,
        byStudent: (studentId: string, termId?: string) =>
            ["attendance", "records", "student", studentId, termId] as const,
        byClassDate: (classId: string, date: string) =>
            ["attendance", "records", "class", classId, date] as const,
    },
    summaries: {
        all: ["attendance", "summaries"] as const,
        byStudent: (studentId: string, termId?: string) =>
            ["attendance", "summaries", "student", studentId, termId] as const,
        byClass: (classId: string, termId?: string) =>
            ["attendance", "summaries", "class", classId, termId] as const,
    },
    calendarStatus: {
        all: ["attendance", "calendar-status"] as const,
        range: (startDate: string, endDate: string) =>
            ["attendance", "calendar-status", startDate, endDate] as const,
    },
};

// ─── Calendar Status ───────────────────────────────────────────────────────

/**
 * Get per-date attendance completion status for a calendar month view.
 * Fetches once per date-range change (not per individual date cell) to avoid N+1.
 */
export function useCalendarStatus(startDate: string, endDate: string, schoolId?: string) {
    return useQuery({
        queryKey: attendanceKeys.calendarStatus.range(startDate, endDate),
        queryFn: () => getCalendarStatus(startDate, endDate),
        enabled: !!startDate && !!endDate && !!schoolId,
        // Keep previous data while fetching new range to avoid flash
        placeholderData: (previousData) => previousData,
    });
}
