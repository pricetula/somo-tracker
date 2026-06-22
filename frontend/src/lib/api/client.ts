/**
 * Base API client for communicating with the Go backend.
 *
 * Canonical Error Response Contract (from internal/middleware/errors.go):
 *
 * Every non-2xx HTTP response from the backend MUST return this exact JSON body:
 *
 *   {
 *     "code":    "snake_case_error_code",
 *     "message": "human readable message",
 *     "errors":  { "field_name": ["Specific field validation message"] }
 *   }
 *
 * code is always a snake_case string the frontend can switch on.
 * message is a safe, human-readable string.
 * errors is an optional object populated exclusively on 400 Bad Request /
 * validation failures, mapping field keys to an array of specific error messages.
 *
 * Backend counterpart: internal/middleware/errors.go
 *
 * All requests are sent with `credentials: "include"` so the HttpOnly
 * `somo_sid` cookie is attached automatically by the browser.
 */

const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "";

// ─── ApiError ──────────────────────────────────────────────────────────────

/**
 * Structured error thrown for every non-2xx API response.
 * The `code` field matches the backend's snake_case error code.
 * The `errors` field carries field-level validation failures (400 responses).
 */
export class ApiError extends Error {
    public status: number;
    public code: string;
    public errors?: Record<string, string[]>;

    constructor(status: number, code: string, message: string, errors?: Record<string, string[]>) {
        super(message);
        this.name = "ApiError";
        this.status = status;
        this.code = code;
        this.errors = errors;
    }
}

// ─── Request options ──────────────────────────────────────────────────────

export interface RequestOptions {
    /** If true, skip the global 401 redirect to /logout. Use for endpoints
     *  where a 401 is structurally expected (e.g. initial me check). */
    skipGlobal401Handler?: boolean;
}

// ─── Base fetch wrapper ───────────────────────────────────────────────────

async function request<T>(
    method: string,
    path: string,
    body?: unknown,
    options?: RequestOptions
): Promise<T> {
    const url = `${API_BASE}${path}`;

    const headers: Record<string, string> = {};
    if (body !== undefined) {
        headers["Content-Type"] = "application/json";
    }

    // Include CSRF token on mutating requests (double-submit cookie pattern)
    if (["POST", "PUT", "PATCH", "DELETE"].includes(method)) {
        const csrf = getCSRFToken();
        if (csrf) {
            headers["X-CSRF-Token"] = csrf;
        }
    }

    const res = await fetch(url, {
        method,
        headers,
        credentials: "include",
        body: body !== undefined ? JSON.stringify(body) : undefined,
    });

    if (!res.ok) {
        let apiErr: { code?: string; message?: string; errors?: Record<string, string[]> };
        try {
            apiErr = (await res.json()) as typeof apiErr;
        } catch {
            apiErr = { code: "unknown", message: res.statusText };
        }

        const error = new ApiError(
            res.status,
            apiErr.code ?? "unknown",
            apiErr.message ?? "Unexpected error",
            apiErr.errors
        );

        // ─── Global 401 Eviction ─────────────────────────────────────────
        // If any API request returns 401 Unauthorized, force a redirect to
        // /logout to clear HTTP session cookies, invalidate local state, and
        // wipe the React Query cache.
        if (res.status === 401 && !options?.skipGlobal401Handler) {
            window.location.href = "/logout";
        }

        throw error;
    }

    // 204 No Content
    if (res.status === 204) {
        return undefined as T;
    }

    // Some endpoints return just a status code
    const contentType = res.headers.get("content-type") ?? "";
    if (contentType.includes("application/json")) {
        return (await res.json()) as T;
    }

    return undefined as T;
}

/** Read the CSRF token from the non-HttpOnly cookie set by the backend. */
function getCSRFToken(): string | null {
    if (typeof document === "undefined") return null;
    const match = document.cookie.match(/(?:^|;\s*)csrf_token=([^;]*)/);
    return match ? decodeURIComponent(match[1]) : null;
}

// ─── Public API surface ───────────────────────────────────────────────────

export const api = {
    get: <T>(path: string, options?: RequestOptions) => request<T>("GET", path, undefined, options),
    post: <T>(path: string, body?: unknown, options?: RequestOptions) =>
        request<T>("POST", path, body, options),
    put: <T>(path: string, body?: unknown, options?: RequestOptions) =>
        request<T>("PUT", path, body, options),
    patch: <T>(path: string, body?: unknown, options?: RequestOptions) =>
        request<T>("PATCH", path, body, options),
    delete: <T>(path: string, body?: unknown, options?: RequestOptions) =>
        request<T>("DELETE", path, body, options),
};
