"use client";

import { useState, useCallback, useEffect, useRef } from "react";

declare global {
  interface Window {
    turnstile?: {
      render: (
        element: HTMLElement,
        opts: {
          sitekey: string;
          callback: (token: string) => void;
          "expired-callback"?: () => void;
        },
      ) => number;
      remove: (widgetId: number) => void;
      getResponse: (widgetId?: number) => string | null;
    };
  }
}

/**
 * Login page — email entry for OTP authentication.
 *
 * Step 1: User completes Turnstile + enters email → POST /api/auth/otp/request → backend sends OTP via email
 * Step 2: User enters 6-digit code → POST /api/auth/otp/verify → backend creates session
 * Step 3: Redirect to /
 *
 * Turnstile (Cloudflare) is rendered on the email step. The site key must be
 * exposed to the browser as NEXT_PUBLIC_TURNSTILE_SITE_KEY. The secret key
 * never reaches the browser — it is verified server-side by the Go backend.
 */
export default function LoginPage() {
  const [step, setStep] = useState<"email" | "otp">("email");
  const [email, setEmail] = useState("");
  const [code, setCode] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [turnstileToken, setTurnstileToken] = useState("");
  const [challengeId, setChallengeId] = useState("");

  const siteKey = process.env.NEXT_PUBLIC_TURNSTILE_SITE_KEY;

  const turnstileRef = useRef<HTMLDivElement>(null);
  const widgetIdRef = useRef<number | null>(null);

  function removeTurnstileWidget() {
    if (widgetIdRef.current == null) return;
    const turnstile = window.turnstile;
    if (turnstile) {
      turnstile.remove(widgetIdRef.current);
    }
    widgetIdRef.current = null;
    setTurnstileToken("");
  }

  // Load the Turnstile API script and render the widget once it's ready.
  // We guard against a race where `window.turnstile` isn't available yet
  // when the effect first runs by attaching a load listener on the script.
  useEffect(() => {
    if (!siteKey || typeof window === "undefined") return;

    console.log("siteKey", siteKey);
    const renderWidget = () => {
      const turnstile = window.turnstile;
      if (!turnstile || !turnstileRef.current) return;

      const id = turnstile.render(turnstileRef.current, {
        sitekey: siteKey,
        callback: (token: string) => setTurnstileToken(token),
        "expired-callback": () => setTurnstileToken(""),
      });
      widgetIdRef.current = id;
    };

    // Already loaded?
    if (window.turnstile) {
      renderWidget();
      return;
    }

    const script = document.createElement("script");
    script.src = "https://challenges.cloudflare.com/turnstile/v0/api.js";
    script.async = true;
    script.defer = true;

    const onLoad = () => {
      renderWidget();
    };

    script.addEventListener("load", onLoad);
    document.head.appendChild(script);

    return () => {
      script.removeEventListener("load", onLoad);
      document.head.removeChild(script);
      removeTurnstileWidget();
    };
  }, [siteKey]);

  // Clean up the Turnstile widget whenever we leave the email step.
  // The widget is rendered inside the email-step form, so when the user
  // submits and step flips to 'otp', React removes the widget's DOM node.
  // Without calling turnstile.remove() here, Cloudflare warns:
  // "Cannot find Widget ... consider using turnstile.remove() to clean up a widget."
  useEffect(() => {
    return () => {
      removeTurnstileWidget();
    };
  }, [step]);

  const requestOTP = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      setError(null);
      setLoading(true);

      // Turnstile is enforced by the backend when TURNSTILE_ENABLED=true.
      // Fail fast here only when a site key is configured so the UX is clear.
      if (siteKey && !turnstileToken) {
        setError("Please complete the security check above.");
        setLoading(false);
        return;
      }

      try {
        const res = await fetch("/api/auth/otp/request", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          credentials: "include",
          body: JSON.stringify({ email, turnstile_token: turnstileToken }),
        });

        if (!res.ok) {
          const body = await res.json().catch(() => ({}));
          setError(body.message || "Failed to send code. Please try again.");
          setLoading(false);
          return;
        }

        const data = await res.json();
        setChallengeId(data.challenge_id || "");
        setStep("otp");
      } catch {
        setError("Failed to send code. Please try again.");
        setLoading(false);
      } finally {
        setLoading(false);
      }
    },
    [email, turnstileToken, siteKey],
  );

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
        // Session cookie is set by backend via Set-Cookie on the response.
        // Use window.location.assign for hard navigation (not client-side router)
        // so the browser sends the new cookies on the follow-up request.
        const nextPath = data.needs_onboarding ? "/onboarding" : "/";
        window.location.assign(nextPath);
      } catch {
        setError("Failed to verify code. Please try again.");
        setLoading(false);
      } finally {
        setLoading(false);
      }
    },
    [code, challengeId],
  );

  return (
    <div className="flex min-h-screen flex-col items-center justify-center p-4">
      {siteKey}
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
              </div>

              {siteKey && <div ref={turnstileRef} className="cf-turnstile" />}

              {error && <p className="text-sm text-destructive">{error}</p>}
              <button
                type="submit"
                disabled={
                  loading || !email || (Boolean(siteKey) && !turnstileToken)
                }
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
