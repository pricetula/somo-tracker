"use client";

import { useQuery } from "@tanstack/react-query";
import { getTeacherSummary, TeacherSummary } from "@/lib/api/teachers";

export const teacherSummaryKeys = {
    summary: () => ["teachers", "summary"] as const,
};

export function useTeacherSummary() {
    return useQuery<TeacherSummary, Error>({
        queryKey: teacherSummaryKeys.summary(),
        queryFn: async () => getTeacherSummary(),
    });
}
