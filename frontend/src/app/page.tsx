import { getAuthUser } from '@/lib/auth'
import { clearSession } from '@/lib/api'
import { redirect } from 'next/navigation'

/**
 * Root page — redirects based on authentication state.
 *
 * Authenticated users → /dashboard
 * Unauthenticated users → /login
 */
export default async function RootPage() {
  const user = await getAuthUser()

  if (user) {
    redirect('/dashboard')
  }

  // Session may be present but invalid — clear it before redirecting
  await clearSession()
}
