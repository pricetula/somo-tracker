"use client";

import { useQuery } from "@tanstack/react-query";
import { getStudentSummary, StudentSummary } from "@/lib/api/students";

export const studentSummaryKeys = {
    summary: () => ["students", "summary"] as const,
};

export function useStudentSummary() {
    return useQuery<StudentSummary, Error>({
        queryKey: studentSummaryKeys.summary(),
        queryFn: async () => getStudentSummary(),
    });
}
