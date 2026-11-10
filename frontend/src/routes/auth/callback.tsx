/**
 * Auth callback route - Handles Stytch redirect
 */

import { createFileRoute } from '@tanstack/react-router';
import { z } from 'zod';
import { AuthCallbackPage } from '@/features/auth/components/callback-page';

const searchSchema = z.object({
  token: z.string().optional(),
  redirect: z.string().optional(),
});

export const Route = createFileRoute('/auth/callback')({
  validateSearch: searchSchema,
  component: AuthCallbackPage,
});