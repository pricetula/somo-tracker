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
 * Layout for all protected routes inside the (authorized) route group.
 *
 * Client component that fetches /api/me on mount to validate the session.
 * If the session is missing or invalid, clears the stale session cookie
 * and redirects to /login.
 *
 * This is an optimistic UX gate. The real enforcement happens in the Go
 * backend on every API request (AuthMiddleware + authz.Require).
 */
export default function AuthorizedLayout({
  children,
}: Readonly<{
  children: React.ReactNode
}>) {
  const router = useRouter()
  const [user, setUser] = useState<AuthUser | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let cancelled = false

    async function checkAuth() {
      async function doLogout() {
        // Call logout endpoint to clear the session cookie, then redirect.
        // fetch processes Set-Cookie; router.push handles navigation.
        await fetch('/api/auth/logout', { method: 'GET', credentials: 'include' })
        if (!cancelled) {
          router.push('/login')
        }
      }

      try {
        const res = await fetch('/api/me', { credentials: 'include' })

        if (res.status === 401) {
          if (!cancelled) {
            await doLogout()
          }
          return
        }

        if (!res.ok) {
          throw new Error('Auth check failed')
        }

        const data = await res.json()
        if (!cancelled) {
          setUser({
            userId: data.user_id,
            fullName: data.full_name,
            email: data.email,
            role: data.role,
            activeSchool: data.active_school
              ? { id: data.active_school.id, name: data.active_school.name }
              : undefined,
            schools: (data.schools ?? []).map((s: { id: string; name: string }) => ({
              id: s.id,
              name: s.name,
            })),
          })
        }
      } catch {
        if (!cancelled) {
          await doLogout()
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

  if (!user) {
    return null
  }

  return <>{children}</>
}
