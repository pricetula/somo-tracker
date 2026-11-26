"use client";

import { useState, useCallback } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useRequestOTP } from "@/features/auth/hooks/use-request-otp";
import { useVerifyOTP } from "@/features/auth/hooks/use-verify-otp";
import {
  requestOTPSchema,
  verifyOTPSchema,
} from "@/features/auth/schemas/otp.schema";
import type {
  RequestOTPFormValues,
  VerifyOTPFormValues,
} from "@/features/auth/schemas/otp.schema";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Turnstile } from "@/components/turnstile";
import { useTurnstile } from "@/features/auth/hooks/use-turnstile";

export default function LoginForm() {
  const [step, setStep] = useState<"email" | "otp">("email");
  const [challengeId, setChallengeId] = useState<string | null>(null);

  const requestOTPMutation = useRequestOTP();
  const verifyOTPMutation = useVerifyOTP();

  const emailForm = useForm<RequestOTPFormValues>({
    resolver: zodResolver(requestOTPSchema),
    defaultValues: { email: "", turnstile_token: null },
    mode: "onChange",
  });

  const otpForm = useForm<VerifyOTPFormValues>({
    resolver: zodResolver(verifyOTPSchema),
    defaultValues: { code: "" },
    mode: "onChange",
  });

  const {
    turnstileToken,
    onTurnstileVerify,
    onTurnstileExpire,
    onTurnstileError,
    reset,
  } = useTurnstile(emailForm);

  const handleVerifyOTP = useCallback(
    async (values: VerifyOTPFormValues) => {
      const data = await verifyOTPMutation.mutateAsync({
        challengeId: challengeId!,
        code: values.code,
      });
      window.location.assign(data.needsOnboarding ? "/onboarding" : "/");
    },
    [challengeId, verifyOTPMutation],
  );

  return (
    <div className="flex min-h-screen items-center justify-center p-4 sm:p-6">
      <div className="max-w-md flex flex-col gap-6">
        {step === "email" && (
          <>
            <h1 className="text-3xl font-bold">Sign in</h1>
            <p className="text-muted-foreground text-sm">
              Enter your email to receive a verification code.
            </p>

            <form
              onSubmit={emailForm.handleSubmit(async (values) => {
                reset();
                const data = await requestOTPMutation.mutateAsync({
                  email: values.email,
                  turnstileToken: values.turnstile_token,
                });
                setChallengeId(data.challengeId);
                setStep("otp");
              })}
              className="space-y-2"
            >
              <Label htmlFor="email">Email address</Label>
              <Input
                id="email"
                type="email"
                autoComplete="email"
                placeholder="you@example.com"
                {...emailForm.register("email")}
                disabled={requestOTPMutation.isPending}
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

              {emailForm.formState.errors.root && (
                <p className="text-sm text-destructive">
                  {emailForm.formState.errors.root.message}
                </p>
              )}

              <Button
                type="submit"
                disabled={requestOTPMutation.isPending || !turnstileToken}
              >
                {requestOTPMutation.isPending ? "Sending…" : "Send code"}
              </Button>
            </form>
          </>
        )}
        {step === "otp" && (
          <>
            <h1 className="text-3xl font-bold">Verify code</h1>
            <p className="text-muted-foreground text-sm">
              Enter the 6-digit code sent to{" "}
              <span className="font-medium">
                {emailForm.getValues("email")}
              </span>
              .
            </p>
            <Button
              type="button"
              variant="link"
              size="sm"
              onClick={() => setStep("email")}
            >
              Use a different email
            </Button>

            <form
              onSubmit={otpForm.handleSubmit(handleVerifyOTP)}
              className="space-y-4"
            >
              <Label htmlFor="code">Verification code</Label>
              <Input
                id="code"
                type="text"
                autoComplete="one-time-code"
                maxLength={6}
                {...otpForm.register("code")}
                disabled={verifyOTPMutation.isPending}
                className="letter-spacing-widest"
              />
              {otpForm.formState.errors.code && (
                <p className="text-sm text-destructive">
                  {otpForm.formState.errors.code.message}
                </p>
              )}

              {otpForm.formState.errors.root && (
                <p className="text-sm text-destructive">
                  {otpForm.formState.errors.root.message}
                </p>
              )}

              <Button
                type="submit"
                disabled={verifyOTPMutation.isPending}
                className="w-full"
              >
                {verifyOTPMutation.isPending ? "Verifying…" : "Verify"}
              </Button>
            </form>
          </>
        )}
      </div>
    </div>
  );
}

export { LoginForm };
