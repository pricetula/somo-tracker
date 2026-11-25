import type {
  RequestOTPInput,
  RequestOTPResponse,
  VerifyOTPInput,
  VerifyOTPResponse,
} from "../types";
import type {
  RequestOTPResponseBody,
  VerifyOTPResponseBody,
} from "../types/generated";

/**
 * requestOTP forwards the OTP request (email + Turnstile token) to the
 * Go backend via the BFF proxy route /api/auth/otp/request.
 * Uses client-side fetch with credentials: include so the browser sends
 * cookies (session, needs_onboarding) to the BFF route handler.
 *
 * The raw backend response uses snake_case keys (per the generated OpenAPI
 * contract). We map to camelCase for the frontend-facing type.
 */
export async function requestOTP(input: RequestOTPInput): Promise<RequestOTPResponse> {
  const res = await fetch("/api/auth/otp/request", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: JSON.stringify({
      email: input.email,
      turnstile_token: input.turnstileToken,
    }),
  });

  if (!res.ok) {
    const body = (await res.json().catch(() => ({}))) as { message?: string };
    const status = res.status;

    if (status === 403) {
      throw new Error("Verification failed. Please try again.");
    }
    if (status === 502) {
      throw new Error("Service temporarily unavailable. Please try again later.");
    }
    throw new Error(body.message || "Failed to send code. Please try again.");
  }

  const raw = (await res.json()) as RequestOTPResponseBody;
  return {
    ok: raw.ok ?? false,
    message: raw.message ?? "",
    challengeId: raw.challenge_id ?? "",
  };
}

/**
 * verifyOTP forwards the OTP verification (challenge_id + code) to the
 * Go backend via the BFF proxy route /api/auth/otp/verify.
 * Uses client-side fetch with credentials: include.
 *
 * The raw backend response uses snake_case keys (per the generated OpenAPI
 * contract). We map to camelCase for the frontend-facing type.
 */
export async function verifyOTP(input: VerifyOTPInput): Promise<VerifyOTPResponse> {
  const res = await fetch("/api/auth/otp/verify", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: JSON.stringify({
      challenge_id: input.challengeId,
      code: input.code,
    }),
  });

  if (!res.ok) {
    const body = (await res.json().catch(() => ({}))) as { message?: string };
    throw new Error(body.message || "Invalid code. Please try again.");
  }

  const raw = (await res.json()) as VerifyOTPResponseBody;
  return {
    ok: raw.ok ?? false,
    message: raw.message ?? "",
    needsOnboarding: raw.needs_onboarding ?? false,
  };
}
