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

/** Total deadline for waiting for backend readiness after a hibernate 429. */
const COLD_START_READINESS_DEADLINE_MS = 60_000;

/** Initial delay (ms) between readiness probes; doubles with each attempt. */
const COLD_START_PROBE_INITIAL_DELAY_MS = 1_000;

/** Maximum delay (ms) between readiness probes. */
const COLD_START_PROBE_MAX_DELAY_MS = 8_000;

/** User-Agent sent on readiness probes to avoid Cloudflare bot challenges on UA-less requests. */
const READINESS_PROBE_USER_AGENT = "somo-tracker-frontend/1.0";

/** Maximum number of times to retry the original request after a hibernate 429. */
const MAX_COLD_START_RETRIES = 3;

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
 * Promise-based delay used between cold-start retries.
 */
function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

/**
 * Resolves the base URL to use for readiness probes.
 *
 * Prefers a direct Render origin (RENDER_DIRECT_URL) when configured so the
 * probe bypasses any Cloudflare edge that might challenge UA-less or
 * header-less subrequests. Falls back to PUBLIC_API_URL otherwise.
 */
function readinessProbeUrl(apiUrl: string): string {
  const direct = process.env.RENDER_DIRECT_URL;
  if (direct) {
    return direct.replace(/\/$/, "");
  }
  return apiUrl;
}

/**
 * Polls /readyz until the backend returns 200 OK or the deadline is exceeded.
 *
 * Uses exponential backoff (1s, 2s, 4s, 8s…) between probes. A 429
 * hibernate-rate-limited response means the backend is still waking up — we
 * keep polling rather than treating it as failure. Network errors during the
 * wake window are likewise expected and retried.
 */
async function awaitBackendReady(apiUrl: string): Promise<void> {
  const url = `${readinessProbeUrl(apiUrl)}${READINESS_PATH}`;
  const deadline = Date.now() + COLD_START_READINESS_DEADLINE_MS;
  let delay = COLD_START_PROBE_INITIAL_DELAY_MS;

  for (let attempt = 1; ; attempt++) {
    const remaining = deadline - Date.now();
    if (remaining <= 0) {
      throw new Error("Cold-start deadline exceeded waiting for backend readiness");
    }

    try {
      const res = await fetch(url, {
        method: "GET",
        cache: "no-store",
        signal: AbortSignal.timeout(Math.min(remaining, 30_000)),
        headers: {
          "User-Agent": READINESS_PROBE_USER_AGENT,
          Accept: "*/*",
          "Cache-Control": "no-store",
        },
      });

      if (res.ok) {
        console.log("[api-client] backend ready after cold-start", { attempt, url });
        return;
      }

      // 429 hibernate-rate-limited means the backend is still waking up — keep polling.
      if (isRenderHibernationResponse(res)) {
        console.log("[api-client] backend cold-start check", {
          attempt,
          url,
          status: res.status,
          render: res.headers.get("x-render-routing"),
        });
      } else {
        console.log("[api-client] backend cold-start check", {
          attempt,
          url,
          status: res.status,
        });
      }
    } catch (err) {
      // Network errors during polling are expected while the backend is waking up.
      console.log("[api-client] backend not yet ready", {
        attempt,
        url,
        error: err instanceof Error ? err.message : String(err),
      });
    }

    const nextRemaining = deadline - Date.now();
    if (nextRemaining <= 0) {
      throw new Error("Cold-start deadline exceeded waiting for backend readiness");
    }

    await sleep(Math.min(delay, nextRemaining));
    delay = Math.min(delay * 2, COLD_START_PROBE_MAX_DELAY_MS);
  }
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
 *   the function polls /readyz with exponential backoff until the backend
 *   returns 200 OK (up to COLD_START_READINESS_DEADLINE_MS), then retries the
 *   original request. If the retry is also rate-limited, it polls and retries
 *   up to MAX_COLD_START_RETRIES times. A structured 503 is returned only
 *   after all retries are exhausted.
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
      let lastRes = res;

      for (let retry = 1; retry <= MAX_COLD_START_RETRIES; retry++) {
        // Wait for the backend to finish booting before replaying the request.
        await awaitBackendReady(apiUrl);

        // Replay the original request with the same body, headers, and method.
        const retryRes = await fetchWithBody(requestBody);

        // If the retry is not a hibernate response, we're done.
        if (!isRenderHibernationResponse(retryRes)) {
          lastRes = retryRes;
          break;
        }

        // Still hibernating — log and retry.
        console.log("[api-client] cold-start retry still rate-limited", {
          path,
          method,
          retry,
          status: retryRes.status,
          header: retryRes.headers.get(RENDER_ROUTING_HEADER),
        });
        lastRes = retryRes;
      }

      // If the last response was still a hibernate 429, give up with 503.
      if (isRenderHibernationResponse(lastRes)) {
        console.error(
          "[api-client] backend did not become ready for cold-start retry",
          { path, method, retries: MAX_COLD_START_RETRIES },
        );
        return coldStartUnavailableResponse();
      }

      // Forward all upstream headers from the successful retry response.
      const retryResponseHeaders = new Headers();
      lastRes.headers.forEach((v, k) => {
        if (k.toLowerCase() !== "set-cookie") {
          retryResponseHeaders.set(k, v);
        }
      });
      for (const cookie of lastRes.headers.getSetCookie()) {
        retryResponseHeaders.append("Set-Cookie", cookie);
      }

      console.log("[api-client] cold-start retry succeeded", {
        path,
        method,
        status: lastRes.status,
      });

      return new NextResponse(await lastRes.arrayBuffer(), {
        status: lastRes.status,
        statusText: lastRes.statusText,
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
