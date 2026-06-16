"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { toast } from "sonner";

import {
  discover,
  getMe,
  logout,
  register,
  verifyToken,
  getApiErrorMessage,
  isApiError,
  type MeResponse,
  type RegisterPayload,
} from "@/lib/api/auth";

// ─── Query keys ───────────────────────────────────────────────────────────

export const authKeys = {
  me: ["auth", "me"] as const,
};

// ─── Hooks ────────────────────────────────────────────────────────────────

/** Fetch the current user session. Returns null when not authenticated. */
export function useMe() {
  return useQuery<MeResponse | null>({
    queryKey: authKeys.me,
    queryFn: async () => {
      try {
        return await getMe();
      } catch {
        return null;
      }
    },
    retry: false,
  });
}

/** PHASE 1: Send a magic link to the given email. */
export function useDiscover() {
  return useMutation({
    mutationFn: (email: string) => discover(email),
    onSuccess: (_data, email) => {
      toast.success("Magic link sent!", {
        description: `Check ${email} for your sign-in link.`,
      });
    },
    onError: (err) => {
      toast.error("Failed to send magic link", {
        description: getApiErrorMessage(err),
      });
    },
  });
}

/** PHASE 2: Verify a magic-link token. */
export function useVerifyToken() {
  return useMutation({
    mutationFn: (token: string) => verifyToken(token),
    onError: (err) => {
      const msg = getApiErrorMessage(err);
      if (msg.includes("expired")) {
        toast.error("Link expired", {
          description: "This magic link has expired. Please request a new one.",
        });
      } else {
        toast.error("Verification failed", {
          description: msg,
        });
      }
    },
  });
}

/** PHASE 3: Register (create tenant + user + session). */
export function useRegister() {
  const queryClient = useQueryClient();
  const router = useRouter();

  return useMutation({
    mutationFn: (payload: RegisterPayload) => register(payload),
    onSuccess: async () => {
      // Invalidate the me query so it re-fetches with the new session cookie
      await queryClient.invalidateQueries({ queryKey: authKeys.me });
      toast.success("Account created!", {
        description: "Welcome to Somotracker.",
      });
      router.push("/");
    },
    onError: (err) => {
      // 401 means the session_ref is expired or already consumed —
      // redirect to login so the user can request a new magic link
      if (isApiError(err) && err.status === 401) {
        toast.error("Link expired", {
          description: "This registration session has expired. Please request a new magic link.",
        });
        router.replace("/login");
        return;
      }
      toast.error("Registration failed", {
        description: getApiErrorMessage(err),
      });
    },
  });
}

/** Logout: destroy session and redirect to login. */
export function useLogout() {
  const queryClient = useQueryClient();
  const router = useRouter();

  return useMutation({
    mutationFn: () => logout(),
    onSuccess: async () => {
      // Clear all cached queries so no stale data leaks across sessions
      queryClient.clear();
      toast.success("Logged out");
      router.push("/login");
    },
    onError: (err) => {
      toast.error("Logout failed", {
        description: getApiErrorMessage(err),
      });
    },
  });
}
