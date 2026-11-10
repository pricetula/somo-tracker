/**
 * Active school hook
 * Returns the currently active school ID
 * The active school is backend state - read via API
 */

import { useQuery } from '@tanstack/react-query';
import { apiFetch } from '@/lib/api/client';

export interface ActiveSchool {
  id: string;
  name: string;
  code: string;
}

export function useActiveSchool() {
  const { data, isLoading, error } = useQuery({
    queryKey: ['active-school'],
    queryFn: async () => {
      const response = await apiFetch<ActiveSchool>('/me/active-school');
      return response;
    },
    staleTime: 1000 * 60 * 5, // 5 minutes
    gcTime: 1000 * 60 * 30, // 30 minutes
  });

  return {
    schoolId: data?.id,
    school: data,
    isLoading,
    error,
  };
}