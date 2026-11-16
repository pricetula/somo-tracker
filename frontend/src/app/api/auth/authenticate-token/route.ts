import { goRequest } from '@/lib/go-client'

export const dynamic = 'force-dynamic'

export async function POST(req: Request) {
  return goRequest('/api/v1/auth/authenticate-token', {
    method: 'POST',
    body: await req.arrayBuffer(),
    headers: {
      'Content-Type': req.headers.get('content-type') ?? 'application/json',
    },
  })
}
