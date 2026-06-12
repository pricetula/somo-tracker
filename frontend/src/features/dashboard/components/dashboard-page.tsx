"use client";

import { useMe, useLogout } from "@/hooks/use-auth";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Loader2, LogOut } from "lucide-react";
import { SESSION_COOKIE_NAME } from "@/lib/auth";

export function DashboardPage() {
  const { data: session, isLoading } = useMe();
  const logoutMutation = useLogout();

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  return (
    <div className="min-h-screen p-6">
      <header className="mb-8 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Dashboard</h1>
          <p className="text-sm text-muted-foreground">
            You have a valid session
          </p>
        </div>
        <Button
          variant="outline"
          onClick={() => logoutMutation.mutate()}
          disabled={logoutMutation.isPending}
        >
          {logoutMutation.isPending ? (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          ) : (
            <LogOut className="mr-2 h-4 w-4" />
          )}
          Sign Out
        </Button>
      </header>

      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Session</CardTitle>
            <CardDescription>
              Authenticated via {SESSION_COOKIE_NAME} cookie
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-1 text-sm">
            {session ? (
              <>
                <p>
                  <span className="font-medium text-muted-foreground">
                    User ID:
                  </span>{" "}
                  <code className="rounded bg-muted px-1 py-0.5 text-xs">
                    {session.user_id}
                  </code>
                </p>
                <p>
                  <span className="font-medium text-muted-foreground">
                    Tenant ID:
                  </span>{" "}
                  <code className="rounded bg-muted px-1 py-0.5 text-xs">
                    {session.tenant_id}
                  </code>
                </p>
              </>
            ) : (
              <p className="text-muted-foreground">
                Unable to load session details.
              </p>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
