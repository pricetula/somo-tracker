import { useQuery } from "@tanstack/react-query";
import { getSetupSummary } from "../services/api";

export function useSetupSummary() {
    return useQuery({
        queryKey: ["dashboard", "setup-summary"],
        queryFn: getSetupSummary,
        staleTime: 60_000,
    });
}
