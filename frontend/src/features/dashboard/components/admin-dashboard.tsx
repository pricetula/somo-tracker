"use client";

import * as React from "react";
import Link from "next/link";
import { useMe } from "@/hooks/use-auth";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Loader2, LogOut, CalendarPlus, School } from "lucide-react";
import { SESSION_COOKIE_NAME } from "@/lib/auth";
import {
  PrepModeAlert,
  useCalendarEvaluator,
} from "@/features/calendar";
import {
  useClassStreamEvaluator,
  useClasses,
} from "@/features/classes";

// ─── Dashboard Composite Component ────────────────────────────────────────

export function DashboardPage() {
  const { data: session, isLoading: sessionLoading } = useMe();
  const calendarState = useCalendarEvaluator();
  const classStreamState = useClassStreamEvaluator();
  const { data: classes } = useClasses();

  // Forms are now route-based (intercepted modal or standalone page).
  // The evaluator hooks will naturally reflect state changes after successful
  // saves, so no local dismissed-state tracking is needed.

  // Combine loading: session or calendar data
  if (sessionLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  // Decide what to render in the top container zone
  const showForm =
    calendarState.type === "form";

  const showPrepAlert =
    calendarState.type === "hidden" && calendarState.alert === "prep-mode";

  // Step 2: Show class/stream generator after calendar is active and no classes exist
  const calendarIsActive = calendarState.type === "hidden" || calendarState.type === "form";
  const showClassStep =
    calendarIsActive &&
    classStreamState.type === "setup";

  // Dashboard is fully unlocked when the calendar is active AND classes are ready
  const dashboardUnlocked =
    calendarState.type === "hidden" && classStreamState.type === "ready";

  return (
    <div className="min-h-screen p-6">
      {/* ─── HIERARCHICAL CONTAINER ZONE (Top of layout) ─────────────── */}
      <div className="mb-8 space-y-6">
        {/* Step 1: Academic Calendar Setup */}
        {showForm && (
          <div className="animate-in slide-in-from-top-4 fade-in duration-300">
            <Link href="/dashboard/calendar/new" className="block">
              <div className="rounded-2xl border bg-card p-6 shadow-sm transition-all hover:border-primary/30 hover:shadow-md cursor-pointer">
                <div className="flex items-center gap-4">
                  <div className="flex h-12 w-12 items-center justify-center rounded-full bg-primary/10">
                    <CalendarPlus className="h-6 w-6 text-primary" />
                  </div>
                  <div>
                    <h2 className="text-lg font-semibold">
                      {calendarState.type === "form" && "mode" in calendarState && calendarState.mode === "next-cycle"
                        ? "Set Up Next Academic Year"
                        : "Set Up Academic Calendar"}
                    </h2>
                    <p className="text-sm text-muted-foreground">
                      Define your academic year and periods to unlock the dashboard
                    </p>
                  </div>
                </div>
              </div>
            </Link>
          </div>
        )}

        {/* Step 2: Class & Stream Generator */}
        {showClassStep && (
          <div className="animate-in slide-in-from-top-4 fade-in duration-300">
            <Link href="/dashboard/classes/generate" className="block">
              <div className="rounded-2xl border bg-card p-6 shadow-sm transition-all hover:border-primary/30 hover:shadow-md cursor-pointer">
                <div className="flex items-center gap-4">
                  <div className="flex h-12 w-12 items-center justify-center rounded-full bg-primary/10">
                    <School className="h-6 w-6 text-primary" />
                  </div>
                  <div>
                    <h2 className="text-lg font-semibold">
                      Establish Classes &amp; Streams
                    </h2>
                    <p className="text-sm text-muted-foreground">
                      Generate your classroom grid from grade tiers and stream sections
                    </p>
                  </div>
                </div>
              </div>
            </Link>
          </div>
        )}

        {showPrepAlert && <PrepModeAlert />}

        {calendarState.type === "loading" && (
          <div className="flex items-center justify-center rounded-2xl border bg-card p-8">
            <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
          </div>
        )}

        {classStreamState.type === "loading" && calendarIsActive && !showClassStep && (
          <div className="flex items-center justify-center rounded-2xl border bg-card p-8">
            <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
          </div>
        )}
      </div>

      {/* ─── HEADER ──────────────────────────────────────────────────── */}
      <header className="mb-8 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Dashboard</h1>
          <p className="text-sm text-muted-foreground">
            {dashboardUnlocked
              ? "All systems operational — dashboard is ready"
              : "Complete the onboarding steps above to unlock analytics"}
          </p>
        </div>
        <Button variant="outline" asChild>
          <Link href="/logout">
            <LogOut className="mr-2 h-4 w-4" />
            Sign Out
          </Link>
        </Button>
      </header>

      {/* ─── TOP OF PAGE STATISTICS SECTION ──────────────────────────── */}
      <div
        className={`mb-8 grid gap-6 md:grid-cols-3 transition-all duration-500 ${
          dashboardUnlocked ? "" : "pointer-events-none opacity-40 blur-sm"
        }`}
      >
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Total Students
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-3xl font-bold">—</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Total Teachers
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-3xl font-bold">—</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Active Classes
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-3xl font-bold">
              {classes && classes.length > 0 ? classes.length : "—"}
            </p>
          </CardContent>
        </Card>
      </div>

      {/* ─── REST OF THE PAGE (Analytics Workspace) ──────────────────── */}
      <div
        className={`space-y-6 transition-all duration-500 ${
          dashboardUnlocked ? "" : "pointer-events-none opacity-30 blur-sm"
        }`}
      >
        {/* Skeleton: Attendance Trends */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Attendance Trends</CardTitle>
            <CardDescription>
              Weekly attendance overview for the current term
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex h-48 items-center justify-center rounded-lg bg-muted/50">
              <p className="text-sm text-muted-foreground">
                {dashboardUnlocked
                  ? "Chart loading..."
                  : "🔒 Complete onboarding to unlock"}
              </p>
            </div>
          </CardContent>
        </Card>

        {/* Skeleton: Performance Assessment Rubrics */}
        <div className="grid gap-6 md:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">
                Performance Assessment
              </CardTitle>
              <CardDescription>
                Student competency rubrics
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="flex h-32 items-center justify-center rounded-lg bg-muted/50">
                <p className="text-sm text-muted-foreground">
                  {dashboardUnlocked
                    ? "Loading..."
                    : "🔒 Locked"}
                </p>
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Session Info</CardTitle>
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
    </div>
  );
}
