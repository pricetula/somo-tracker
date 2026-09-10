"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { toast } from "sonner";

import { getErrorMessage } from "@/lib/errors";

import { sendMagicLink, logout, type MagicLinkResponse, type LogoutResponse } from "@/lib/api/auth";

// ─── Query keys ───────────────────────────────────────────────────────────

export const authKeys = {
    me: ["auth", "me"] as const,
};

/** Fetch the current user session. Returns null when not authenticated.
 *  Note: Backend doesn't have a /me endpoint, so this will always return null.
 *  Use this only if backend adds a /me endpoint in the future. */
export function useMe() {
    return useQuery<null>({
        queryKey: authKeys.me,
        queryFn: async () => null,
        enabled: false,
    });
}

/** PHASE 1: Send a magic link to the given email. */
export function useSendMagicLink() {
    return useMutation<MagicLinkResponse, Error, string>({
        mutationFn: (email: string) => sendMagicLink(email),
        onSuccess: (_data, email) => {
            toast.success("Magic link sent!", {
                description: `Check ${email} for your sign-in link.`,
            });
        },
        onError: (err) => {
            toast.error("Failed to send magic link", {
                description: getErrorMessage(err),
            });
        },
    });
}

/** Logout: destroy session and redirect to login. */
export function useLogout() {
    const queryClient = useQueryClient();
    const router = useRouter();

    return useMutation<LogoutResponse, Error, void>({
        mutationFn: () => logout(),
        onSuccess: async () => {
            // Clear all cached queries so no stale data leaks across sessions
            queryClient.clear();
            toast.success("Logged out");
            router.push("/login");
        },
        onError: (err) => {
            toast.error("Logout failed", {
                description: getErrorMessage(err),
            });
            // Still redirect to login to clear local state
            router.push("/login");
        },
    });
}
