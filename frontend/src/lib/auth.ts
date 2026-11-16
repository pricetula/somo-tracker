import { cache } from 'react'
import { goRequest } from './go-client'

export interface AuthUser {
  userId: string
  fullName: string
  email: string
  role: string
  activeSchool?: { id: string; name: string }
  schools: Array<{ id: string; name: string }>
}

/**
 * getAuthUser validates the session against Go /api/v1/me and returns the
 * user or null.
 *
 * Results are deduplicated within a single render pass via React cache(),
 * so calling this from both the layout and a page only hits the backend once.
 */
export const getAuthUser = cache(async (): Promise<AuthUser | null> => {
  try {
    const res = await goRequest('/api/v1/me')
    if (!res.ok) return null

    const data = await res.json()
    return {
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
  } catch {
    return null
  }
})
