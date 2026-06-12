/**
 * Base API client for communicating with the Go backend.
 *
 * All requests are sent with `credentials: "include"` so the HttpOnly
 * `somo_sid` cookie is attached automatically by the browser.
 */

const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:3030";

export interface ApiError {
  error: string;
  message?: string;
}

export class ApiRequestError extends Error {
  public status: number;
  public body: ApiError;

  constructor(status: number, body: ApiError) {
    super(body.message ?? body.error);
    this.name = "ApiRequestError";
    this.status = status;
    this.body = body;
  }
}

async function request<T>(
  method: string,
  path: string,
  body?: unknown,
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
    let apiErr: ApiError;
    try {
      apiErr = (await res.json()) as ApiError;
    } catch {
      apiErr = { error: "unknown", message: res.statusText };
    }
    throw new ApiRequestError(res.status, apiErr);
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

export const api = {
  get: <T>(path: string) => request<T>("GET", path),
  post: <T>(path: string, body?: unknown) => request<T>("POST", path, body),
  put: <T>(path: string, body?: unknown) => request<T>("PUT", path, body),
  patch: <T>(path: string, body?: unknown) => request<T>("PATCH", path, body),
  delete: <T>(path: string, body?: unknown) =>
    request<T>("DELETE", path, body),
};
