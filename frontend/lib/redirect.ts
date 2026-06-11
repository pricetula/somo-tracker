/**
 * Open redirect guard for Next.js route handlers and client-side navigation.
 *
 * Next.js redirect() and router.push() are commonly abused in phishing attacks
 * when redirect targets are constructed from user-supplied query parameters
 * (e.g. /login?next=/dashboard → manipulated to /login?next=https://evil.com).
 *
 * This whitelist-based redirect sanitiser must be used everywhere a redirect
 * target originates from user input (query params, form data, etc.).
 */

const ALLOWED_REDIRECT_PREFIXES = [
  "/dashboard",
  "/settings",
  "/tenants",
];

/**
 * Sanitises a redirect target from user-supplied input.
 * Only allows relative paths on the allowlist. Returns /dashboard as the
 * safe default for anything that doesn't match.
 */
export function sanitiseRedirect(raw: string | null | undefined): string {
  if (!raw) return "/dashboard";

  // Reject anything that looks absolute (has a protocol or starts with //)
  if (/^https?:\/\//i.test(raw) || raw.startsWith("//")) return "/dashboard";

  // Reject anything with a newline (response splitting defence)
  if (raw.includes("\n") || raw.includes("\r")) return "/dashboard";

  const normalised = decodeURIComponent(raw).split("?")[0];
  const isAllowed = ALLOWED_REDIRECT_PREFIXES.some((p) =>
    normalised.startsWith(p),
  );
  return isAllowed ? raw : "/dashboard";
}
