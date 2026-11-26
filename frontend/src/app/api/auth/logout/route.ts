import { apiRequest } from '@/lib/api-client'

export const dynamic = 'force-dynamic'

export async function POST(req: Request) {
  return apiRequest('/api/v1/auth/logout', {
    method: 'POST',
    body: await req.arrayBuffer(),
    headers: {
      'Content-Type': req.headers.get('content-type') ?? 'application/json',
    },
  }, req)
}
