import { useQuery } from "@tanstack/react-query";
import { listCountries } from "../api/countries.api";
import { countryKeys } from "../api/countries.keys";
import type { Country } from "../types/countries";

/**
 * useCountries fetches all countries via TanStack Query.
 */
export function useCountries() {
  return useQuery<Country[], Error>({
    queryKey: countryKeys.list(),
    queryFn: listCountries,
    staleTime: 5 * 60 * 1000, // 5 minutes
  });
}
