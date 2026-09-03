/**
 * React Query hook for a teacher's weekly lesson timeline.
 * Uses infinite query for scroll-through weeks.
 */
"use client";

import { useInfiniteQuery } from "@tanstack/react-query";
import { addWeeks, startOfWeek, formatISO } from "date-fns";
import { api } from "@/lib/api/client";

// ─── Types ────────────────────────────────────────────────────────────────

export interface LessonEntry {
    id: string;
    subject_name: string;
    subject_id: string;
    class_name: string;
    class_id: string;
    period_name: string;
    start_time: string; // ISO datetime string
    end_time: string; // ISO datetime string
    room?: string;
    is_break?: boolean;
}

export interface LessonTimelinePage {
    entries: LessonEntry[];
    next_cursor?: string; // week offset as string
}

interface TeacherLessonTimelineFilters {
    teacherId?: string;
}

// ─── Query Key ────────────────────────────────────────────────────────────

export const teacherLessonTimelineKeys = {
    all: ["teacher-lesson-timeline"] as const,
    list: (filters: TeacherLessonTimelineFilters) =>
        ["teacher-lesson-timeline", "list", filters] as const,
};

// ─── Fetch Function ───────────────────────────────────────────────────────

async function fetchTeacherLessonsPage(
    filters: TeacherLessonTimelineFilters,
    weekOffset: number
): Promise<LessonTimelinePage> {
    const weekStart = startOfWeek(addWeeks(new Date(), weekOffset), { weekStartsOn: 1 });
    const params = new URLSearchParams({
        week_start: formatISO(weekStart),
        limit: "20",
    });

    const data = await api.get<LessonTimelinePage>(
        `/api/v1/teachers/${encodeURIComponent(filters.teacherId ?? "")}/lessons?${params.toString()}`
    );
    return data;
}

// ─── Hook ─────────────────────────────────────────────────────────────────

export function useTeacherLessonTimeline(filters: TeacherLessonTimelineFilters = {}) {
    return useInfiniteQuery({
        queryKey: teacherLessonTimelineKeys.list(filters),
        queryFn: ({ pageParam }) =>
            fetchTeacherLessonsPage(filters, parseInt(pageParam ?? "0", 10)),
        initialPageParam: "0",
        getNextPageParam: (lastPage) => lastPage.next_cursor ?? undefined,
        enabled: !!filters.teacherId,
    });
}
