/**
 * Onboarding tenant name route - School name input
 * No cookie required, but redirects if cookie present
 */

import { createFileRoute } from '@tanstack/react-router';
import { OnboardingTenantName } from '@/features/auth/components/onboarding-tenant-name';

export const Route = createFileRoute('/onboarding/tenant-name')({
  component: OnboardingTenantNamePage,
});

function OnboardingTenantNamePage() {
  return (
    <div className="min-h-screen flex items-center justify-center bg-background px-4 py-12">
      <OnboardingTenantName />
    </div>
  );
}