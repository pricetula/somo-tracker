import { useQuery } from "@tanstack/react-query";
import { listEducationSystems } from "../api/education-systems.api";
import { educationSystemKeys } from "../api/education-systems.keys";
import type { EducationSystem } from "../types/education-systems";

/**
 * useEducationSystems fetches all education systems via TanStack Query.
 */
export function useEducationSystems() {
  return useQuery<EducationSystem[], Error>({
    queryKey: educationSystemKeys.list(),
    queryFn: listEducationSystems,
    staleTime: 5 * 60 * 1000, // 5 minutes
  });
}
