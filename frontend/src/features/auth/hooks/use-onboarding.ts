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
import { useActiveSchool } from '@/lib/hooks/use-active-school';
import { isApiError } from '@/lib/api/errors';

export function useOnboardingCurrent() {
  const { schoolId } = useActiveSchool();
  return useQuery({
    queryKey: schoolId ? authQueryKeys.onboardingCurrent(schoolId) : ['auth', 'onboarding', 'no-school'],
    queryFn: getOnboardingCurrent,
    enabled: !!schoolId,
    staleTime: 0,
    gcTime: 0,
  });
}

export function useUpdateOnboarding() {
  const { schoolId } = useActiveSchool();
  const queryClient = useQueryClient();

  return useMutation({
    mutationKey: schoolId ? authMutationKeys.updateOnboarding(schoolId) : ['auth', 'update-onboarding', 'no-school'],
    mutationFn: async (data: unknown) => {
      if (!schoolId) throw new Error('No active school');
      const parsed = updateOnboardingSchema.safeParse(data);
      if (!parsed.success) {
        throw new Error(parsed.error.issues[0].message);
      }
      return updateOnboarding(parsed.data);
    },
    onSuccess: () => {
      if (schoolId) {
        queryClient.invalidateQueries({ queryKey: authQueryKeys.onboardingCurrent(schoolId) });
      }
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
  const { schoolId } = useActiveSchool();
  const queryClient = useQueryClient();

  return useMutation({
    mutationKey: schoolId ? authMutationKeys.completeOnboarding(schoolId) : ['auth', 'complete-onboarding', 'no-school'],
    mutationFn: async (data: unknown) => {
      if (!schoolId) throw new Error('No active school');
      const parsed = completeOnboardingSchema.safeParse(data);
      if (!parsed.success) {
        throw new Error(parsed.error.issues[0].message);
      }
      return completeOnboarding(parsed.data);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: authQueryKeys.all });
      toast.success('Onboarding complete! Welcome to Somotracker.');
    },
    onError: (error: unknown) => {
      if (isApiError(error)) {
        // Map field errors to form if needed; fallback to generic toast with request id
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