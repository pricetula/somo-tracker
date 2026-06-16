// SECURITY RULES — DO NOT VIOLATE:
// 1. NEXT_PUBLIC_ variables must contain ZERO secrets.
//    Acceptable: API base URL, feature flags, analytics IDs.
//    Never: session secrets, DB URLs, API keys, service account credentials.
// 2. The backend DATABASE_URL and REDIS_URL must never appear in this file.
// 3. Internal session secrets or signing keys must never appear in this file under any name.
// 4. Server-only secrets go in .env.local (gitignored) and are accessed
//    only inside Server Components, Route Handlers, or getServerSideProps.
// 5. Run `npx @next/codemod` or manually audit: grep -r "NEXT_PUBLIC_" ./
//    and verify each result contains no credentials.

import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  async rewrites() {
    return [
      {
        source: "/api/:path*",
        destination: `${process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:3030"}/api/:path*`,
      },
    ];
  },
  async headers() {
    return [
      {
        source: "/(.*)",
        headers: [
          {
            key: "Content-Security-Policy",
            value: [
              "default-src 'self'",
              "script-src 'self' 'unsafe-eval' 'unsafe-inline'",
              "style-src 'self' 'unsafe-inline'",
              "img-src 'self' data: blob:",
              "font-src 'self'",
              `connect-src 'self' ${process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:3030"}`,
              "frame-ancestors 'none'",
              "base-uri 'self'",
              "form-action 'self'",
            ].join("; "),
          },
          { key: "X-Frame-Options", value: "DENY" },
          { key: "X-Content-Type-Options", value: "nosniff" },
          {
            key: "Referrer-Policy",
            value: "strict-origin-when-cross-origin",
          },
          {
            key: "Permissions-Policy",
            value: "camera=(), microphone=(), geolocation=()",
          },
        ],
      },
    ];
  },
};

export default nextConfig;
