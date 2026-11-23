import LoginForm from "@/components/login-form";

export const dynamic = "force-dynamic";

/**
 * Login page server component.
 *
 * Turnstile site key is hardcoded in the Turnstile component itself.
 */
export default function LoginPage() {
  return <LoginForm />;
}
