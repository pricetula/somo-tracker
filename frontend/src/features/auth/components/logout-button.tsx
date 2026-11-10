/**
 * Logout button - POST with JSON body via button onClick fetch
 * Never triggered by TanStack Router preloading
 */

'use client';

import { Button } from '@/components/ui/button';
import { LogOut, Loader2 } from 'lucide-react';
import { useLogout } from '../hooks/use-logout';

export function LogoutButton({ className = '' }: { className?: string }) {
  const { mutate: logout, isPending } = useLogout();

  return (
    <Button
      variant="ghost"
      size="icon"
      onClick={() => logout()}
      disabled={isPending}
      className={className}
      aria-label="Log out"
    >
      {isPending ? (
        <Loader2 className="h-4 w-4 animate-spin" />
      ) : (
        <LogOut className="h-4 w-4" />
      )}
    </Button>
  );
}