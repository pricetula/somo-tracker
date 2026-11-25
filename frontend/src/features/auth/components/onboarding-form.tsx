"use client";

import { FormProvider, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useOnboard } from "@/features/auth/hooks/use-onboard";
import { onboardSchema } from "@/features/auth/schemas/onboard.schema";
import type { OnboardFormValues } from "@/features/auth/schemas/onboard.schema";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { CountryCombobox, useCountries } from "@/features/countries";
import { EducationSystemCombobox, useEducationSystems } from "@/features/education-systems";

/**
 * Onboarding form — creates the user's organization and school.
 *
 * POST /api/auth/onboard → backend upgrades the temporary session to full.
 * Requires a valid temporary session cookie (set during OTP verification).
 * On success, the middleware redirects to /.
 *
 * Uses react-hook-form + zod for validation and TanStack Query for the API call.
 * Country and education system comboboxes live in their own feature modules.
 */
export default function OnboardingForm() {
  const onboardMutation = useOnboard();

  const form = useForm<OnboardFormValues>({
    resolver: zodResolver(onboardSchema),
    defaultValues: {
      organizationName: "",
      schoolName: "",
      userName: "",
      countryId: "",
      educationSystemId: "",
    },
    mode: "onChange",
  });

  const { data: countries, isPending: countriesPending } = useCountries();
  const { data: systems, isPending: systemsPending } = useEducationSystems();

  const handleSubmit = async (values: OnboardFormValues) => {
    await onboardMutation.mutateAsync(values);
  };

  return (
    <FormProvider {...form}>
      <div className="flex min-h-screen flex-col items-center justify-center p-4 sm:p-6">
        <div className="w-full max-w-md flex flex-col gap-6">
          <>
            <h1 className="text-3xl font-bold">Set up your organization</h1>
            <p className="text-muted-foreground text-sm">
              Welcome! Let&rsquo;s create your organization and school to get started.
            </p>
          </>

          <form
            onSubmit={form.handleSubmit(handleSubmit)}
            className="space-y-4"
          >
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

            <CountryCombobox
              name="countryId"
              label="Country"
              countries={countries ?? []}
              disabled={onboardMutation.isPending || countriesPending}
            />
            {form.formState.errors.countryId && (
              <p className="text-sm text-destructive">
                {form.formState.errors.countryId.message}
              </p>
            )}

            <EducationSystemCombobox
              name="educationSystemId"
              label="Education system"
              systems={systems ?? []}
              disabled={onboardMutation.isPending || systemsPending}
            />
            {form.formState.errors.educationSystemId && (
              <p className="text-sm text-destructive">
                {form.formState.errors.educationSystemId.message}
              </p>
            )}

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
    </FormProvider>
  );
}

export { OnboardingForm };
