/**
 * Mock Go backend for magic-link e2e testing.
 *
 * Mimics the two auth handlers:
 *   POST /api/v1/auth/request-magic-link  -> returns 200
 *   POST /api/v1/auth/authenticate-token  -> returns 200 with a mock ticket
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

  return res.status(200).json({ success: true })
})

// BFF path used by frontend
app.post('/api/auth/request-magic-link', (req, res) => {
  req.url = '/api/v1/auth/request-magic-link'
  app.handle(req, res)
})

// POST /api/v1/auth/authenticate-token
app.post('/api/v1/auth/authenticate-token', (req, res) => {
  const { token } = req.body

  if (!token) {
    return res.status(400).json({
      code: 'INVALID_REQUEST',
      message: 'invalid request body',
      errors: {},
    })
  }

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
