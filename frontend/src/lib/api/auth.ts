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

import { api, ApiRequestError } from "./client";

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
    first_name: string;
    last_name: string;
}

export interface MeResponse {
    user_id: string;
    tenant_id: string;
    school_id: string;
    first_name: string;
    last_name: string;
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

/** Fetch the current session's user and tenant IDs. */
export async function getMe(): Promise<MeResponse> {
    return api.get<MeResponse>("/api/auth/me");
}

/** Logout: destroy the current session. */
export async function logout(): Promise<void> {
    await api.delete("/api/auth/session");
}

// ─── Error helpers ────────────────────────────────────────────────────────

export function isApiError(err: unknown): err is ApiRequestError {
    return err instanceof ApiRequestError;
}

export function getApiErrorMessage(err: unknown): string {
    if (isApiError(err)) {
        return err.body.message ?? err.body.error ?? "An error occurred";
    }
    if (err instanceof Error) return err.message;
    return "An unexpected error occurred";
}
