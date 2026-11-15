import { goRequest, forwardSetCookieHeaders } from '@/lib/go-client'

export const dynamic = 'force-dynamic'

export async function POST(req: Request): Promise<Response> {
  const body = await req.json()

  const result = await goRequest('/api/v1/auth/request-magic-link', {
    method: 'POST',
    body,
  })

  const res = NextResponse.json(result.data, { status: result.status })
  forwardSetCookieHeaders(res, result.setCookieHeaders)
  return res
}
