'use client'

import { Suspense, useEffect, useState } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'
import { Button } from '@/components/ui/button'

export const dynamic = 'force-dynamic'

/**
 * Inner callback content — uses useSearchParams, must be wrapped in Suspense.
 */
function CallbackContent() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false

    async function authenticate() {
      const token = searchParams.get('token')

      if (!token) {
        if (!cancelled) {
          setError('No magic link token found. Please request a new link.')
          setLoading(false)
        }
        return
      }

      try {
        const res = await fetch('/api/auth/authenticate-token', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ token }),
          credentials: 'include',
        })

        if (!res.ok) {
          if (!cancelled) {
            setError('Invalid or expired magic link. Please request a new one.')
            setLoading(false)
          }
          return
        }

        const data = await res.json()

        // If onboarding is required, redirect to onboarding flow
        if (data.auth_state === 'pending_onboarding' || data.ticket_token) {
          if (!cancelled) {
            router.replace('/onboarding')
          }
          return
        }

        // Successful auth — session cookie is set by Go (HttpOnly)
        if (!cancelled) {
          router.replace('/dashboard')
        }
      } catch {
        if (!cancelled) {
          setError('Failed to authenticate. Please try again.')
          setLoading(false)
        }
      }
    }

    authenticate()

    return () => {
      cancelled = true
    }
  }, [searchParams, router])

  if (loading) {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center p-4">
        <div className="w-full max-w-md space-y-6">
          <h1 className="text-3xl font-bold">Signing in…</h1>
          <p className="text-muted-foreground">
            Please wait while we verify your magic link.
          </p>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center p-4">
        <div className="w-full max-w-md space-y-6">
          <div>
            <h1 className="text-3xl font-bold">Sign in failed</h1>
            <p className="mt-2 text-muted-foreground">{error}</p>
          </div>
          <Button onClick={() => router.push('/login')}>
            Request new magic link
          </Button>
        </div>
      </div>
    )
  }

  return null
}

/**
 * Auth callback page — public route.
 *
 * Stytch redirects here after the user clicks the magic link in their email.
 * The URL contains a ?token=... query parameter.
 *
 * We POST the token to BFF /api/auth/authenticate-token → Go validates it,
 * exchanges the IST for a real Stytch session, and sets the
 * __Host-session_token cookie (HttpOnly). Then we redirect to /dashboard.
 *
 * If onboarding is required (new user), Go returns auth_state and we
 * redirect to the onboarding flow instead.
 */
export default function CallbackPage() {
  return (
    <Suspense
      fallback={
        <div className="flex min-h-screen flex-col items-center justify-center p-4">
          <div className="w-full max-w-md space-y-6">
            <h1 className="text-3xl font-bold">Signing in…</h1>
            <p className="text-muted-foreground">
              Please wait while we verify your magic link.
            </p>
          </div>
        </div>
      }
    >
      <CallbackContent />
    </Suspense>
  )
}
