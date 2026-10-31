/**
 * Auth feature types
 * Generated from backend OpenAPI spec - do not edit manually
 */

// Generated from OpenAPI spec - do not edit manually
import type { components } from './api/generated/types';

export type RequestMagicLinkRequest = components['schemas']['RequestMagicLinkRequest'];
export type RequestMagicLinkResponse = components['schemas']['RequestMagicLinkResponse'];
export type AuthenticateTokenRequest = components['schemas']['AuthenticateTokenRequest'];
export type AuthenticateTokenResponse = components['schemas']['AuthenticateTokenResponse'];
export type OnboardingState = components['schemas']['OnboardingState'];
export type GetOnboardingResponse = components['schemas']['OnboardingStateResponse'];
export type UpdateOnboardingRequest = components['schemas']['UpdateOnboardingRequest'];
export type CompleteOnboardingRequest = components['schemas']['ProvisionTenantRequest'];
export type CompleteOnboardingResponse = components['schemas']['ProvisionTenantResponse'];
export type LogoutRequest = Record<string, never>;
export type LogoutResponse = components['schemas']['LogoutResponse'];
export type ErrorResponse = components['schemas']['ErrorResponse'];

export interface AuthState {
  hasSession: boolean;
}

export type OnboardingStep =
  | 'tenant-name'
  | 'school-details'
  | 'admin-user'
  | 'complete';

export const ONBOARDING_STEPS: OnboardingStep[] = [
  'tenant-name',
  'school-details',
  'admin-user',
  'complete',
];

export function isOnboardingStep(step: string): step is OnboardingStep {
  return ONBOARDING_STEPS.includes(step as OnboardingStep);
}