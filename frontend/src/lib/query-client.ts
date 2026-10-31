/**
 * TanStack Query client configuration
 * Global error handling for 401 responses
 */

import { QueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 1000 * 60 * 5, // 5 minutes
      gcTime: 1000 * 60 * 30, // 30 minutes
      retry: (failureCount, error) => {
        // Don't retry on 401/403
        if (error && typeof error === 'object' && 'status' in error) {
          const status = (error as { status: number }).status;
          if (status === 401 || status === 403) return false;
        }
        return failureCount < 3;
      },
    },
    mutations: {
      retry: false,
    },
  },
});

// Global mutation error handler - shows toast for all mutation errors
queryClient.getMutationCache().config.onError = (error) => {
  // Don't show toast for 401 - handled by global 401 handler
  if (error && typeof error === 'object' && 'status' in error) {
    const status = (error as { status: number }).status;
    if (status === 401) return;
  }

  const message = error instanceof Error ? error.message : 'An error occurred';
  toast.error(message);
};