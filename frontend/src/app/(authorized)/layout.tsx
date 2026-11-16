import { getAuthUser } from '@/lib/auth'
import { redirect } from 'next/navigation'

/**
 * Layout for all protected routes inside the (authorized) route group.
 *
 * Calls /api/v1/me once per render to validate the session and load the
 * user's role + active school. If the session is missing or invalid,
 * redirects to /login.
 *
 * This is an optimistic UX gate. The real enforcement happens in the Go
 * backend on every API request (AuthMiddleware + authz.Require).
 */
export default async function AuthorizedLayout({
  children,
}: Readonly<{
  children: React.ReactNode
}>) {
  const user = await getAuthUser()

  if (!user) {
    redirect('/login')
  }

  return <>{children}</>
}
