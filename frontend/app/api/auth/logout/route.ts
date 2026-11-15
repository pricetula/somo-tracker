import { goRequest, forwardSetCookieHeaders, getSessionCookieName } from '@/lib/go-client'
import { cookies } from 'next/headers'
import { NextResponse } from 'next/server'

export const dynamic = 'force-dynamic'

export async function POST(req: Request): Promise<Response> {
  const result = await goRequest('/api/v1/auth/logout', {
    method: 'POST',
    forwardCookies: true,
  })

  const res = NextResponse.json(result.data, { status: result.status })

  // Forward Set-Cookie from Go (clears session on backend)
  forwardSetCookieHeaders(res, result.setCookieHeaders)

  // Also clear the cookie on the Next.js domain (browser boundary)
  const cookieStore = await cookies()
  const sessionName = getSessionCookieName()
  if (cookieStore.get(sessionName)) {
    res.cookies.set({
      name: sessionName,
      value: '',
      path: '/',
      httpOnly: true,
      secure: process.env.NODE_ENV === 'production',
      sameSite: 'lax',
      maxAge: 0,
    })
  }

  return res
}
