"use client";

import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api/client";
import type { UserCounts } from "../types/user-counts";

export function useUserCounts(schoolId?: string) {
    return useQuery<UserCounts>({
        queryKey: ["userCounts", schoolId],
        queryFn: async () => {
            if (!schoolId) throw new Error("schoolId is required");
            return api.get<UserCounts>(`/api/schools/${schoolId}/users/count`);
        },
        enabled: !!schoolId,
        staleTime: 60_000,
    });
}
