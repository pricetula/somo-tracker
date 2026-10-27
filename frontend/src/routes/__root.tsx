import {
  createRootRoute,
  Link,
  Outlet,
  HeadContent,
  Scripts,
} from "@tanstack/react-router";
import "../styles/globals.css";
import { TanStackRouterDevtools } from "@tanstack/router-devtools";

export const Route = createRootRoute({
  head: () => ({
    meta: [
      { charSet: "utf-8" },
      { name: "viewport", content: "width=device-width, initial-scale=1" },
      { title: "Somotracker" },
    ],
  }),
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
