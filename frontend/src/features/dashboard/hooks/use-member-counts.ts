"use client";

import { useQuery } from "@tanstack/react-query";
import { getMemberCounts, MemberCounts } from "@/lib/api/members";

// ─── Query keys ───────────────────────────────────────────────────────────

export const memberCountsKeys = {
    get: ["memberCounts"] as const,
};

// ─── Hooks ────────────────────────────────────────────────────────────────

/** Fetch aggregate member counts (students, admins, nurses, teachers, parents, finance). */
export function useMemberCounts() {
    return useQuery<MemberCounts, Error>({
        queryKey: memberCountsKeys.get,
        queryFn: async () => {
            const response = await getMemberCounts();
            return response?.data;
        },
        staleTime: 60 * 1000, // 1 minute
    });
}
