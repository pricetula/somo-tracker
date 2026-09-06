/**
 * Auth API functions — all calls to the Go backend auth endpoints.
 *
 * Endpoints (matching backend/internal/api/auth_handler.go):
 *   POST   /api/auth/magic-link/send   — initiate magic-link flow
 *   GET    /api/auth/callback          — Stytch redirect callback (sets session_token cookie)
 *   POST   /api/auth/logout            — revoke session, clear cookie
 *
 * The backend uses HttpOnly cookie 'session_token' for session management.
 * No CSRF token, no role cookie, no registration endpoint, no /me endpoint.
 */

import { api } from "./client";
import type {
    MagicLinkRequest,
    MagicLinkResponse,
    CallbackResponse,
    LogoutResponse,
} from "./generated";

// ─── Re-export generated types used by consumers ─────────────────────────

export type {
    MagicLinkRequest,
    MagicLinkResponse,
    CallbackResponse,
    LogoutResponse,
} from "./generated";

/** PHASE 1: Send a magic link to the given email. */
export async function sendMagicLink(email: string, orgId?: string): Promise<MagicLinkResponse> {
    const body: MagicLinkRequest = { email };
    if (orgId) body.org_id = orgId;
    return api.post<MagicLinkResponse>("/api/auth/magic-link/send", body);
}

/**
 * The callback endpoint is NOT called directly by the frontend.
 * Stytch redirects the user's browser to /api/auth/callback?token=...
 * The backend validates the token, sets the session_token cookie, and redirects.
 * This function exists only for documentation completeness.
 */
export async function handleCallback(token: string): Promise<CallbackResponse> {
    return api.get<CallbackResponse>(`/api/auth/callback?token=${encodeURIComponent(token)}`);
}

/** Logout: destroy the current session. */
export async function logout(): Promise<LogoutResponse> {
    return api.post<LogoutResponse>("/api/auth/logout");
}
