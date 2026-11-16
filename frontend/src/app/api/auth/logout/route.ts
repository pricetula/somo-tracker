import { goRequest } from '@/lib/go-client'

export const dynamic = 'force-dynamic'

export async function POST() {
  return goRequest('/api/v1/auth/logout', { method: 'POST' })
}
