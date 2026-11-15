import { goRequest, forwardSetCookieHeaders } from '@/lib/go-client'
import { NextResponse } from 'next/server'

export const dynamic = 'force-dynamic'

export async function POST(req: Request): Promise<Response> {
  const body = await req.json()

  const result = await goRequest('/api/v1/onboarding/complete', {
    method: 'POST',
    body,
    forwardCookies: true,
  })

  const res = NextResponse.json(result.data, { status: result.status })
  forwardSetCookieHeaders(res, result.setCookieHeaders)
  return res
}
