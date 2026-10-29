/**
 * Auth feature types
 * Generated from backend OpenAPI spec - do not edit manually
 */

export interface RequestMagicLinkRequest {
  email: string;
}

export interface RequestMagicLinkResponse {
  success: boolean;
}

export interface AuthenticateTokenRequest {
  token: string;
}

export interface AuthenticateTokenResponse {
  status: 'onboarding_required' | 'organization_selection_required' | 'authenticated';
  auth_state: 'onboarding_required' | 'organization_selection_required' | 'authenticated';
  ticket_token?: string;
  onboarding?: OnboardingState;
}

export interface OnboardingState {
  verifiedEmail: string;
  currentStep: string;
  formData: Record<string, unknown>;
  schoolName?: string;
  tenantId?: string;
}

export interface GetOnboardingResponse {
  verifiedEmail: string;
  currentStep: string;
  formData: Record<string, unknown>;
  schoolName?: string;
  tenantId?: string;
}

export interface UpdateOnboardingRequest {
  currentStep: string;
  formData: Record<string, unknown>;
  schoolName?: string;
}

export interface CompleteOnboardingRequest {
  schoolName: string;
  idempotencyKey: string;
}

export interface CompleteOnboardingResponse {
  success: boolean;
}

export type LogoutRequest = Record<string, never>;

export interface LogoutResponse {
  success: boolean;
}

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