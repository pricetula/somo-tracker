"use client";

import { useState, useCallback, useRef } from "react";

import { Turnstile } from "@/components/turnstile";

/**
 * Login form — email entry for OTP authentication.
 *
 * Step 1: User enters email → POST /api/auth/otp/request → backend sends OTP via email
 * Step 2: User enters 6-digit code → POST /api/auth/otp/verify → backend creates session
 * Step 3: Redirect to /
 *
 * Turnstile site key is hardcoded in the Turnstile component itself.
 * The Turnstile token is captured in state and sent to the backend alongside
 * the email. The Go backend verifies the token with Cloudflare before sending
 * any email.
 */
export default function LoginForm() {
  const [step, setStep] = useState<"email" | "otp">("email");
  const [email, setEmail] = useState("");
  const [code, setCode] = useState("");
  const [challengeId, setChallengeId] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [turnstileToken, setTurnstileToken] = useState<string | null>(null);
  const turnstileRef = useRef<React.ComponentRef<typeof Turnstile>>(null);

  const requestOTP = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      setError(null);
      setLoading(true);

      try {
        const res = await fetch("/api/auth/otp/request", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          credentials: "include",
          body: JSON.stringify({ email, turnstile_token: turnstileToken }),
        });

        if (!res.ok) {
          const body = await res.json().catch(() => ({}));
          if (res.status === 403) {
            setError(
              "Verification failed. Please try again.",
            );
          } else if (res.status === 502) {
            setError(
              "Service temporarily unavailable. Please try again later.",
            );
          } else {
            setError(
              body.message || "Failed to send code. Please try again.",
            );
          }
          setLoading(false);
          // Reset widget and clear token after every submit attempt.
          turnstileRef.current?.reset();
          setTurnstileToken(null);
          return;
        }

        const data = await res.json();
        setChallengeId(data.challenge_id);
        setStep("otp");
      } catch {
        setError("Failed to send code. Please try again.");
        setLoading(false);
      } finally {
        setLoading(false);
        // Reset widget and clear token after every submit attempt,
        // whether it succeeded or failed.
        turnstileRef.current?.reset();
        setTurnstileToken(null);
      }
    },
    [email, turnstileToken],
  );

  const onTurnstileVerify = useCallback((token: string) => {
    setTurnstileToken(token);
    setError(null);
  }, []);

  const onTurnstileExpire = useCallback(() => {
    setTurnstileToken(null);
  }, []);

  const onTurnstileError = useCallback(() => {
    setTurnstileToken(null);
  }, []);

  const verifyOTP = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      setError(null);
      setLoading(true);

      try {
        const res = await fetch("/api/auth/otp/verify", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          credentials: "include",
          body: JSON.stringify({ challenge_id: challengeId, code }),
        });

        if (!res.ok) {
          const body = await res.json().catch(() => ({}));
          setError(body.message || "Invalid code. Please try again.");
          setLoading(false);
          return;
        }

        const data = await res.json();
        const nextPath = data.needs_onboarding ? "/onboarding" : "/";
        window.location.assign(nextPath);
      } catch {
        setError("Failed to verify code. Please try again.");
        setLoading(false);
      } finally {
        setLoading(false);
      }
    },
    [code],
  );

  return (
    <div className="flex min-h-screen flex-col items-center justify-center p-4">
      <div className="w-full max-w-md space-y-6">
        {step === "email" && (
          <>
            <h1 className="text-3xl font-bold">Sign in</h1>
            <p className="text-muted-foreground">
              Enter your email to receive a verification code.
            </p>
            <form onSubmit={requestOTP} className="space-y-4">
              <div className="space-y-2">
                <label htmlFor="email" className="block text-sm font-medium">
                  Email address
                </label>
                <input
                  id="email"
                  type="email"
                  placeholder="you@example.com"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  required
                  disabled={loading}
                  className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:opacity-50"
                />
                <Turnstile
                  ref={turnstileRef}
                  onVerify={onTurnstileVerify}
                  onExpire={onTurnstileExpire}
                  onError={onTurnstileError}
                />
              </div>

              {error && <p className="text-sm text-destructive">{error}</p>}
              <button
                type="submit"
                disabled={loading || !email || !turnstileToken}
                className="w-full inline-flex items-center justify-center rounded-md border border-transparent bg-primary px-4 py-2 text-sm font-medium text-primary-foreground shadow-sm hover:bg-primary/90 disabled:opacity-50"
              >
                {loading ? "Sending…" : "Send code"}
              </button>
            </form>
          </>
        )}
        {step === "otp" && (
          <>
            <h1 className="text-3xl font-bold">Verify code</h1>
            <p className="text-muted-foreground">
              Enter the 6-digit code sent to{" "}
              <strong className="font-medium">{email}</strong>.
            </p>
            <form onSubmit={verifyOTP} className="space-y-4">
              <div className="space-y-2">
                <label htmlFor="code" className="block text-sm font-medium">
                  Verification code
                </label>
                <input
                  id="code"
                  type="text"
                  maxLength={6}
                  value={code}
                  onChange={(e) => setCode(e.target.value.replace(/\D/g, ""))}
                  required
                  disabled={loading}
                  className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:opacity-50 letter-spacing-widest"
                />
              </div>
              {error && <p className="text-sm text-destructive">{error}</p>}
              <button
                type="submit"
                disabled={loading || code.length !== 6}
                className="w-full inline-flex items-center justify-center rounded-md border border-transparent bg-primary px-4 py-2 text-sm font-medium text-primary-foreground shadow-sm hover:bg-primary/90 disabled:opacity-50"
              >
                {loading ? "Verifying…" : "Verify"}
              </button>
            </form>
          </>
        )}
      </div>
    </div>
  );
}
