/**
 * Hook for requesting magic link
 */

import { useMutation } from '@tanstack/react-query';
import { requestMagicLink } from '../api/request-magic-link';
import { authMutationKeys } from '../query-keys';
import { requestMagicLinkSchema } from '../schemas';
import { toast } from 'sonner';

export function useRequestMagicLink() {
  return useMutation({
    mutationKey: authMutationKeys.requestMagicLink(),
    mutationFn: (email: string) => {
      const parsed = requestMagicLinkSchema.safeParse({ email });
      if (!parsed.success) {
        throw new Error(parsed.error.issues[0].message);
      }
      return requestMagicLink(parsed.data.email);
    },
    onSuccess: () => {
      toast.success('Magic link sent! Check your email.');
    },
    onError: (error: Error) => {
      toast.error(error.message || 'Failed to send magic link');
    },
  });
}