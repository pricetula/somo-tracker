"use client";

import { useQuery } from "@tanstack/react-query";
import { getGuardianSummary, GuardianSummary } from "@/lib/api/guardians";

export const guardianSummaryKeys = {
    summary: () => ["guardians", "summary"] as const,
};

export function useGuardianSummary() {
    return useQuery<GuardianSummary, Error>({
        queryKey: guardianSummaryKeys.summary(),
        queryFn: async () => getGuardianSummary(),
    });
}
