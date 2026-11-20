'use client'

import { useState, useCallback } from 'react'
import { useRouter } from 'next/navigation'

/**
 * Onboarding page — shown after OTP verification when no account exists.
 *
 * Collects organization name, school name, and user name in a single POST
 * to /api/auth/onboard. Requires a valid temporary session cookie (set by
 * the backend during OTP verification). On success, the session is upgraded
 * to full and the middleware redirects to /dashboard.
 */
export default function OnboardingPage() {
  const router = useRouter()
  const [organizationName, setOrganizationName] = useState('')
  const [schoolName, setSchoolName] = useState('')
  const [userName, setUserName] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const submit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault()
      setError(null)
      setLoading(true)

      try {
        const res = await fetch('/api/auth/onboard', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          credentials: 'include',
          body: JSON.stringify({
            organization_name: organizationName,
            school_name: schoolName,
            user_name: userName,
          }),
        })

        if (!res.ok) {
          const body = await res.json().catch(() => ({}))
          setError(body.message || 'Failed to complete onboarding. Please try again.')
          setLoading(false)
          return
        }

        // Session upgraded to full by backend. Middleware redirects to /dashboard.
        router.replace('/dashboard')
      } catch {
        setError('Failed to complete onboarding. Please try again.')
        setLoading(false)
      } finally {
        setLoading(false)
      }
    },
    [organizationName, schoolName, userName, router]
  )

  return (
    <div className="flex min-h-screen flex-col items-center justify-center p-4">
      <div className="w-full max-w-md space-y-6">
        <h1 className="text-3xl font-bold">Set up your organization</h1>
        <p className="text-muted-foreground">
          Welcome! Let's create your organization and school to get started.
        </p>
        <form onSubmit={submit} className="space-y-4">
          <div className="space-y-2">
            <label htmlFor="organization_name" className="block text-sm font-medium">
              Organization name
            </label>
            <input
              id="organization_name"
              type="text"
              placeholder="e.g. Acme School District"
              value={organizationName}
              onChange={(e) => setOrganizationName(e.target.value)}
              required
              disabled={loading}
              className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:opacity-50"
            />
          </div>

          <div className="space-y-2">
            <label htmlFor="school_name" className="block text-sm font-medium">
              School name
            </label>
            <input
              id="school_name"
              type="text"
              placeholder="e.g. Lincoln High School"
              value={schoolName}
              onChange={(e) => setSchoolName(e.target.value)}
              required
              disabled={loading}
              className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:opacity-50"
            />
          </div>

          <div className="space-y-2">
            <label htmlFor="user_name" className="block text-sm font-medium">
              Your name
            </label>
            <input
              id="user_name"
              type="text"
              placeholder="e.g. Jane Doe"
              value={userName}
              onChange={(e) => setUserName(e.target.value)}
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
            disabled={loading || !organizationName || !schoolName || !userName}
            className="w-full inline-flex items-center justify-center rounded-md border border-transparent bg-primary px-4 py-2 text-sm font-medium text-primary-foreground shadow-sm hover:bg-primary/90 disabled:opacity-50"
          >
            {loading ? 'Setting up…' : 'Create organization & school'}
          </button>
        </form>
      </div>
    </div>
  )
}
