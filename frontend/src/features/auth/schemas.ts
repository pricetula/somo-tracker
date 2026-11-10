/**
 * Auth feature Zod validation schemas
 */

import { z } from 'zod';

export const requestMagicLinkSchema = z.object({
  email: z.string().email('Invalid email address'),
});

export type RequestMagicLinkInput = z.infer<typeof requestMagicLinkSchema>;

export const authenticateTokenSchema = z.object({
  token: z.string().min(1, 'Token is required'),
});

export type AuthenticateTokenInput = z.infer<typeof authenticateTokenSchema>;

export const updateOnboardingSchema = z.object({
  current_step: z.string().min(1, 'Current step is required'),
  form_data: z.record(z.string(), z.unknown()),
  school_name: z.string().optional(),
});

export type UpdateOnboardingInput = z.infer<typeof updateOnboardingSchema>;
// Compile-time check: ensure schema matches generated API type
// eslint-disable-next-line @typescript-eslint/no-unused-vars
const _updateOnboardingTypeCheck: UpdateOnboardingInput extends import('./types').UpdateOnboardingRequest ? true : false = true;

export const completeOnboardingSchema = z.object({
  school_name: z.string().min(1, 'School name is required').max(255).transform((val) => val.trim()).refine((val) => val.length > 0, 'School name is required'),
  idempotency_key: z.string().min(1, 'Idempotency key is required'),
});

export type CompleteOnboardingInput = z.infer<typeof completeOnboardingSchema>;
// Compile-time check: ensure schema matches generated API type
// eslint-disable-next-line @typescript-eslint/no-unused-vars
const _completeOnboardingTypeCheck: CompleteOnboardingInput extends import('./types').CompleteOnboardingRequest ? true : false = true;

export const loginFormSchema = z.object({
  email: z.string().email('Invalid email address'),
});

export type LoginFormInput = z.infer<typeof loginFormSchema>;

export const tenantNameFormSchema = z.object({
  school_name: z.string().min(1, 'School name is required').max(255).transform((val) => val.trim()).refine((val) => val.length > 0, 'School name is required'),
});

export type TenantNameFormInput = z.infer<typeof tenantNameFormSchema>;