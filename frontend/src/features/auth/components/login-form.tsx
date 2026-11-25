"use client";

import { useState, useCallback } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useRequestOTP } from "@/features/auth/hooks/use-request-otp";
import { useVerifyOTP } from "@/features/auth/hooks/use-verify-otp";
import { requestOTPSchema, verifyOTPSchema } from "@/features/auth/schemas/otp.schema";
import type { RequestOTPFormValues, VerifyOTPFormValues } from "@/features/auth/schemas/otp.schema";
import { Turnstile } from "@/components/turnstile";

/**
 * Login form — email entry for OTP authentication.
 *
 * Step 1: User enters email → POST /api/auth/otp/request → backend sends OTP via email
 * Step 2: User enters 6-digit code → POST /api/auth/otp/verify → backend creates session
 * Step 3: Redirect to / or /onboarding
 *
 * Uses react-hook-form + zod for validation and TanStack Query mutations for API calls.
 * Turnstile site key is hardcoded in the Turnstile component itself.
 */
export default function LoginForm() {
  const [step, setStep] = useState<"email" | "otp">("email");
  const [turnstileToken, setTurnstileToken] = useState<string | null>(null);
  const [challengeId, setChallengeId] = useState<string | null>(null);

  const requestOTPMutation = useRequestOTP();
  const verifyOTPMutation = useVerifyOTP();

  const emailForm = useForm<RequestOTPFormValues>({
    resolver: zodResolver(requestOTPSchema),
    defaultValues: {
      email: "",
      turnstile_token: null,
    },
    mode: "onChange",
  });

  const otpForm = useForm<VerifyOTPFormValues>({
    resolver: zodResolver(verifyOTPSchema),
    defaultValues: {
      code: "",
    },
    mode: "onChange",
  });

  const onTurnstileVerify = useCallback((token: string) => {
    setTurnstileToken(token);
    emailForm.clearErrors("turnstile_token");
    emailForm.setValue("turnstile_token", token, { shouldValidate: true });
  }, [emailForm]);

  const onTurnstileExpire = useCallback(() => {
    setTurnstileToken(null);
    emailForm.setValue("turnstile_token", null, { shouldValidate: true });
    emailForm.setError("turnstile_token", {
      type: "required",
      message: "Turnstile verification is required",
    });
  }, [emailForm]);

  const onTurnstileError = useCallback(() => {
    setTurnstileToken(null);
    emailForm.setValue("turnstile_token", null, { shouldValidate: true });
  }, [emailForm]);

  const handleVerifyOTP = useCallback(
    async (values: VerifyOTPFormValues) => {
      const data = await verifyOTPMutation.mutateAsync({
        challengeId: challengeId!,
        code: values.code,
      });

      const nextPath = data.needsOnboarding ? "/onboarding" : "/";
      window.location.assign(nextPath);
    },
    [challengeId, verifyOTPMutation]
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
            <form
              onSubmit={emailForm.handleSubmit(async (values) => {
                setTurnstileToken(null);
                emailForm.setValue("turnstile_token", null);

                const data = await requestOTPMutation.mutateAsync({
                  email: values.email,
                  turnstileToken: values.turnstile_token,
                });
                setChallengeId(data.challengeId);
                setStep("otp");
              })}
              className="space-y-4"
            >
              <div className="space-y-2">
                <label htmlFor="email" className="block text-sm font-medium">
                  Email address
                </label>
                <input
                  id="email"
                  type="email"
                  placeholder="you@example.com"
                  {...emailForm.register("email")}
                  disabled={requestOTPMutation.isPending}
                  className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:opacity-50"
                />
                {emailForm.formState.errors.email && (
                  <p className="text-sm text-destructive">
                    {emailForm.formState.errors.email.message}
                  </p>
                )}
                <Turnstile
                  onVerify={onTurnstileVerify}
                  onExpire={onTurnstileExpire}
                  onError={onTurnstileError}
                />
                {emailForm.formState.errors.turnstile_token && (
                  <p className="text-sm text-destructive">
                    {emailForm.formState.errors.turnstile_token.message}
                  </p>
                )}
              </div>

              {emailForm.formState.errors.root && (
                <p className="text-sm text-destructive">
                  {emailForm.formState.errors.root.message}
                </p>
              )}
              <button
                type="submit"
                disabled={
                  requestOTPMutation.isPending ||
                  !emailForm.getValues("email") ||
                  !turnstileToken
                }
                className="w-full inline-flex items-center justify-center rounded-md border border-transparent bg-primary px-4 py-2 text-sm font-medium text-primary-foreground shadow-sm hover:bg-primary/90 disabled:opacity-50"
              >
                {requestOTPMutation.isPending ? "Sending…" : "Send code"}
              </button>
            </form>
          </>
        )}
        {step === "otp" && (
          <>
            <h1 className="text-3xl font-bold">Verify code</h1>
            <p className="text-muted-foreground">
              Enter the 6-digit code sent to{" "}
              <strong className="font-medium">{emailForm.getValues("email")}</strong>.
            </p>
            <form onSubmit={otpForm.handleSubmit(handleVerifyOTP)} className="space-y-4">
              <div className="space-y-2">
                <label htmlFor="code" className="block text-sm font-medium">
                  Verification code
                </label>
                <input
                  id="code"
                  type="text"
                  maxLength={6}
                  {...otpForm.register("code")}
                  disabled={verifyOTPMutation.isPending}
                  className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:opacity-50 letter-spacing-widest"
                />
                {otpForm.formState.errors.code && (
                  <p className="text-sm text-destructive">
                    {otpForm.formState.errors.code.message}
                  </p>
                )}
              </div>

              {otpForm.formState.errors.root && (
                <p className="text-sm text-destructive">
                  {otpForm.formState.errors.root.message}
                </p>
              )}
              <button
                type="submit"
                disabled={
                  verifyOTPMutation.isPending ||
                  otpForm.getValues("code").length !== 6
                }
                className="w-full inline-flex items-center justify-center rounded-md border border-transparent bg-primary px-4 py-2 text-sm font-medium text-primary-foreground shadow-sm hover:bg-primary/90 disabled:opacity-50"
              >
                {verifyOTPMutation.isPending ? "Verifying…" : "Verify"}
              </button>
            </form>
          </>
        )}
      </div>
    </div>
  );
}

export { LoginForm };
