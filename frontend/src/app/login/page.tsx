import LoginForm from "@/components/login-form";

/**
 * Login page server component.
 *
 * Reads `TURNSTILE_SITE_KEY` from the server environment (injected by Doppler
 * at runtime on Vercel) and passes it to the client form so the Turnstile
 * widget can render. The secret key is never exposed to the browser — it is
 * used only in the API route for server-side verification.
 */
export default function LoginPage() {
  const siteKey = process.env.TURNSTILE_SITE_KEY;
  return <LoginForm siteKey={siteKey} />;
}
