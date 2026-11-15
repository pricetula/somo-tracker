import { goRequest, forwardSetCookieHeaders } from '@/lib/go-client'
import { NextResponse } from 'next/server'

export const dynamic = 'force-dynamic'

export async function GET(): Promise<Response> {
  const result = await goRequest('/api/v1/onboarding/current', {
    method: 'GET',
    forwardCookies: true,
  })

  const res = NextResponse.json(result.data, { status: result.status })
  forwardSetCookieHeaders(res, result.setCookieHeaders)
  return res
}

export async function PUT(req: Request): Promise<Response> {
  const body = await req.json()

  const result = await goRequest('/api/v1/onboarding/current', {
    method: 'PUT',
    body,
    forwardCookies: true,
  })

  const res = NextResponse.json(result.data, { status: result.status })
  forwardSetCookieHeaders(res, result.setCookieHeaders)
  return res
}
