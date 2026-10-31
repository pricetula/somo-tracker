/**
 * Hook for auth state
 * Uses server function to get cookie presence on server
 */

import { useQuery } from '@tanstack/react-query';
import { getAuthState } from '../api/get-auth-state';

export function useAuthState() {
  return useQuery({
    queryKey: ['auth', 'state'],
    queryFn: () => getAuthState(),
    staleTime: 1000 * 60 * 5, // 5 minutes
    gcTime: 1000 * 60 * 10, // 10 minutes
  });
}