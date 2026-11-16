import { goRequest } from '@/lib/go-client'

export const dynamic = 'force-dynamic'

export async function GET() {
  return goRequest('/api/v1/me')
}
