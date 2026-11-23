/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  async rewrites() {
    return []
  },
  async headers() {
    return [
      {
        source: '/:path*',
        headers: [
          {
            key: 'Content-Security-Policy',
            value: [
              "default-src 'self'",
              "script-src 'self' 'unsafe-eval' 'unsafe-inline' https://challenges.cloudflare.com https://*.cloudflare.com",
              "connect-src 'self' data: https://challenges.cloudflare.com https://hagen.challenges.cloudflare.com https://brunhild.challenges.cloudflare.com https://*.alg13.dnstest.challenges.cloudflare.com https://*.mldsa44.dnstest.challenges.cloudflare.com https://dnstest.dev https://ca2.cloudflarecaptcha.com https://ca1.cloudflarecaptcha.com",
              "frame-src 'self' https://challenges.cloudflare.com https://*.cloudflare.com",
              "style-src 'self' 'unsafe-inline'",
              "img-src 'self' data:",
            ].join('; '),
          },
        ],
      },
    ]
  },
}

export default nextConfig
