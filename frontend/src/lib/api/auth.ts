/**
 * Auth API functions — all calls to the Go backend auth endpoints.
 *
 * Endpoints:
 *   POST /api/auth/discover   — initiate magic-link flow
 *   POST /api/auth/verify     — verify magic-link token, return session_ref
 *   POST /api/auth/register   — complete registration, set session cookie
 *   GET  /api/auth/me         — fetch current session info
 *   DELETE /api/auth/session  — logout
 */

import { api, ApiError } from "./client";
import { isApiError, getErrorMessage } from "../errors";

// ─── Types ────────────────────────────────────────────────────────────────

export interface DiscoverPayload {
    email: string;
}

export interface VerifyResponse {
    session_ref: string;
}

export interface RegisterPayload {
    school_name: string;
    session_ref: string;
    full_name: string;
}

export interface MeResponse {
    user_id: string;
    tenant_id: string;
    school_id: string;
    full_name: string;
    email: string;
    role: string;
}

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
export async function register(payload: RegisterPayload): Promise<void> {
    await api.post("/api/auth/register", payload);
}

/** Fetch the current session's user and tenant IDs.
 *
 * If the request fails for any reason (network error, 401, 500, etc.),
 * the user is redirected to /logout. A failing /me means the session
 * is invalid or unreachable, so we treat the user as logged out.
 */
export async function getMe(): Promise<MeResponse> {
    try {
        return await api.get<MeResponse>("/api/auth/me");
    } catch (err) {
        // Any error means the session is invalid or unreachable —
        // redirect to /logout to clear state and force re-auth.
        window.location.href = "/logout";
        throw err;
    }
}

/** Logout: destroy the current session. */
export async function logout(): Promise<void> {
    await api.delete("/api/auth/session");
}

// ─── Re-exported error helpers for backward compatibility ─────────────────

export { ApiError, isApiError, getErrorMessage };
