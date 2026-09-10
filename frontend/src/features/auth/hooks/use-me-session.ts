import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api/client";
import type { MeResult } from "@/features/auth/lib/types";

/**
 * Client-side hook to access the authenticated user's session data.
 *
 * Uses ``useQuery`` with the ``['me']`` key and a ``staleTime`` of 60 seconds.
 *
 * The hook will return the ``MeResult`` object if:
 *   - The data was hydrated server-side by ``DashboardAuthLayout``, or
 *   - The query executed successfully on the client.
 *
 * If the session is invalid or expired, the underlying API request
 * (triggered automatically by React Query on mount or refetch) will
 * cause a 401 response, which the API client's global error handler
 * will turn into a redirect to ``/logout``.
 *
 * Example:
 *   ```tsx
 *   const { data: me, isLoading } = useMeSession();
 *   if (isLoading) return <Spinner />;
 *   return <div>Hello, {me?.user_name}!</div>;
 *   ```
 */
export function useMeSession() {
    return useQuery<MeResult, Error>({
        queryKey: ["me"],
        queryFn: async () => {
            const res = await api.get<MeResult>("/api/me");
            return res;
        },
        staleTime: 60_000, // 1 minute
        retry: 0,
    });
}
