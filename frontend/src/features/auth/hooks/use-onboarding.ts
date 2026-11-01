/**
 * Hooks for onboarding flow
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { getOnboardingCurrent } from '../api/get-onboarding';
import { updateOnboarding } from '../api/update-onboarding';
import { completeOnboarding } from '../api/complete-onboarding';
import { authQueryKeys, authMutationKeys } from '../query-keys';
import { updateOnboardingSchema, completeOnboardingSchema } from '../schemas';
import { toast } from 'sonner';
import { isApiError } from '@/lib/api/errors';
import { useRouter } from '@tanstack/react-router';

export function useOnboardingCurrent() {
  return useQuery({
    queryKey: authQueryKeys.onboardingCurrent(),
    queryFn: getOnboardingCurrent,
    staleTime: 0,
    gcTime: 0,
  });
}

export function useUpdateOnboarding() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationKey: authMutationKeys.updateOnboarding(),
    mutationFn: async (data: unknown) => {
      const parsed = updateOnboardingSchema.safeParse(data);
      if (!parsed.success) {
        throw new Error(parsed.error.issues[0].message);
      }
      return updateOnboarding(parsed.data);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: authQueryKeys.onboardingCurrent() });
      toast.success('Progress saved');
    },
    onError: (error: unknown) => {
      if (isApiError(error)) {
        const extra = error.requestId ? ` Ref: ${error.requestId}` : '';
        toast.error((error.fieldErrors && Object.keys(error.fieldErrors).length ? 'Please fix the highlighted fields' : error.message) + extra);
        return;
      }
      toast.error((error as Error)?.message || 'Failed to save progress');
    },
  });
}

export function useCompleteOnboarding() {
  const queryClient = useQueryClient();
  const router = useRouter();

  return useMutation({
    mutationKey: authMutationKeys.completeOnboarding(),
    mutationFn: async (data: unknown) => {
      const parsed = completeOnboardingSchema.safeParse(data);
      if (!parsed.success) {
        throw new Error(parsed.error.issues[0].message);
      }
      return completeOnboarding(parsed.data);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: authQueryKeys.all });
      toast.success('Onboarding complete! Welcome to Somotracker.');
      // Redirect to home/dashboard after successful provisioning
      router.navigate({ to: '/' });
    },
    onError: (error: unknown) => {
      if (isApiError(error)) {
        const msg = error.fieldErrors && Object.keys(error.fieldErrors).length > 0
          ? 'Please fix the highlighted fields'
          : error.message;
        const extra = error.requestId ? ` Ref: ${error.requestId}` : '';
        toast.error(msg + extra);
        return;
      }
      toast.error((error as Error)?.message || 'Failed to complete onboarding');
    },
  });
}
