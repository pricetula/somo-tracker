import { useQuery } from '@tanstack/react-query';
import { listCountries } from '../api/list-countries';
import { countryKeys } from '../query-keys';

export function useCountries() {
  return useQuery({
    queryKey: countryKeys.list(),
    queryFn: listCountries,
    staleTime: 1000 * 60 * 5, // 5 minutes
    gcTime: 1000 * 60 * 30, // 30 minutes
  });
}
