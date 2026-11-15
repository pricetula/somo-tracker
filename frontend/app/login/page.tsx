'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'

export default function LoginPage() {
  const [email, setEmail] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const router = useRouter()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)
    setLoading(true)

    try {
      const res = await fetch('/api/auth/request-magic-link', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email }),
      })

      if (!res.ok) {
        const errData = await res.json().catch(() => ({}))
        setError(errData.message || 'Failed to send magic link')
        return
      }

      // Generic success - don't reveal if email exists
      alert('If the email address is eligible, a sign-in link has been sent. Check your inbox.')
      router.push('/auth/callback')
    } catch {
      setError('Failed to send magic link')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex flex-col items-center justify-center min-h-screen px-4">
      <div className="w-full max-w-md space-y-6">
        <h1 className="text-2xl font-bold">Sign in to Somotracker</h1>
        <p className="text-muted-foreground">
          Enter your email to receive a magic link
        </p>

        {error && (
          <div className="rounded-md bg-red-50 p-3 text-sm text-red-700">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-foreground">
              Email address
            </label>
            <input
              type="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="mt-1 block w-full rounded-md border border-input bg-background px-3 py-2 text-sm focus:border-primary focus:outline-none"
              placeholder="you@school.edu"
            />
          </div>
          <button
            type="submit"
            disabled={loading}
            className="inline-flex w-full items-center gap-2 px-3 py-1.5 text-xs font-medium rounded-md border border-input bg-input/50 hover:bg-input/80 disabled:opacity-50 text-foreground"
          >
            {loading ? 'Sending…' : 'Send magic link'}
          </button>
        </form>
      </div>
    </div>
  )
}
