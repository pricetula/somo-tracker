"use client";

import { useQuery } from "@tanstack/react-query";
import { listSchools, type SchoolItem } from "@/lib/api/schools";

export const schoolsQueryKeys = {
    list: ["schools"] as const,
};

export function useSchoolsList() {
    return useQuery<SchoolItem[]>({
        queryKey: schoolsQueryKeys.list,
        queryFn: async () => {
            const res = await listSchools();
            return res.schools;
        },
        staleTime: 5 * 60 * 1000,
    });
}
