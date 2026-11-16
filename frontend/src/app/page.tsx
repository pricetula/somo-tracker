'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'

interface AuthUser {
  userId: string
  fullName: string
  email: string
  role: string
  activeSchool?: { id: string; name: string }
  schools: Array<{ id: string; name: string }>
}

/**
 * Root page — redirects based on authentication state.
 *
 * Client component that fetches /api/me on mount and redirects:
 * - Authenticated users → /dashboard
 * - Unauthenticated users → /login
 */
export default function RootPage() {
  const router = useRouter()
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let cancelled = false

    async function checkAuth() {
      try {
        const res = await fetch('/api/me', { credentials: 'include' })

        if (res.status === 401) {
          if (!cancelled) {
            router.push('/login')
          }
          return
        }

        if (!res.ok) {
          throw new Error('Auth check failed')
        }

        const data = await res.json()
        if (!cancelled && data.user_id) {
          router.push('/dashboard')
        } else if (!cancelled) {
          router.push('/login')
        }
      } catch {
        if (!cancelled) {
          router.push('/login')
        }
      } finally {
        if (!cancelled) {
          setLoading(false)
        }
      }
    }

    checkAuth()

    return () => {
      cancelled = true
    }
  }, [router])

  if (loading) {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center p-4">
        <div className="w-full max-w-md space-y-6">
          <h1 className="text-3xl font-bold">Loading…</h1>
          <p className="text-muted-foreground">
            Checking your session…
          </p>
        </div>
      </div>
    )
  }

  return null
}
