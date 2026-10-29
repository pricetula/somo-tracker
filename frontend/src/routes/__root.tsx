/**
 * Root route with authentication guard
 * Layer B: beforeLoad for client-side navigation (uses server function for auth state)
 */

import {
  createRootRoute,
  Link,
  Outlet,
  HeadContent,
  Scripts,
} from '@tanstack/react-router';
import { TanStackRouterDevtools } from '@tanstack/router-devtools';
import { getAuthState } from '@/features/auth/api/get-auth-state';
import { routePolicy } from '@/features/auth/utils/route-policy';
import '../styles/globals.css';

// ============================================================================
// Layer B: beforeLoad - Client-side navigation guard
// ============================================================================

async function authBeforeLoad({ location }: { location: { pathname: string } }) {
  // Get auth state from server function (reads cookie on server)
  const { hasSession } = await getAuthState();

  // Apply route policy
  const decision = routePolicy(location.pathname, hasSession);

  switch (decision.type) {
    case 'allow':
      return { hasSession };
    case 'redirect':
      throw new Response(null, {
        status: 302,
        headers: { Location: decision.to },
      });
    case 'redirect-with-next': {
      const redirectUrl = new URL(decision.to, window.location.origin);
      redirectUrl.searchParams.set('next', decision.next);
      throw new Response(null, {
        status: 302,
        headers: { Location: redirectUrl.toString() },
      });
    }
  }
}

// ============================================================================
// Root Route Definition
// ============================================================================

export const Route = createRootRoute({
  head: () => ({
    meta: [
      { charSet: 'utf-8' },
      { name: 'viewport', content: 'width=device-width, initial-scale=1' },
      { title: 'Somotracker' },
    ],
  }),
  beforeLoad: authBeforeLoad,
  component: RootComponent,
});

function RootComponent() {
  return (
    <RootDocument>
      <div className="min-h-screen bg-background text-foreground">
        <header className="border-b px-4 py-3">
          <nav className="flex gap-4">
            <Link to="/" className="[&.active]:font-bold">
              Home
            </Link>
          </nav>
        </header>
        <main className="p-6">
          <Outlet />
        </main>
      </div>
      <TanStackRouterDevtools />
    </RootDocument>
  );
}

function RootDocument({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <head>
        <HeadContent />
      </head>
      <body>
        {children}
        <Scripts />
      </body>
    </html>
  );
}

// Export types for child routes to use
export { routePolicy };