'use client'

import { useEffect } from 'react'
import { useRouter } from 'next/navigation'

/**
 * Root page — redirects to /login when unauthenticated.
 */
export default function RootPage() {
  const router = useRouter()

  useEffect(() => {
    router.replace('/login')
  }, [router])

  return null
}
