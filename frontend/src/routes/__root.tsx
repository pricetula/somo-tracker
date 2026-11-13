/**
 * Root route with authentication guard
 * Layer B: beforeLoad for client-side navigation (uses server function for auth state)
 */

import { HeadContent, Scripts, Link } from "@tanstack/react-router";
import { createRootRoute, Outlet, redirect } from "@tanstack/react-router";
import { TanStackRouterDevtools } from "@tanstack/router-devtools";
import { getAuthState } from "@/features/auth/api/get-auth-state";
import { routePolicy } from "@/features/auth/utils/route-policy";
import { QueryClientProvider } from "@tanstack/react-query";
import { queryClient } from "@/lib/query-client";
import { AppSidebar } from "@/components/shared/app-sidebar/app-sidebar";
import { ShellSidebarProvider } from "@/components/shared/sidebar-provider";
import { SidebarInset, SidebarTrigger } from "@/components/ui/sidebar";
import { Separator } from "@/components/ui/separator";
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import { ChevronRightIcon } from "lucide-react";
import "../styles/globals.css";

// ============================================================================
// Layer B: beforeLoad - Client-side navigation guard
// ============================================================================

async function authBeforeLoad({
  location,
}: {
  location: { pathname: string };
}) {
  // Get auth state from server function (reads cookie on server)
  const { hasSession } = await getAuthState();

  // Apply route policy
  const decision = routePolicy(location.pathname, hasSession);

  switch (decision.type) {
    case "allow":
      return { hasSession };
    case "redirect":
      throw redirect({ to: decision.to });
    case "redirect-with-next":
      throw redirect({ to: decision.to, search: { next: decision.next } });
  }
}

// ============================================================================
// Root Route Definition
// ============================================================================

export const Route = createRootRoute({
  head: () => ({
    meta: [
      { charSet: "utf-8" },
      { name: "viewport", content: "width=device-width, initial-scale=1" },
      { title: "Somotracker" },
    ],
  }),
  beforeLoad: authBeforeLoad,
  component: RootComponent,
});

function RootComponent() {
  return (
    <html lang="en">
      <head>
        <HeadContent />
      </head>
      <body>
        <ShellSidebarProvider>
          <div className="relative flex h-screen w-full min-w-0">
            <AppSidebar variant={undefined} />
            <SidebarInset className="flex-1 min-w-0 border-0">
              <header className="flex items-center gap-4 px-4 py-3 shrink-0">
                <SidebarTrigger className="border-0" />
                <Separator orientation="vertical" className="h-6 shrink-0" />
                <Breadcrumb className="flex-1 min-w-0">
                  <BreadcrumbList>
                    <BreadcrumbItem>
                      <BreadcrumbLink render={<Link to="/">Home</Link>} />
                    </BreadcrumbItem>
                    <BreadcrumbSeparator>
                      <ChevronRightIcon />
                    </BreadcrumbSeparator>
                    <BreadcrumbItem>
                      <BreadcrumbPage>Dashboard</BreadcrumbPage>
                    </BreadcrumbItem>
                  </BreadcrumbList>
                </Breadcrumb>
              </header>
              <div className="flex flex-1 flex-col min-w-0 overflow-hidden px-4">
                <QueryClientProvider client={queryClient}>
                  <Outlet />
                </QueryClientProvider>
              </div>
            </SidebarInset>
          </div>
        </ShellSidebarProvider>
        <Scripts />
        <TanStackRouterDevtools />
      </body>
    </html>
  );
}

// Export types for child routes to use
export { routePolicy };
