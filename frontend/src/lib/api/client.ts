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
 * so the HttpOnly `session_token` cookie is attached automatically by the browser.
 *
 * Backend counterpart: internal/middleware/errors.go
 */

// ─── Environment detection ────────────────────────────────────────────────

/**
 * Determines if code is running in a browser context.
 * Works in RSC, Server Actions, Route Handlers, Middleware, and Client Components.
 */
function isBrowser(): boolean {
    return typeof window !== "undefined";
}

/**
 * Gets the API base URL for the current execution context.
 *
 * - Server (RSC, Server Actions, Route Handlers, Middleware): Uses `API_URL` env var
 *   (e.g., `http://somotracker_api:3030` in Docker, `https://api.example.com` in prod).
 *   This bypasses the Next.js proxy for direct backend access.
 * - Client (Browser): Uses the Next.js rewrite proxy prefix (e.g., `/backend`).
 *   The proxy forwards to the backend via the same-origin rewrite.
 *
 * This function is evaluated at **call time**, not module load time, so it works
 * correctly regardless of which bundle (server/client) the code runs in.
 */
function getApiBase(): string {
    if (isBrowser()) {
        // Client-side: use the public proxy prefix configured in next.config.ts
        // Falls back to "/backend" if not set (matches default in next.config.ts).
        return process.env.NEXT_PUBLIC_API_PROXY_PREFIX ?? "/backend";
    }
    // Server-side: use the direct backend URL from server-only env var.
    // Falls back to localhost for local development outside Docker.
    return process.env.API_URL ?? "http://somotracker_api:3030";
}

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

function getCsrfToken(): string | null {
    if (!isBrowser()) return null;
    const match = document.cookie.match(/(?:^|;)\s*csrf_token=([^;]+)/);
    return match ? decodeURIComponent(match[1]) : null;
}

export interface RequestOptions {
    skipGlobal401Handler?: boolean;
    headers?: Record<string, string>;
    /** Server-only: raw Cookie header to forward (e.g. from next/headers or req.headers.cookie). */
    cookieHeader?: string;
    /** Server-only: CSRF token value to send as X-CSRF-Token header. */
    csrfHeader?: string;
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

/**
 * Resets the correlation id. Useful for testing or when navigating to a new
 * logical "page" in a SPA context without full reload.
 */
export function resetCorrelationId(): void {
    correlationId = null;
}

// ─── Base fetch wrapper ───────────────────────────────────────────────────

async function request<T>(
    method: string,
    path: string,
    body?: unknown,
    options?: RequestOptions
): Promise<T> {
    // Resolve base URL at call time to handle both server and client contexts correctly.
    const baseUrl = getApiBase();
    const url = `${baseUrl}${path}`;

    const headers: Record<string, string> = {
        "X-Request-ID": getCorrelationId(),
        ...(options?.headers ?? {}),
    };

    if (body !== undefined) {
        headers["Content-Type"] = "application/json";
    }

    // Server-only: forward an explicit Cookie header if the caller supplied one
    // (e.g. via next/headers on App Router, or req.headers.cookie on Pages
    // Router). Never applied in the browser — the browser already attaches
    // cookies itself via credentials: "include".
    if (!isBrowser() && options?.cookieHeader) {
        headers["Cookie"] = options.cookieHeader;
    }

    // Client-side: include CSRF token for mutating requests per backend csrf middleware.
    if (isBrowser()) {
        const csrf = getCsrfToken();
        if (csrf) headers["X-CSRF-Token"] = csrf;
    }

    // Server-only: include CSRF token header if caller provided it (e.g. via serverApi).
    if (!isBrowser() && options?.csrfHeader) {
        headers["X-CSRF-Token"] = options.csrfHeader;
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
        if (res.status === 401 && !options?.skipGlobal401Handler && isBrowser()) {
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
