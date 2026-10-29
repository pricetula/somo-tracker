/**
 * Login route - Email input + magic link trigger
 */

import { createFileRoute } from '@tanstack/react-router';
import { LoginForm } from '@/features/auth/components/login-form';

export const Route = createFileRoute('/login')({
  component: LoginPage,
});

function LoginPage() {
  return (
    <div className="min-h-screen flex items-center justify-center bg-background px-4 py-12">
      <LoginForm />
    </div>
  );
}