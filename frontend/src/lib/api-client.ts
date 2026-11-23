import { cookies } from "next/headers";
import { NextResponse } from "next/server";

function sessionCookieName(): string {
  const name = process.env.SESSION_COOKIE_NAME;
  if (!name) {
    throw new Error("SESSION_COOKIE_NAME is not configured");
  }
  return name;
}

function onboardingCookieName(): string {
  const sessionName = sessionCookieName();
  if (sessionName.startsWith("__Host-")) {
    return "__Host-needs_onboarding";
  }
  return "needs_onboarding";
}

/**
 * apiRequest forwards a request to the backend API, attaching the HttpOnly
 * session cookie from the browser. Set-Cookie headers from the backend
 * responses are forwarded back to the browser.
 * Origin is forwarded on mutating methods for CSRF middleware.
 */
export async function apiRequest(
  path: string,
  init: RequestInit = {},
): Promise<Response> {
  const apiUrl = process.env.PUBLIC_API_URL;
  if (!apiUrl) {
    throw new Error("PUBLIC_API_URL is not configured");
  }

  const cookieStore = await cookies();
  const sessionToken = cookieStore.get(sessionCookieName())?.value;

  // Forward cookies as proper "name=value" pairs so the backend can parse them.
  const cookieParts: string[] = [];
  if (sessionToken) {
    cookieParts.push(`${sessionCookieName()}=${sessionToken}`);
  }

  const headers: Record<string, string> = {};
  if (init.headers) {
    if (init.headers instanceof Headers) {
      init.headers.forEach((v, k) => {
        headers[k] = v;
      });
    } else if (Array.isArray(init.headers)) {
      for (const [k, v] of init.headers) {
        headers[k] = v;
      }
    } else {
      Object.assign(headers, init.headers);
    }
  }

  if (cookieParts.length) {
    headers["Cookie"] =
      `${headers["Cookie"] ?? ""} ${cookieParts.join("; ")}`.trim();
  }

  const method = init.method ?? "GET";
  if (["POST", "PUT", "PATCH", "DELETE"].includes(method)) {
    headers["Origin"] = process.env.PUBLIC_APP_URL ?? "";
  }

  // Forward client IP for rate limiting
  const res = await fetch(`${apiUrl}${path}`, {
    ...init,
    headers,
    cache: "no-store",
  });

  // Forward all upstream headers except Set-Cookie (handled below).
  const responseHeaders = new Headers();
  res.headers.forEach((v, k) => {
    if (k.toLowerCase() !== "set-cookie") {
      responseHeaders.set(k, v);
    }
  });
  // Append all Set-Cookie headers individually
  for (const cookie of res.headers.getSetCookie()) {
    responseHeaders.append("Set-Cookie", cookie);
  }

  return new NextResponse(await res.arrayBuffer(), {
    status: res.status,
    statusText: res.statusText,
    headers: responseHeaders,
  });
}
