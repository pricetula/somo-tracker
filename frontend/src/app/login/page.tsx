'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { Button } from '@/components/ui/button'

/**
 * Login page — public route.
 *
 * User enters their email to request a magic link.
 * POSTs to BFF /api/auth/request-magic-link → Go /api/v1/auth/request-magic-link.
 * Go sets the __Host-pkce_verifier cookie and sends the magic link via Stytch.
 * User clicks the link in email → Stytch redirects to /auth/callback?token=...
 */
export default function LoginPage() {
  const router = useRouter()
  const [email, setEmail] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [sent, setSent] = useState(false)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    setLoading(true)

    try {
      const res = await fetch('/api/auth/request-magic-link', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email }),
        // Include cookies so the browser stores the Set-Cookie (pkce_verifier)
        // returned by the BFF. Without this the PKCE cookie can be dropped on
        // some clients, causing MISSING_PKCE on authenticate-token.
        credentials: 'include',
      })

      if (!res.ok) {
        // Generic success per backend — don't reveal if email exists
        setSent(true)
        setEmail('')
        return
      }

      const data = await res.json()
      if (data.success) {
        setSent(true)
        setEmail('')
      } else {
        setError('Failed to send magic link. Please try again.')
      }
    } catch {
      setError('Failed to send magic link. Please try again.')
    } finally {
      setLoading(false)
    }
  }

  if (sent) {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center p-4">
        <div className="w-full max-w-md space-y-6">
          <div>
            <h1 className="text-3xl font-bold">Check your email</h1>
            <p className="mt-2 text-muted-foreground">
              A magic link has been sent to {email}. Click it to sign in.
            </p>
          </div>
          <Button onClick={() => router.push('/login')}>
            Back to login
          </Button>
        </div>
      </div>
    )
  }

  return (
    <div className="flex min-h-screen flex-col items-center justify-center p-4">
      <div className="w-full max-w-md space-y-6">
        <div>
          <h1 className="text-3xl font-bold">Sign in</h1>
          <p className="mt-2 text-muted-foreground">
            Enter your school email to receive a magic link.
          </p>
        </div>

        {error && (
          <div className="rounded-md border border-red-500 bg-red-50 p-4">
            <p className="text-sm text-red-700">{error}</p>
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <label htmlFor="email" className="text-sm font-medium">
              Email address
            </label>
            <input
              id="email"
              type="email"
              autoComplete="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full rounded-border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
              placeholder="name@school.ac.ke"
            />
          </div>

          <Button type="submit" disabled={loading} className="w-full">
            {loading ? 'Sending magic link…' : 'Send magic link'}
          </Button>
        </form>
      </div>
    </div>
  )
}
