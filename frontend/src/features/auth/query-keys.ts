/**
 * Auth feature query keys
 * Follows pattern: [schoolId, '<feature>', '<scope>', params]
 * Built only in this feature's query-keys.ts
 */

import type { OnboardingStep } from './types';

export const authQueryKeys = {
  all: ['auth'] as const,
  onboarding: () => [...authQueryKeys.all, 'onboarding'] as const,
  onboardingCurrent: () =>
    [...authQueryKeys.onboarding(), 'current'] as const,
  session: (schoolId: string) => [...authQueryKeys.all, schoolId, 'session'] as const,
} as const;

export const authMutationKeys = {
  requestMagicLink: () => [...authQueryKeys.all, 'request-magic-link'] as const,
  authenticateToken: () => [...authQueryKeys.all, 'authenticate-token'] as const,
  updateOnboarding: () =>
    [...authQueryKeys.all, 'update-onboarding'] as const,
  completeOnboarding: () =>
    [...authQueryKeys.all, 'complete-onboarding'] as const,
  logout: (schoolId: string) => [...authQueryKeys.all, schoolId, 'logout'] as const,
} as const;

export function getOnboardingStepRoute(step: OnboardingStep): string {
  return `/onboarding/${step}`;
}

export function getAuthCallbackRoute(redirect?: string): string {
  const base = '/auth/callback';
  if (redirect) {
    return `${base}?redirect=${encodeURIComponent(redirect)}`;
  }
  return base;
}

export function getLoginRoute(redirect?: string): string {
  const base = '/login';
  if (redirect) {
    return `${base}?redirect=${encodeURIComponent(redirect)}`;
  }
  return base;
}