import { QueryClient } from "@tanstack/react-query";

/**
 * Creates a new QueryClient with default options.
 * Mutations retry once on failure; queries are stale immediately.
 */
export function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      mutations: {
        retry: 1,
      },
      queries: {
        staleTime: 0,
        gcTime: 1000 * 60 * 5, // 5 minutes
      },
    },
  });
}
