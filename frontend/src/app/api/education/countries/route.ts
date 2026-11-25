import { apiRequest } from '@/lib/api-client'

export const dynamic = 'force-dynamic'

export async function GET() {
  return apiRequest('/api/v1/education/countries', { method: 'GET' })
}
