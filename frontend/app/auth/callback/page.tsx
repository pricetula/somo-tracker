'use client'

import { useEffect, useState } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'

export default function AuthCallback() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const token = searchParams.get('token')
    const errorParam = searchParams.get('error')

    if (errorParam) {
      setError(errorParam)
      return
    }

    if (!token) {
      setError('No token provided')
      return
    }

    fetch('/api/auth/authenticate-token', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ token }),
    })
      .then(async (res) => {
        if (!res.ok) {
          const errData = await res.json().catch(() => ({}))
          setError(errData.message || 'Authentication failed')
          return
        }
        const data = await res.json()
        if (data.auth_state === 'organization_selection_required' || data.status === 'tenant-name') {
          router.push('/onboarding/tenant-name')
        } else {
          router.push('/dashboard')
        }
      })
      .catch(() => {
        setError('Authentication failed')
      })
  }, [searchParams, router])

  if (error) {
    return (
      <div className="p-6 flex flex-col items-center justify-center min-h-[400px]">
        <h1 className="text-red-500 font-bold">Authentication failed</h1>
        <p className="mt-2 text-muted-foreground">{error}</p>
        <button
          onClick={() => router.push('/login')}
          className="mt-4 inline-flex items-center gap-2 px-3 py-1.5 text-xs font-medium rounded-md border border-input bg-input/50 hover:bg-input/80 text-foreground"
        >
          Back to login
        </button>
      </div>
    )
  }

  return (
    <div className="p-6 flex flex-col items-center justify-center min-h-[400px]">
      <div className="animate-pulse rounded-lg bg-muted w-96 h-32" />
      <p className="mt-4 text-sm text-muted-foreground">Signing in…</p>
    </div>
  )
}
