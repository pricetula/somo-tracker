import "server-only";

import { cookies, headers } from "next/headers";
import { api, type RequestOptions } from "./client";

// Get cookies that exist from the browser side and inject them to server side before making request to backend
async function withCookies(options?: RequestOptions): Promise<RequestOptions> {
    const store = await cookies();
    const hdrs = await headers();
    const all = store.getAll();
    const cookieHeader = all.map((c) => `${c.name}=${c.value}`).join("; ");
    const csrfCookie = all.find((c) => c.name === "csrf_token");
    const csrfHeader = csrfCookie ? csrfCookie.value : undefined;

    // Forward fingerprint headers from browser so session middleware doesn't
    // trigger fingerprint mismatch and delete the session from Redis.
    const fingerprintHeaders: Record<string, string> = {};
    const ua = hdrs.get("user-agent");
    if (ua) fingerprintHeaders["User-Agent"] = ua;
    const al = hdrs.get("accept-language");
    if (al) fingerprintHeaders["Accept-Language"] = al;
    const ae = hdrs.get("accept-encoding");
    if (ae) fingerprintHeaders["Accept-Encoding"] = ae;

    return {
        ...options,
        cookieHeader,
        csrfHeader,
        headers: { ...fingerprintHeaders, ...(options?.headers ?? {}) },
    };
}

export const serverApi = {
    get: async <T>(path: string, options?: RequestOptions) =>
        api.get<T>(path, await withCookies(options)),
    post: async <T>(path: string, body?: unknown, options?: RequestOptions) =>
        api.post<T>(path, body, await withCookies(options)),
    put: async <T>(path: string, body?: unknown, options?: RequestOptions) =>
        api.put<T>(path, body, await withCookies(options)),
    patch: async <T>(path: string, body?: unknown, options?: RequestOptions) =>
        api.patch<T>(path, body, await withCookies(options)),
    delete: async <T>(path: string, body?: unknown, options?: RequestOptions) =>
        api.delete<T>(path, body, await withCookies(options)),
};
