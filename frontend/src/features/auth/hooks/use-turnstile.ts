import { useCallback, useState } from "react";
import { useForm } from "react-hook-form";
import type { RequestOTPFormValues } from "../schemas/otp.schema";

/**
 * Wires the Cloudflare Turnstile widget to the email form:
 * - on verify: store the token, clear any error, set the form field
 * - on expire: clear the token, unset the form field, surface a required error
 * - on error: clear the token and unset the form field
 *
 * Returns the token state and the three callbacks to pass to <Turnstile />.
 */
export function useTurnstile(emailForm: ReturnType<typeof useForm<RequestOTPFormValues>>) {
  const [turnstileToken, setTurnstileToken] = useState<string | null>(null);

  const onTurnstileVerify = useCallback(
    (token: string) => {
      setTurnstileToken(token);
      emailForm.clearErrors("turnstile_token");
      emailForm.setValue("turnstile_token", token, { shouldValidate: true });
    },
    [emailForm],
  );

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

  const reset = useCallback(() => {
    setTurnstileToken(null);
    emailForm.setValue("turnstile_token", null);
  }, [emailForm]);

  return { turnstileToken, onTurnstileVerify, onTurnstileExpire, onTurnstileError, reset };
}
