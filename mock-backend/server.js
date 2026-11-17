/**
 * Mock Go backend for PKCE magic-link e2e testing.
 *
 * Mimics the two auth handlers:
 *   POST /api/v1/auth/request-magic-link  -> sets pkce_verifier cookie, returns 200
 *   POST /api/v1/auth/authenticate-token  -> REQUIRES pkce_verifier cookie, else 400 MISSING_PKCE
 *
 * Run:  node server.js   (listens on :8080)
 * Frontend must set PUBLIC_API_URL=http://localhost:8080
 */

const express = require('express')
const cookieParser = require('cookie-parser')

const app = express()
app.use(cookieParser())
app.use(express.json())

// POST /api/v1/auth/request-magic-link
app.post('/api/v1/auth/request-magic-link', (req, res) => {
  const { email } = req.body
  if (!email) {
    return res.status(400).json({
      code: 'INVALID_REQUEST',
      message: 'invalid request body',
      errors: {},
    })
  }

  const pkceVerifier = require('crypto').randomBytes(32).toString('hex')

  res.cookie('pkce_verifier', pkceVerifier, {
    path: '/',
    httpOnly: true,
    secure: false, // localhost over http
    sameSite: 'lax',
    maxAge: 10 * 60 * 1000,
  })

  return res.status(200).json({ success: true })
})

// BFF path used by frontend
app.post('/api/auth/request-magic-link', (req, res) => {
  req.url = '/api/v1/auth/request-magic-link'
  app.handle(req, res)
})

// POST /api/v1/auth/authenticate-token
app.post('/api/v1/auth/authenticate-token', (req, res) => {
  const pkceVerifier = req.cookies['pkce_verifier']
  const { token } = req.body

  if (!token) {
    return res.status(400).json({
      code: 'INVALID_REQUEST',
      message: 'invalid request body',
      errors: {},
    })
  }
  if (!pkceVerifier) {
    return res.status(400).json({
      code: 'MISSING_PKCE',
      message: 'PKCE verifier missing, please restart login flow',
      errors: {},
      request_id: 'mock_req_' + Date.now(),
    })
  }

  // Clear pkce cookie like Go does after successful authenticate
  res.clearCookie('pkce_verifier', {
    path: '/',
    httpOnly: true,
    secure: false,
    sameSite: 'lax',
  })

  return res.status(200).json({
    ticket_token: 'mock_ticket_' + Date.now(),
    auth_state: 'authenticated',
    status: 'complete',
  })
})

// BFF path used by frontend
app.post('/api/auth/authenticate-token', (req, res) => {
  req.url = '/api/v1/auth/authenticate-token'
  app.handle(req, res)
})

const PORT = process.env.PORT || 8080
app.listen(PORT, () => {
  console.log(`Mock backend listening on http://localhost:${PORT}`)
})
