"use client";

import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api/client";
import type { UserCounts } from "../types/user-counts";

export function useUserCounts() {
    return useQuery<UserCounts>({
        queryKey: ["userCounts"],
        queryFn: async () => {
            return api.get<UserCounts>(`/api/school/users/count`);
        },
        staleTime: 60_000,
    });
}
