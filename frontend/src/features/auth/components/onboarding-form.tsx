"use client";

import { useCallback } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useOnboard } from "@/features/auth/hooks/use-onboard";
import { onboardSchema } from "@/features/auth/schemas/onboard.schema";
import type { OnboardFormValues } from "@/features/auth/schemas/onboard.schema";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

/**
 * Onboarding form — creates the user's organization and school.
 *
 * POST /api/auth/onboard → backend upgrades the temporary session to full.
 * Requires a valid temporary session cookie (set during OTP verification).
 * On success, the middleware redirects to /.
 *
 * Uses react-hook-form + zod for validation and TanStack Query for the API call.
 */
export default function OnboardingForm() {
  const onboardMutation = useOnboard();

  const form = useForm<OnboardFormValues>({
    resolver: zodResolver(onboardSchema),
    defaultValues: {
      organizationName: "",
      schoolName: "",
      userName: "",
    },
    mode: "onChange",
  });

  const handleSubmit = useCallback(
    async (values: OnboardFormValues) => {
      await onboardMutation.mutateAsync(values);
    },
    [onboardMutation]
  );

  return (
    <div className="flex min-h-screen flex-col items-center justify-center p-4 sm:p-6">
      <div className="w-full max-w-md space-y-6">
        <div className="space-y-2">
          <h1 className="text-3xl font-bold">Set up your organization</h1>
          <p className="text-muted-foreground text-sm">
            Welcome! Let&rsquo;s create your organization and school to get started.
          </p>
        </div>

        <form
          onSubmit={form.handleSubmit(handleSubmit)}
          className="space-y-4"
        >
          <div className="space-y-2">
            <Label htmlFor="organization_name">Organization name</Label>
            <Input
              id="organization_name"
              type="text"
              placeholder="e.g. Acme School District"
              {...form.register("organizationName")}
              disabled={onboardMutation.isPending}
            />
            {form.formState.errors.organizationName && (
              <p className="text-sm text-destructive">
                {form.formState.errors.organizationName.message}
              </p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="school_name">School name</Label>
            <Input
              id="school_name"
              type="text"
              placeholder="e.g. Lincoln High School"
              {...form.register("schoolName")}
              disabled={onboardMutation.isPending}
            />
            {form.formState.errors.schoolName && (
              <p className="text-sm text-destructive">
                {form.formState.errors.schoolName.message}
              </p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="user_name">Your name</Label>
            <Input
              id="user_name"
              type="text"
              placeholder="e.g. Jane Doe"
              {...form.register("userName")}
              disabled={onboardMutation.isPending}
            />
            {form.formState.errors.userName && (
              <p className="text-sm text-destructive">
                {form.formState.errors.userName.message}
              </p>
            )}
          </div>

          {onboardMutation.error && (
            <p className="text-sm text-destructive">
              {onboardMutation.error.message}
            </p>
          )}

          <Button
            type="submit"
            disabled={onboardMutation.isPending}
            className="w-full"
          >
            {onboardMutation.isPending ? "Setting up…" : "Create organization & school"}
          </Button>
        </form>
      </div>
    </div>
  );
}

export { OnboardingForm };
