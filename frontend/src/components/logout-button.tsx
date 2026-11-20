'use client'

import { useRouter } from 'next/navigation'

export function LogoutButton() {
  const router = useRouter()

  const handleLogout = async () => {
    const res = await fetch('/api/auth/logout', {
      method: 'POST',
      credentials: 'include',
    })

    if (res.ok) {
      router.push('/login')
    }
  }

  return (
    <button
      onClick={handleLogout}
      className="inline-flex items-center justify-center rounded-md border border-input bg-background px-4 py-2 text-sm font-medium text-muted-foreground shadow-sm hover:bg-accent hover:text-accent-foreground disabled:opacity-50"
    >
      Logout
    </button>
  )
}
