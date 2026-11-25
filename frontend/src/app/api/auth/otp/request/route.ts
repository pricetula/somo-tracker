import { apiRequest } from '@/lib/api-client'

export const dynamic = 'force-dynamic'

interface RequestOTPBody {
  email: string;
  turnstile_token: string;
}

/**
 * Forwards the OTP request (email + Turnstile token) to the Go backend.
 * Turnstile verification is performed by the Go backend; this route simply
 * relays the token so it never needs to be handled here.
 */
export async function POST(req: Request) {
  let body: RequestOTPBody;
  try {
    body = (await req.json()) as RequestOTPBody;
  } catch {
    return new Response(
      JSON.stringify({ message: "invalid request body" }),
      { status: 400, headers: { "Content-Type": "application/json" } },
    );
  }

  if (!body.email || !body.turnstile_token) {
    return new Response(
      JSON.stringify({ message: "email and turnstile token are required" }),
      { status: 400, headers: { "Content-Type": "application/json" } },
    );
  }

  return apiRequest("/api/v1/auth/otp/request", {
    method: "POST",
    body: JSON.stringify({ email: body.email, turnstile_token: body.turnstile_token }),
    headers: {
      "Content-Type": "application/json",
    },
  });
}
