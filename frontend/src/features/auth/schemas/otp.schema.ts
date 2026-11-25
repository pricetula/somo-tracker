import { z } from "zod";

export const requestOTPSchema = z.object({
  email: z.string().email("Invalid email address"),
  turnstile_token: z.string().min(1, "Turnstile verification is required").nullable(),
});

export const verifyOTPSchema = z.object({
  code: z.string().length(6, "Code must be 6 digits"),
});

export type RequestOTPFormValues = z.infer<typeof requestOTPSchema>;
export type VerifyOTPFormValues = z.infer<typeof verifyOTPSchema>;
