/**
 * Hook for logout
 * POST with JSON body via button onClick fetch, credentials:'include'
 * Never triggered by TanStack Router preloading
 */

import { useMutation } from '@tanstack/react-query';
import { logout } from '../api/logout';
import { authMutationKeys } from '../query-keys';
import { useNavigate } from '@tanstack/react-router';
import { queryClient } from '@/lib/query-client';
import { toast } from 'sonner';

export function useLogout() {
  const navigate = useNavigate();

  return useMutation({
    mutationKey: authMutationKeys.logout(''),
    mutationFn: logout,
    onSuccess: () => {
      // Clear all cached data
      queryClient.clear();
      toast.success('Logged out successfully');
      // Navigate to login - use replace to avoid back button issues
      navigate({ to: '/login', replace: true });
    },
    onError: (error: Error) => {
      toast.error(error.message || 'Logout failed');
    },
  });
}