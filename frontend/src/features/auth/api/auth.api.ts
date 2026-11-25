import { apiRequest } from "@/lib/api-client";
import type {
  RequestOTPInput,
  RequestOTPResponse,
  VerifyOTPInput,
  VerifyOTPResponse,
} from "../types";

/**
 * RequestOTPAPI forwards the OTP request (email + Turnstile token) to the
 * Go backend via the BFF proxy route /api/auth/otp/request.
 */
export async function requestOTP(input: RequestOTPInput): Promise<RequestOTPResponse> {
  const res = await apiRequest("/api/auth/otp/request", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
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
 * VerifyOTPAPI forwards the OTP verification (challenge_id + code) to the
 * Go backend via the BFF proxy route /api/auth/otp/verify.
 */
export async function verifyOTP(input: VerifyOTPInput): Promise<VerifyOTPResponse> {
  const res = await apiRequest("/api/auth/otp/verify", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
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
