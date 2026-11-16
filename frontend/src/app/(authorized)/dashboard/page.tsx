import { getAuthUser } from '@/lib/auth'

/**
 * Dashboard page — protected route inside (authorized) group.
 *
 * The layout already gates this page (redirects to /login if no session),
 * but we call getAuthUser() here too to load user data for the UI.
 * React cache() deduplicates the /api/v1/me call with the layout's call.
 */
export default async function DashboardPage() {
  const user = await getAuthUser()

  if (!user) {
    // Should not happen — layout already gated — but fail closed anyway.
    return null
  }

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold">Dashboard</h1>
      <p className="mt-2 text-muted-foreground">
        Welcome back, {user.fullName}
      </p>

      <div className="mt-6 grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="border rounded-lg p-4">
          <h2 className="text-sm font-medium text-muted-foreground">Role</h2>
          <p className="mt-1">{user.role}</p>
        </div>
        <div className="border rounded-lg p-4">
          <h2 className="text-sm font-medium text-muted-foreground">
            Active school
          </h2>
          <p className="mt-1">
            {user.activeSchool?.name ?? 'None'}
          </p>
        </div>
        <div className="border rounded-lg p-4">
          <h2 className="text-sm font-medium text-muted-foreground">
            Available schools
          </h2>
          <p className="mt-1">{user.schools.length}</p>
        </div>
      </div>
    </div>
  )
}
