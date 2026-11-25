import OnboardingForm from "@/features/auth/components/onboarding-form";

export const dynamic = "force-dynamic";

/**
 * Onboarding page — shown after OTP verification when no account exists.
 *
 * Delegates rendering to OnboardingForm. The backend upgrades the temporary
 * session to full on success; the middleware then redirects to /.
 */
export default function OnboardingPage() {
  return <OnboardingForm />;
}
