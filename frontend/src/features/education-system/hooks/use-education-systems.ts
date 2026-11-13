import { useQuery } from '@tanstack/react-query';
import { listEducationSystems } from '../api/list-education-systems';
import { educationKeys } from '../query-keys';

export function useEducationSystems(countryId: string | null) {
  return useQuery({
    queryKey: educationKeys.list(countryId ?? ''),
    queryFn: () => listEducationSystems(countryId!),
    enabled: !!countryId,
    staleTime: 1000 * 60 * 5, // 5 minutes
    gcTime: 1000 * 60 * 30, // 30 minutes
  });
}
