import { goRequest } from '@/lib/go-client'
import { NextResponse } from 'next/server'

export const dynamic = 'force-dynamic'

export async function GET(): Promise<Response> {
  const result = await goRequest('/api/v1/me', {
    method: 'GET',
    forwardCookies: true,
  })

  return NextResponse.json(result.data, { status: result.status })
}
