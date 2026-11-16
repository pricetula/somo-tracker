import { getAuthUser } from '@/lib/auth'
import { clearSession } from '@/lib/api'

/**
 * Layout for all protected routes inside the (authorized) route group.
 *
 * Calls /api/v1/me once per render to validate the session and load the
 * user's role + active school. If the session is missing or invalid,
 * clears the stale session cookie and redirects to /login.
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
    // Session cookie may be present but invalid/expired — clear it
    // and redirect to /login to break any redirect loop with the proxy.
    await clearSession()
  }

  return <>{children}</>
}
