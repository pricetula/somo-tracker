import { goRequest } from '@/lib/go-client'

export const dynamic = 'force-dynamic'

/**
 * BFF GET /api/students — proxies to Go /api/v1/students.
 * Forwards query params (limit, cursor, sort) and the session cookie.
 */
export async function GET(request: Request) {
  const { searchParams } = new URL(request.url)
  const query = searchParams.toString()
  const path = `/api/v1/students${query ? `?${query}` : ''}`

  return goRequest(path)
}
