"use client";

import { useQuery } from "@tanstack/react-query";
import { listAcademicPeriods, type ListAcademicPeriodResponse } from "@/lib/api/academic-terms";

export const academicPeriodsQueryKeys = {
    list: ["academic-periods"] as const,
};

export function useAcademicPeriods() {
    return useQuery<ListAcademicPeriodResponse[]>({
        queryKey: academicPeriodsQueryKeys.list,
        queryFn: async () => {
            const res = await listAcademicPeriods();
            return res.data;
        },
        staleTime: 5 * 60 * 1000,
    });
}
