import { apiRequest } from '@/lib/api-client'

export const dynamic = 'force-dynamic'

export async function GET(req: Request) {
  return apiRequest('/api/v1/education/systems', { method: 'GET' }, req)
}
