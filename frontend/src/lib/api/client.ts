/**
 * Base API client for communicating with the Go backend.
 *
 * Canonical Error Response Contract (from internal/middleware/errors.go):
 *
 * Every non-2xx HTTP response from the backend MUST return this exact JSON body:
 *
 *   {
 *     "code":       "snake_case_error_code",
 *     "message":    "human readable message",
 *     "errors":     { "field_name": ["Specific field validation message"] },
 *     "request_id": "uuid — correlation id echoed from X-Request-ID"
 *   }
 *
 * code is always a snake_case string the frontend can switch on.
 * message is a safe, human-readable string.
 * errors is an optional object populated exclusively on 400 Bad Request /
 * validation failures, mapping field keys to an array of specific error messages.
 * request_id is an optional correlation id (backend middleware/requestid.go)
 * appended to every error body; it is also echoed in the X-Request-ID response
 * header. Rate-limit responses additionally carry retry_after_seconds.
 *
 * All requests carry a per-page-load correlation id in the X-Request-ID header
 * (honored + echoed by the backend) and are sent with `credentials: "include"`
 * so the HttpOnly `somo_sid` cookie is attached automatically by the browser.
 *
 * Backend counterpart: internal/middleware/errors.go
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
    /** Backend correlation id (X-Request-ID echoed in the error body). */
    public requestId?: string;
    /** Optional extra fields carried in the error body (e.g. active_job_id). */
    public extra?: Record<string, unknown>;

    constructor(
        status: number,
        code: string,
        message: string,
        errors?: Record<string, string[]>,
        requestId?: string,
        extra?: Record<string, unknown>
    ) {
        super(message);
        this.name = "ApiError";
        this.status = status;
        this.code = code;
        this.errors = errors;
        this.requestId = requestId;
        this.extra = extra;
    }
}

// ─── Request options ──────────────────────────────────────────────────────

export interface RequestOptions {
    /** If true, skip the global 401 redirect to /logout. Use for endpoints
     *  where a 401 is structurally expected (e.g. initial me check). */
    skipGlobal401Handler?: boolean;
    /** If true, skip the global 403 redirect to /unauthorized (only applies
     *  to GET /api/auth/me — a session rejected by the backend's resolver). */
    skipGlobal403Handler?: boolean;
}

// ─── Correlation id ────────────────────────────────────────────────────────

/**
 * Per-page-load correlation id sent as X-Request-ID on every request.
 *
 * The backend (middleware/requestid.go) honors a well-formed incoming
 * X-Request-ID, echoes it in the response header, and threads it into every
 * error body — so a single support ticket maps to one id across the whole
 * stack. Keeping the id stable for the lifetime of the page makes all
 * requests issued from one view correlate to the same trace.
 */
let correlationId: string | null = null;

function getCorrelationId(): string {
    if (!correlationId) {
        correlationId =
            typeof crypto !== "undefined" && "randomUUID" in crypto
                ? crypto.randomUUID()
                : `req-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 12)}`;
    }
    return correlationId;
}

// ─── Base fetch wrapper ───────────────────────────────────────────────────

async function request<T>(
    method: string,
    path: string,
    body?: unknown,
    options?: RequestOptions
): Promise<T> {
    const url = `${API_BASE}${path}`;

    const headers: Record<string, string> = {
        "X-Request-ID": getCorrelationId(),
    };
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
        let apiErr: {
            code?: string;
            message?: string;
            errors?: Record<string, string[]>;
            request_id?: string;
        };
        try {
            apiErr = (await res.json()) as typeof apiErr;
        } catch {
            apiErr = { code: "unknown", message: res.statusText };
        }

        // Collect any extra fields from the error body beyond the standard ones
        // (e.g. active_job_id, retry_after_seconds) using delete to avoid
        // unused variable warnings.
        const extra: Record<string, unknown> = { ...apiErr };
        delete extra.code;
        delete extra.message;
        delete extra.errors;
        delete extra.request_id;

        const error = new ApiError(
            res.status,
            apiErr.code ?? "unknown",
            apiErr.message ?? "Unexpected error",
            apiErr.errors,
            apiErr.request_id,
            Object.keys(extra).length > 0 ? extra : undefined
        );

        // ─── Global 401 Eviction ─────────────────────────────────────────
        // If any API request returns 401 Unauthorized, force a redirect to
        // /logout to clear HTTP session cookies, invalidate local state, and
        // wipe the React Query cache.
        if (res.status === 401 && !options?.skipGlobal401Handler) {
            window.location.href = "/logout";
        }

        // ─── Global 403 — session without any membership (B3) ───────────
        // The session resolver rejects a VALID session whose user has zero
        // active memberships with 403 forbidden on every request. GET /me is
        // the session probe, so a 403 there unambiguously means "authenticated
        // but entitled to nothing" — show /unauthorized (offers sign-out +
        // contact-admin guidance) instead of silently treating the user as
        // logged out.
        if (
            res.status === 403 &&
            path === "/api/auth/me" &&
            error.code === "forbidden" &&
            !options?.skipGlobal403Handler
        ) {
            window.location.href = "/unauthorized";
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
