import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'

export default function DashboardPage() {
  const [user, setUser] = useState<any>(null)
  const [loading, setLoading] = useState(true)
  const router = useRouter()

  useEffect(() => {
    fetch('/api/me')
      .then(async (res) => {
        if (res.status === 401) {
          router.push('/login')
          return
        }
        const data = await res.json()
        setUser(data)
        setLoading(false)
      })
      .catch(() => {
        router.push('/login')
        setLoading(false)
      })
  }, [router])

  if (loading) {
    return (
      <div className="flex flex-col items-center justify-center min-h-[400px]">
        <div className="animate-pulse rounded-lg bg-muted w-96 h-32" />
      </div>
    )
  }

  if (!user) {
    return null
  }

  return (
    <div className="p-6">
      <div className="flex flex-col space-y-1 pb-6">
        <h1 className="text-2xl font-bold">Dashboard</h1>
        <p className="text-sm text-muted-foreground">
          Welcome back, {user.full_name}
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div className="p-4 bg-muted rounded-lg">
          <p className="text-xs text-muted-foreground uppercase tracking-wider">User ID</p>
          <p className="font-mono text-sm">{user.user_id}</p>
        </div>
        <div className="p-4 bg-muted rounded-lg">
          <p className="text-xs text-muted-foreground uppercase tracking-wider">Role</p>
          <p className="text-sm">{user.role}</p>
        </div>
        <div className="p-4 bg-muted rounded-lg">
          <p className="text-xs text-muted-foreground uppercase tracking-wider">Active School</p>
          <p className="text-sm">{user.active_school?.name || 'None'}</p>
        </div>
      </div>

      <div className="mt-8">
        <h2 className="text-lg font-bold mb-4">Your Schools</h2>
        {user.schools && user.schools.length > 0 ? (
          <ul className="space-y-2">
            {user.schools.map((school: { id: string; name: string }) => (
              <li key={school.id} className="flex items-center justify-between p-2 bg-background/50 rounded border border-border">
                <span>{school.name}</span>
                <span className="text-xs text-muted-foreground">{school.id}</span>
              </li>
            ))}
          </ul>
        ) : (
          <p className="text-muted-foreground">No schools found.</p>
        )}
      </div>

      <div className="mt-8">
        <button
          onClick={() => fetch('/api/auth/logout', { method: 'POST' }).then(() => router.push('/login'))}
          className="inline-flex items-center gap-2 px-3 py-1.5 text-xs font-medium rounded-md border border-input bg-input/50 hover:bg-input/80 text-foreground"
        >
          Sign out
        </button>
      </div>
    </div>
  )
}
