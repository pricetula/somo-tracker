import { getAuthUser } from '@/lib/auth'
import { apiRequest, clearSession, SessionExpiredError } from '@/lib/api'

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

/**
 * Students list page — protected route inside (authorized) group.
 *
 * Fetches students from the BFF /api/students route, which proxies to
 * Go /api/v1/students with the session cookie attached. Go enforces
 * tenant scoping and guardian/teacher object-level access.
 *
 * On 401 (session revoked mid-session), clears the session and redirects
 * to /login.
 */
export default async function StudentsPage() {
  const user = await getAuthUser()

  if (!user) {
    return null
  }

  let students: Student[] = []
  let nextCursor: string | null = null

  try {
    const res = await apiRequest('/api/students?limit=50')
    const data: ListResponse = await res.json()
    students = data.items
    nextCursor = data.nextCursor ?? null
  } catch (err) {
    if (err instanceof SessionExpiredError) {
      // Session expired or revoked — clear cookie and redirect to login
      await clearSession()
    }
    console.error('Failed to load students:', err)
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
