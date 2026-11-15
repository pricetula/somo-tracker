import { cookies } from 'next/headers'

export function getSessionCookieName(): string {
  return process.env.NODE_ENV === 'production'
    ? '__Host-session_token'
    : 'session_token_development'
}

export function getOnboardingCookieName(): string {
  return process.env.NODE_ENV === 'production'
    ? '__Host-onboarding'
    : 'onboarding_development'
}

export function getPkceCookieName(): string {
  return process.env.NODE_ENV === 'production'
    ? '__Host-pkce_verifier'
    : 'pkce_verifier_development'
}

export function isProduction(): boolean {
  return process.env.NODE_ENV === 'production'
}

export function getBackendUrl(): string {
  return process.env.GO_BACKEND_URL || 'http://127.0.0.1:8080'
}

export function getAppUrl(): string {
  return process.env.NEXT_PUBLIC_APP_URL || 'http://localhost:3000'
}
