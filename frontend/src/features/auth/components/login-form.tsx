/**
 * Login form - Email input + magic link trigger
 */

'use client';

import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Separator } from '@/components/ui/separator';
import { Mail, Loader2 } from 'lucide-react';
import { loginFormSchema, type LoginFormInput } from '../schemas';
import { useRequestMagicLink } from '../hooks/use-request-magic-link';

export function LoginForm({ _redirect }: { _redirect?: string }) {
  const [emailSent, setEmailSent] = useState(false);
  const { mutate: requestMagicLink, isPending } = useRequestMagicLink();

  const form = useForm<LoginFormInput>({
    resolver: zodResolver(loginFormSchema),
    defaultValues: { email: '' },
  });

  const onSubmit = (data: LoginFormInput) => {
    requestMagicLink(data.email, {
      onSuccess: () => setEmailSent(true),
    });
  };

  return (
    <Card className="w-full max-w-md mx-auto">
      <CardHeader className="space-y-4">
        <CardTitle className="text-2xl text-center">Welcome back</CardTitle>
        <CardDescription className="text-center">
          Enter your email to receive a magic link for sign in
        </CardDescription>
      </CardHeader>
      <CardContent>
        {emailSent && (
          <Alert className="mb-4" variant="default">
            <AlertDescription className="text-center">
              Magic link sent! Check your email and click the link to continue.
            </AlertDescription>
          </Alert>
        )}

        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="email">Email</Label>
            <Input
              id="email"
              type="email"
              placeholder="you@school.edu"
              autoComplete="email"
              {...form.register('email')}
              disabled={isPending || emailSent}
              aria-describedby={form.formState.errors.email ? 'email-error' : undefined}
            />
            {form.formState.errors.email && (
              <p id="email-error" className="text-sm text-destructive" role="alert">
                {form.formState.errors.email.message}
              </p>
            )}
          </div>

          <Button
            type="submit"
            className="w-full"
            disabled={isPending || emailSent}
            size="default"
          >
            {isPending ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Sending...
              </>
            ) : (
              <>
                <Mail className="mr-2 h-4 w-4" />
                Send Magic Link
              </>
            )}
          </Button>
        </form>

        <Separator className="my-6" />

        <p className="text-center text-sm text-muted-foreground">
          By continuing, you agree to our{' '}
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