'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'

interface Student {
  student_id: string
  admission_number: string
  full_name: string
  date_of_birth: string
  gender: string
}

interface ListResponse {
  items: Student[]
  nextCursor: string | null
}

interface AuthUser {
  userId: string
  fullName: string
  email: string
  role: string
  activeSchool?: { id: string; name: string }
  schools: Array<{ id: string; name: string }>
}

/**
 * Students list page — protected route inside (authorized) group.
 *
 * Client component that fetches the user profile from /api/me and students
 * from /api/students (BFF → Go /api/v1/students). The BFF forwards the
 * HttpOnly session cookie server-side, so the client uses credentials:
 * 'include' on relative fetch paths.
 *
 * On 401 (session revoked mid-session), clears the session and redirects
 * to /login.
 */
export default function StudentsPage() {
  const router = useRouter()
  const [user, setUser] = useState<AuthUser | null>(null)
  const [students, setStudents] = useState<Student[]>([])
  const [nextCursor, setNextCursor] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false

    async function doLogout() {
      // Call logout endpoint to clear the session cookie, then redirect.
      await fetch('/api/auth/logout', { method: 'GET', credentials: 'include' })
      if (!cancelled) {
        router.push('/login')
      }
    }

    async function load() {
      try {
        // Load user profile first
        const meRes = await fetch('/api/me', { credentials: 'include' })

        if (meRes.status === 401) {
          if (!cancelled) {
            await doLogout()
          }
          return
        }

        if (!meRes.ok) {
          throw new Error('Failed to load user profile')
        }

        const data = await meRes.json()
        const authUser: AuthUser = {
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
        }
        if (!cancelled) {
          setUser(authUser)
        }

        // Load students
        const studentsRes = await fetch(
          '/api/students?limit=50',
          { credentials: 'include' }
        )

        if (studentsRes.status === 401) {
          if (!cancelled) {
            await doLogout()
          }
          return
        }

        if (!studentsRes.ok) {
          throw new Error('Failed to load students')
        }

        const studentsData: ListResponse = await studentsRes.json()
        if (!cancelled) {
          setStudents(studentsData.items)
          setNextCursor(studentsData.nextCursor ?? null)
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : 'Failed to load')
          console.error('Failed to load students:', err)
        }
      } finally {
        if (!cancelled) {
          setLoading(false)
        }
      }
    }

    load()

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
            Please wait while we load your students.
          </p>
        </div>
      </div>
    )
  }

  if (!user) {
    return null
  }

  if (error) {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center p-4">
        <div className="w-full max-w-md space-y-6">
          <h1 className="text-3xl font-bold">Failed to load</h1>
          <p className="mt-2 text-muted-foreground">{error}</p>
        </div>
      </div>
    )
  }

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold">Students</h1>
      <p className="mt-2 text-muted-foreground">
        Students in {user.activeSchool?.name ?? 'your school'}
      </p>

      {students.length === 0 ? (
        <p className="mt-4">No students found.</p>
      ) : (
        <div className="mt-4 border rounded-lg overflow-hidden">
          <table className="w-full text-left">
            <thead>
              <tr className="border-b">
                <th className="p-3 font-medium">Admission #</th>
                <th className="p-3 font-medium">Name</th>
                <th className="p-3 font-medium">DOB</th>
                <th className="p-3 font-medium">Gender</th>
              </tr>
            </thead>
            <tbody>
              {students.map((s) => (
                <tr key={s.student_id} className="border-b">
                  <td className="p-3">{s.admission_number}</td>
                  <td className="p-3">{s.full_name}</td>
                  <td className="p-3">{s.date_of_birth}</td>
                  <td className="p-3">{s.gender}</td>
                </tr>
              ))}
            </tbody>
          </table>
          {nextCursor && (
            <p className="mt-2 text-xs text-muted-foreground">
              More results available (cursor: {nextCursor.slice(0, 8)}…)
            </p>
          )}
        </div>
      )}
    </div>
  )
}
