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

// ---------------------------------------------------------------------------
// Render cold-start handling constants
// ---------------------------------------------------------------------------

/** Header set by Render when the backend is hibernating and rate-limiting requests. */
const RENDER_ROUTING_HEADER = "x-render-routing";

/** Value of the above header when the backend is hibernating. */
const RENDER_HIBERNATION_VALUE = "hibernate-rate-limited";

/** Health-check endpoint on the Go backend used to determine readiness. */
const READINESS_PATH = "/readyz";

/** Maximum number of readiness-poll attempts. */
const MAX_READINESS_ATTEMPTS = 60;

/** Delay between readiness polls in milliseconds. */
const READINESS_POLL_INTERVAL_MS = 4000;

/** Overall deadline for cold-start handling (health checks + retry) in milliseconds. */
const COLD_START_TIMEOUT_MS = 80000;

/** Error code returned in the structured 503 response when the backend never becomes ready. */
const COLD_START_ERROR_CODE = "BACKEND_UNAVAILABLE";

/**
 * Returns true when the response specifically indicates Render's
 * hibernate-rate-limited condition (not an arbitrary application-level 429).
 */
function isRenderHibernationResponse(res: Response): boolean {
  return (
    res.status === 429 &&
    res.headers.get(RENDER_ROUTING_HEADER) === RENDER_HIBERNATION_VALUE
  );
}

/**
 * Promise-based delay used between readiness polls.
 */
function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

/**
 * Polls the backend /readyz endpoint until it returns 200 OK or the deadline
 * is exceeded. Resolves when the backend is ready; rejects on timeout.
 */
async function awaitBackendReady(apiUrl: string): Promise<void> {
  const deadline = Date.now() + COLD_START_TIMEOUT_MS;

  for (let attempt = 1; attempt <= MAX_READINESS_ATTEMPTS; attempt++) {
    const remaining = deadline - Date.now();
    if (remaining <= 0) {
      throw new Error(
        "Cold-start deadline exceeded waiting for backend readiness",
      );
    }

    try {
      const res = await fetch(`${apiUrl}${READINESS_PATH}`, {
        method: "GET",
        cache: "no-store",
        // Keep the health check lightweight: no body, no cookies.
      });
      if (res.ok) {
        console.log("[api-client] backend ready after cold-start", { attempt });
        return;
      }
      console.log("[api-client] backend cold-start check", {
        attempt,
        url: `${apiUrl}${READINESS_PATH}`,
        method: "GET",
        status: res?.status ?? null,
        render: res?.headers.get("x-render-routing"),
        mitigated: res?.headers.get("cf-mitigated"),
        origin: res?.headers.get("x-render-origin-server"),
      });
    } catch (err) {
      // Network errors during polling are expected while the backend is
      // waking up; log and continue polling.
      console.log("[api-client] backend not yet ready", {
        attempt,
        error: err instanceof Error ? err.message : String(err),
      });
    }

    // Do not sleep past the deadline.
    const nextRemaining = deadline - Date.now();
    if (nextRemaining > 0) {
      await sleep(Math.min(READINESS_POLL_INTERVAL_MS, nextRemaining));
    }
  }

  throw new Error("Backend did not become ready within maximum attempts");
}

/**
 * Builds a structured 503 Service Unavailable response consistent with
 * existing BFF error conventions.
 */
function coldStartUnavailableResponse(): Response {
  return new NextResponse(
    JSON.stringify({
      code: COLD_START_ERROR_CODE,
      message: "Backend is temporarily unavailable. Please try again shortly.",
    }),
    {
      status: 503,
      statusText: "Service Unavailable",
      headers: {
        "Content-Type": "application/json",
      },
    },
  );
}

/**
 * apiRequest forwards a request to the backend API, attaching the HttpOnly
 * session cookie from the browser. Set-Cookie headers from the backend
 * responses are forwarded back to the browser.
 * Origin is forwarded on mutating methods for CSRF middleware.
 *
 * Render cold-start handling:
 *   If the backend returns 429 with x-render-routing: hibernate-rate-limited,
 *   the function polls /readyz until the backend is healthy, then retries the
 *   original request. If the backend never becomes ready within the overall
 *   timeout, a structured 503 is returned.
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

  // Forward client IP for rate limiting (Cloudflare sets CF-Connecting-IP)
  const incomingHeaders = new Headers(init.headers);
  const cfConnectingIP = incomingHeaders.get("CF-Connecting-IP");
  if (cfConnectingIP) {
    headers["X-Forwarded-For"] = cfConnectingIP;
  }

  // Capture the body before the first fetch so it can be replayed if a
  // cold-start retry is needed. ArrayBuffer (used by all current callers) can
  // be passed to fetch multiple times safely.
  const requestBody = init.body;

  /**
   * Performs the actual fetch to the backend with the given body.
   */
  const fetchWithBody = (
    body: BodyInit | null | undefined,
  ): Promise<Response> =>
    fetch(`${apiUrl}${path}`, {
      ...init,
      body,
      headers,
      cache: "no-store",
    });

  const res = await fetchWithBody(requestBody);

  // -----------------------------------------------------------------------
  // Render cold-start detection and retry
  // -----------------------------------------------------------------------
  if (isRenderHibernationResponse(res)) {
    console.log("[api-client] detected Render hibernation rate-limit", {
      path,
      method,
      status: res.status,
      header: res.headers.get(RENDER_ROUTING_HEADER),
    });

    try {
      await awaitBackendReady(apiUrl);

      // Replay the original request with the same body, headers, and method.
      const retryRes = await fetchWithBody(requestBody);

      // Forward all upstream headers from the retry response.
      const retryResponseHeaders = new Headers();
      retryRes.headers.forEach((v, k) => {
        if (k.toLowerCase() !== "set-cookie") {
          retryResponseHeaders.set(k, v);
        }
      });
      for (const cookie of retryRes.headers.getSetCookie()) {
        retryResponseHeaders.append("Set-Cookie", cookie);
      }

      console.log("[api-client] cold-start retry succeeded", {
        path,
        method,
        status: retryRes.status,
      });

      return new NextResponse(await retryRes.arrayBuffer(), {
        status: retryRes.status,
        statusText: retryRes.statusText,
        headers: retryResponseHeaders,
      });
    } catch (err) {
      console.error(
        "[api-client] backend did not become ready for cold-start retry",
        {
          path,
          method,
          error: err instanceof Error ? err.message : String(err),
        },
      );

      return coldStartUnavailableResponse();
    }
  }

  // -----------------------------------------------------------------------
  // Normal (non-hibernation) response path — forward unchanged
  // -----------------------------------------------------------------------
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
