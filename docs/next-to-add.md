Doppler vars to add

Recommended names for production. Add the same vars per env, only values change.

```
  # Origins / CORS
  CORS_ORIGINS=https://app.somotracker.com,http://localhost:3000

  # Cookie sharing between api.somotracker.com and app.somotracker.com
  COOKIE_DOMAIN=.somotracker.com

  # CSP Report-Only rollout
  CSP_REPORT_ONLY_ENABLED=true
  CSP_REPORT_ONLY_URI=https://api.somotracker.com/csp-report
  CSP_REPORT_ONLY_POLICY=default-src 'self'; script-src 'self'; connect-src 'self'
https://api.somotracker.com; img-src 'self' data:; style-src 'self' 'unsafe-inline'; frame-ancestors
'none'

  # Existing security baseline vars you already have
  APP_ENV=production
  STYTCH_ENV=live
  EMAIL_HMAC_KEY=<your key>
```

Good-judgement defaults:

- CORS_ORIGINS drives both Fiber CORS and CSRF origin validation. Keep http://localhost:3000 for local
  dev, remove it in production via Doppler env.
- COOKIE_DOMAIN=.somotracker.com makes the __Secure-session_token cookie visible to both
  api.somotracker.com and app.somotracker.com. Leave empty in development to keep host-only cookies.
- CSP is Report-Only first. Once you’ve reviewed reports at CSP_REPORT_ONLY_URI, flip to enforcing by
  removing the Report-Only header and adding a normal Content-Security-Policy header. The policy above is
  conservative for an API – it blocks inline scripts and frames.

Multi-tenant isolation integration tests are still missing per spec AC-11. The contract test
TestContract_CrossTenant_Returns404NotFound exists but it’s a simulation only. Let me know if you want me
to scaffold the real integration test with testcontainers + tenant fixtures next.
