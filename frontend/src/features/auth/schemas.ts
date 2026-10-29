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
  currentStep: z.string().min(1, 'Current step is required'),
  formData: z.record(z.string(), z.unknown()),
  schoolName: z.string().optional(),
});

export type UpdateOnboardingInput = z.infer<typeof updateOnboardingSchema>;

export const completeOnboardingSchema = z.object({
  schoolName: z.string().min(1, 'School name is required').max(255),
  idempotencyKey: z.string().min(1, 'Idempotency key is required'),
});

export type CompleteOnboardingInput = z.infer<typeof completeOnboardingSchema>;

export const loginFormSchema = z.object({
  email: z.string().email('Invalid email address'),
});

export type LoginFormInput = z.infer<typeof loginFormSchema>;

export const tenantNameFormSchema = z.object({
  schoolName: z.string().min(1, 'School name is required').max(255),
});

export type TenantNameFormInput = z.infer<typeof tenantNameFormSchema>;