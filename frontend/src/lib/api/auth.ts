/**
 * Auth API functions — all calls to the Go backend auth endpoints.
 *
 * Endpoints:
 *   POST   /api/auth/discover   — initiate magic-link flow
 *   POST   /api/auth/verify     — verify magic-link token, return session_ref
 *   POST   /api/auth/register   — complete registration, set session cookie
 *   GET    /api/auth/me         — fetch current session info
 *   POST   /api/auth/switch-school — switch the user's active school
 *   DELETE /api/auth/session    — logout
 */

import { api } from "./client";
import type { VerifyResponse, RegistrationPayload, MeResponse } from "./generated";

// ─── Re-export generated types used by consumers ─────────────────────────

export type {
    DiscoveryPayload,
    VerifyResponse,
    RegistrationPayload,
    MeResponse,
} from "./generated";

/** @deprecated Use RegistrationPayload instead. */
export type RegisterPayload = RegistrationPayload;

// ─── Functions ────────────────────────────────────────────────────────────

/** PHASE 1: Send a magic link to the given email. */
export async function discover(email: string): Promise<void> {
    await api.post("/api/auth/discover", { email });
}

/** PHASE 2: Verify a magic-link token and return the session_ref. */
export async function verifyToken(token: string): Promise<VerifyResponse> {
    return api.post<VerifyResponse>("/api/auth/verify", { token });
}

/** PHASE 3: Complete registration (creates tenant + user + session). */
export async function register(payload: RegistrationPayload): Promise<void> {
    await api.post("/api/auth/register", payload);
}

/** Fetch the current session's user and tenant IDs.
 *
 * Notes:
 * - 401 responses are already handled globally by `client.ts` which
 *   performs a hard redirect to /logout before throwing.
 * - Network errors (backend unreachable) do NOT redirect — they propagate
 *   up so `useMe()` can return null gracefully without forcing logout.
 */
export async function getMe(): Promise<MeResponse> {
    try {
        return await api.get<MeResponse>("/api/auth/me");
    } catch (err) {
        // Network errors (Failed to fetch) should NOT force a logout —
        // the session may still be valid when the backend recovers.
        // Only 401/403 responses trigger the global redirect in client.ts.
        throw err;
    }
}

/** Logout: destroy the current session. */
export async function logout(): Promise<void> {
    await api.delete("/api/auth/session");
}

/**
 * Switch the current user's active school.
 * POST /api/auth/switch-school — body { school_id }.
 * The backend updates the somo_school_id cookie and returns the new school id.
 */
export async function switchSchool(schoolId: string): Promise<{ school_id: string }> {
    return api.post<{ school_id: string }>("/api/auth/switch-school", { school_id: schoolId });
}
