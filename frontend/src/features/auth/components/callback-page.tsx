/**
 * Auth callback page - Handles Stytch redirect
 * Extracts token, removes from URL via history.replaceState,
 * sets Referrer-Policy:no-referrer, POSTs to backend
 */

'use client';

import { useEffect, useState, useRef } from 'react';
import { useNavigate, useSearch } from '@tanstack/react-router';
import { authenticateToken } from '../api/authenticate-token';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardTitle, CardDescription } from '@/components/ui/card';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Loader2, CheckCircle, AlertCircle } from 'lucide-react';
import { toast } from 'sonner';

type CallbackStatus = 'loading' | 'success' | 'error';

export function AuthCallbackPage() {
  const search = useSearch({ from: '/auth/callback' });
  const navigate = useNavigate();
  const [status, setStatus] = useState<CallbackStatus>('loading');
  const [errorMessage, setErrorMessage] = useState<string>('');
  const handledRef = useRef(false);

  useEffect(() => {
    if (handledRef.current) return;
    handledRef.current = true;

    async function handleCallback() {
      // Read token directly from URL to avoid router re-render race after replaceState
      const params = new URLSearchParams(window.location.search);
      const token = params.get('token');
      const redirect = params.get('redirect') || search.redirect;

      // Remove token from URL immediately to prevent leakage (AC-2)
      if (token) {
        window.history.replaceState(null, '', '/auth/callback');
      }

      // Set Referrer-Policy to no-referrer (AC-2)
      const meta = document.createElement('meta');
      meta.name = 'referrer';
      meta.content = 'no-referrer';
      document.head.appendChild(meta);

      if (!token) {
        setStatus('error');
        setErrorMessage('No authentication token found in URL');
        return;
      }

      try {
        const response = await authenticateToken(token);

        // Use auth_state from backend (AC-8)
        const authState = response.auth_state || response.status;

        switch (authState) {
          case 'authenticated': {
            setStatus('success');
            // Redirect to dashboard or intended destination
            const redirectTarget = redirect || '/';
            navigate({ to: redirectTarget, replace: true });
            break;
          }
          case 'onboarding_required': {
            setStatus('success');
            // Redirect to onboarding tenant-name step
            navigate({ to: '/onboarding/tenant-name', replace: true });
            break;
          }
          case 'organization_selection_required': {
            setStatus('success');
            // Redirect to tenant-name for now - organization selection handled later
            navigate({ to: '/onboarding/tenant-name', replace: true });
            break;
          }
          default: {
            setStatus('error');
            setErrorMessage(`Unknown authentication status: ${authState}`);
            break;
          }
        }
      } catch (error) {
        setStatus('error');
        const message = error instanceof Error ? error.message : 'Authentication failed';
        setErrorMessage(message);
        toast.error(message);
      }
    }

    handleCallback();
  }, [search.token, search.redirect, navigate]);

  if (status === 'loading') {
    return (
      <Card className="w-full max-w-md mx-auto mt-20">
        <CardContent className="py-12 text-center">
          <Loader2 className="h-8 w-8 animate-spin mx-auto text-primary mb-4" />
          <CardTitle>Completing sign in...</CardTitle>
          <CardDescription>Please wait while we verify your magic link</CardDescription>
        </CardContent>
      </Card>
    );
  }

  if (status === 'error') {
    return (
      <Card className="w-full max-w-md mx-auto mt-20">
        <CardContent className="py-12 text-center">
          <AlertCircle className="h-8 w-8 text-destructive mx-auto mb-4" />
          <CardTitle>Sign in failed</CardTitle>
          <Alert variant="destructive" className="mt-4">
            <AlertDescription>{errorMessage}</AlertDescription>
          </Alert>
          <Button
            variant="outline"
            className="mt-4"
            onClick={() => navigate({ to: '/login', replace: true })}
          >
            Try again
          </Button>
        </CardContent>
      </Card>
    );
  }

  // Success - brief flash before redirect
  return (
    <Card className="w-full max-w-md mx-auto mt-20">
      <CardContent className="py-12 text-center">
        <CheckCircle className="h-8 w-8 text-emerald-600 mx-auto mb-4" />
        <CardTitle>Signed in successfully</CardTitle>
        <CardDescription>Redirecting...</CardDescription>
      </CardContent>
    </Card>
  );
}