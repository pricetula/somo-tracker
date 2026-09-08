import "server-only";

import { cookies } from "next/headers";
import { api, type RequestOptions } from "./client";

// Get cookies that exist from the browser side and inject them to server side before making request to backend
async function withCookies(options?: RequestOptions): Promise<RequestOptions> {
    const store = await cookies();
    const all = store.getAll();
    const cookieHeader = all.map((c) => `${c.name}=${c.value}`).join("; ");
    return { ...options, cookieHeader };
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
