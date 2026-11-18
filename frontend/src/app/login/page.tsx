'use client'

import { useState, useCallback } from 'react'
import { useRouter } from 'next/navigation'

/**
 * Login page — email entry for OTP authentication.
 *
 * Step 1: User enters email → POST /api/auth/otp/request → backend sends OTP via email
 * Step 2: User enters 6-digit code → POST /api/auth/otp/verify → backend creates session
 * Step 3: Redirect to /dashboard
 */
export default function LoginPage() {
  const router = useRouter()
  const [step, setStep] = useState<'email' | 'otp'>('email')
  const [email, setEmail] = useState('')
  const [code, setCode] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const requestOTP = useCallback(async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)
    setLoading(true)

    try {
      const res = await fetch('/api/auth/otp/request', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ email }),
      })

      if (!res.ok) {
        const body = await res.json().catch(() => ({}))
        setError(body.message || 'Failed to send code. Please try again.')
        setLoading(false)
        return
      }

      setStep('otp')
    } catch {
      setError('Failed to send code. Please try again.')
      setLoading(false)
    } finally {
      setLoading(false)
    }
  }, [email])

  const verifyOTP = useCallback(async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)
    setLoading(true)

    try {
      const res = await fetch('/api/auth/otp/verify', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ challenge_id: '', code }),
      })

      if (!res.ok) {
        const body = await res.json().catch(() => ({}))
        setError(body.message || 'Invalid code. Please try again.')
        setLoading(false)
        return
      }

      // Session cookie is set by backend via Set-Cookie on the response.
      router.replace('/dashboard')
    } catch {
      setError('Failed to verify code. Please try again.')
      setLoading(false)
    } finally {
      setLoading(false)
    }
  }, [code, router])

  return (
    <div className="flex min-h-screen flex-col items-center justify-center p-4">
      <div className="w-full max-w-md space-y-6">
        {step === 'email' && (
          <>
            <h1 className="text-3xl font-bold">Sign in</h1>
            <p className="text-muted-foreground">
              Enter your email to receive a verification code.
            </p>
            <form onSubmit={requestOTP} className="space-y-4">
              <div className="space-y-2">
                <label htmlFor="email" className="block text-sm font-medium">
                  Email address
                </label>
                <input
                  id="email"
                  type="email"
                  placeholder="you@example.com"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  required
                  disabled={loading}
                  className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:opacity-50"
                />
              </div>
              {error && (
                <p className="text-sm text-destructive">{error}</p>
              )}
              <button
                type="submit"
                disabled={loading || !email}
                className="w-full inline-flex items-center justify-center rounded-md border border-transparent bg-primary px-4 py-2 text-sm font-medium text-primary-foreground shadow-sm hover:bg-primary/90 disabled:opacity-50"
              >
                {loading ? 'Sending…' : 'Send code'}
              </button>
            </form>
          </>
        )}
        {step === 'otp' && (
          <>
            <h1 className="text-3xl font-bold">Verify code</h1>
            <p className="text-muted-foreground">
              Enter the 6-digit code sent to{' '}
              <strong className="font-medium">{email}</strong>.
            </p>
            <form onSubmit={verifyOTP} className="space-y-4">
              <div className="space-y-2">
                <label htmlFor="code" className="block text-sm font-medium">
                  Verification code
                </label>
                <input
                  id="code"
                  type="text"
                  maxLength={6}
                  value={code}
                  onChange={(e) => setCode(e.target.value.replace(/\D/g, ''))}
                  required
                  disabled={loading}
                  className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:opacity-50 letter-spacing-widest"
                />
              </div>
              {error && (
                <p className="text-sm text-destructive">{error}</p>
              )}
              <button
                type="submit"
                disabled={loading || code.length !== 6}
                className="w-full inline-flex items-center justify-center rounded-md border border-transparent bg-primary px-4 py-2 text-sm font-medium text-primary-foreground shadow-sm hover:bg-primary/90 disabled:opacity-50"
              >
                {loading ? 'Verifying…' : 'Verify'}
              </button>
            </form>
          </>
        )}
      </div>
    </div>
  )
}
