/**
 * Onboarding tenant name step - School name + country + education system
 * Guards access by checking onboarding session validity on mount
 */

'use client';

import { useEffect, useState } from 'react';
import { useRouter } from '@tanstack/react-router';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Separator } from '@/components/ui/separator';
import { Loader2, Building2, ArrowRight } from 'lucide-react';
import { tenantNameFormSchema, type TenantNameFormInput } from '../schemas';
import { useCompleteOnboarding } from '../hooks/use-onboarding';
import { useOnboardingCurrent } from '../hooks/use-onboarding';
import { isApiError } from '@/lib/api/errors';
import { toast } from 'sonner';
import { CountryCombobox } from '@/features/country';
import { EducationSystemCombobox } from '@/features/onboarding/education-system-combobox';

export function OnboardingTenantName() {
  const router = useRouter();
  const { mutateAsync: completeOnboarding, isPending } = useCompleteOnboarding();
  const { data, isError, isLoading } = useOnboardingCurrent();

  const [countryId, setCountryId] = useState<string | null>(null);
  const [educationSystemId, setEducationSystemId] = useState<string | null>(null);

  // Guard: redirect to login if onboarding session is invalid or expired
  useEffect(() => {
    if (!isLoading && (isError || !data)) {
      toast.error('Onboarding session expired, please sign in again');
      router.navigate({ to: '/login' });
    }
  }, [isLoading, isError, data, router]);

  const form = useForm<TenantNameFormInput>({
    resolver: zodResolver(tenantNameFormSchema),
    defaultValues: { school_name: '' },
  });

  const onSubmit = async (data: TenantNameFormInput) => {
    const idempotencyKey = `onboarding-${Date.now()}-${Math.random().toString(36).slice(2, 11)}`;
    if (!countryId) {
      toast.error('Please select a country');
      return;
    }
    if (!educationSystemId) {
      toast.error('Please select an education system');
      return;
    }
    try {
      await completeOnboarding({
        school_name: data.school_name,
        idempotency_key: idempotencyKey,
        country_id: countryId,
        education_system_id: educationSystemId,
      });
    } catch (err) {
      if (isApiError(err) && err.fieldErrors) {
        // Map server field errors to react-hook-form
        for (const [field, errs] of Object.entries(err.fieldErrors)) {
          form.setError(field as keyof TenantNameFormInput, {
            type: 'server',
            message: errs[0]?.message ?? 'Invalid value',
          });
        }
        // Show generic form error if present
        if (err.fieldErrors._form) {
          form.setError('school_name', { type: 'server', message: err.fieldErrors._form[0]?.message });
        }
      }
      // toast is handled by hook for non-field errors
    }
  };

  const isLoadingCountries = isLoading || isPending;
  const isLoadingEducationSystems = !countryId || isLoading || isPending;

  return (
    <Card className="w-full max-w-md mx-auto mt-20">
      <CardHeader className="space-y-4">
        <div className="mx-auto h-12 w-12 rounded-lg bg-primary/10 flex items-center justify-center">
          <Building2 className="h-6 w-6 text-primary" />
        </div>
        <CardTitle className="text-2xl text-center">Name your school</CardTitle>
        <CardDescription className="text-center">
          Select your country and education system, then name your school
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="country-combobox">Country</Label>
            <CountryCombobox
              value={countryId}
              onChange={setCountryId}
              disabled={isLoadingCountries}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="education-system-combobox">Education System</Label>
            <EducationSystemCombobox
              value={educationSystemId}
              countryId={countryId}
              onChange={setEducationSystemId}
              disabled={isLoadingEducationSystems}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="school_name">School Name</Label>
            <Input
              id="school_name"
              type="text"
              placeholder="e.g., Sunshine Primary School"
              autoComplete="organization"
              {...form.register('school_name')}
              disabled={isPending || !countryId || !educationSystemId}
              aria-describedby={form.formState.errors.school_name ? 'school_name-error' : undefined}
            />
            {form.formState.errors.school_name && (
              <p id="school_name-error" className="text-sm text-destructive" role="alert">
                {form.formState.errors.school_name.message}
              </p>
            )}
          </div>

          <Button
            type="submit"
            className="w-full"
            disabled={isPending || isLoading || !countryId || !educationSystemId}
            size="default"
          >
            {isPending ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Creating your school...
              </>
            ) : (
              <>
                Create School
                <ArrowRight className="ml-2 h-4 w-4" />
              </>
            )}
          </Button>
        </form>

        <Separator className="my-6" />

        <p className="text-center text-sm text-muted-foreground">
          By creating a school, you agree to our{' '}
          <a href="/terms" className="underline hover:text-foreground">
            Terms of Service
          </a>{' '}
          and{' '}
          <a href="/privacy" className="underline hover:text-foreground">
            Privacy Policy
          </a>
        </p>
      </CardContent>
    </Card>
  );
}
