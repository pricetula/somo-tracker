import { createRootRoute, Link, Outlet } from '@tanstack/react-router';
import { TanStackRouterDevtools } from '@tanstack/router-devtools';

export const Route = createRootRoute({
  component: () => (
    <>
      <div className="min-h-screen bg-background text-foreground">
        <header className="border-b px-4 py-3">
          <nav className="flex gap-4">
            <Link to="/" className="[&.active]:font-bold">Home</Link>
          </nav>
        </header>
        <main className="p-6">
          <Outlet />
        </main>
      </div>
      <TanStackRouterDevtools />
    </>
  ),
});
