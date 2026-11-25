import type {
  RequestOTPInput,
  RequestOTPResponse,
  VerifyOTPInput,
  VerifyOTPResponse,
} from "../types";

/**
 * requestOTP forwards the OTP request (email + Turnstile token) to the
 * Go backend via the BFF proxy route /api/auth/otp/request.
 * Uses client-side fetch with credentials: include so the browser sends
 * cookies (session, needs_onboarding) to the BFF route handler.
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

  return res.json();
}

/**
 * verifyOTP forwards the OTP verification (challenge_id + code) to the
 * Go backend via the BFF proxy route /api/auth/otp/verify.
 * Uses client-side fetch with credentials: include.
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

  return res.json();
}
