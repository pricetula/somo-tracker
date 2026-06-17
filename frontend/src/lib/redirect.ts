/**
 * Open redirect guard for Next.js route handlers and client-side navigation.
 *
 * Next.js redirect() and router.push() are commonly abused in phishing attacks
 * when redirect targets are constructed from user-supplied query parameters
 * (e.g. /login?next=/ → manipulated to /login?next=https://evil.com).
 *
 * This whitelist-based redirect sanitiser must be used everywhere a redirect
 * target originates from user input (query params, form data, etc.).
 */

const ALLOWED_REDIRECT_PREFIXES = ["/dashboard", "/settings", "/tenants", "/register", "/login"];

/**
 * Sanitises a redirect target from user-supplied input.
 * Only allows relative paths on the allowlist. Returns / as the safe default
 * for anything that doesn't match.
 */
export function sanitiseRedirect(raw: string | null | undefined): string {
    if (!raw) return "/";

    // Reject anything that looks absolute (has a protocol or starts with //)
    if (/^https?:\/\//i.test(raw) || raw.startsWith("//")) return "/";

    // Reject anything with a newline (response splitting defence)
    if (raw.includes("\n") || raw.includes("\r")) return "/";

    const normalised = decodeURIComponent(raw).split("?")[0];

    // Allow root path (dashboard) explicitly
    if (normalised === "/") return raw;

    const isAllowed = ALLOWED_REDIRECT_PREFIXES.some((p) => normalised.startsWith(p));
    return isAllowed ? raw : "/";
}
