import { apiRequest } from '@/lib/api-client'

export const dynamic = 'force-dynamic'

interface RequestOTPBody {
  email: string;
  turnstile_token?: string;
}

/**
 * Verifies a Cloudflare Turnstile token before forwarding the OTP request to
 * the backend. Verification happens here (on Vercel) so the
 * `TURNSTILE_SECRET_KEY` never needs to leave this deployment.
 *
 * Enforced only when `TURNSTILE_SECRET_KEY` is set; when unset (local dev)
 * the request is forwarded as-is so development isn't blocked.
 */
async function verifyTurnstile(token: string | undefined): Promise<void> {
  const secret = process.env.TURNSTILE_SECRET_KEY;
  if (!secret) {
    // Turnstile not configured — skip verification (e.g. local development).
    return;
  }

  if (!token) {
    throw new Error("turnstile token required");
  }

  const res = await fetch(
    "https://challenges.cloudflare.com/turnstile/v0/siteverify",
    {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      body: `secret=${encodeURIComponent(secret)}&response=${encodeURIComponent(token)}`,
      cache: "no-store",
    },
  );

  const result = await res.json().catch(() => ({}));
  if (!result.success) {
    throw new Error("turnstile verification failed");
  }
}

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

  try {
    await verifyTurnstile(body.turnstile_token);
  } catch (err) {
    return new Response(
      JSON.stringify({ message: err instanceof Error ? err.message : "turnstile verification failed" }),
      { status: 400, headers: { "Content-Type": "application/json" } },
    );
  }

  return apiRequest("/api/v1/auth/otp/request", {
    method: "POST",
    body: JSON.stringify({ email: body.email }),
    headers: {
      "Content-Type": "application/json",
    },
  });
}
