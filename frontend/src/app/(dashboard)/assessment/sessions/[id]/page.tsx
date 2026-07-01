/**
 * Session Detail page.
 *
 * Shows session metadata and linked rubric results.
 * Provides link to the full scoring grid.
 * Maps to GET /api/v1/assessment/sessions/:id.
 */

"use client";

import * as React from "react";
import { useParams, useRouter } from "next/navigation";

import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { ArrowLeft, ClipboardCheck, ExternalLink } from "lucide-react";

import { useSessionDetail } from "@/features/assessment";

const RUBRIC_COLORS: Record<string, string> = {
    EE: "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400",
    ME: "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400",
    AE: "bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400",
    BE: "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400",
};

export default function SessionDetailPage() {
    const params = useParams();
    const router = useRouter();
    const id = params.id as string;

    const { data: sessionData, isLoading, isError } = useSessionDetail(id);

    const session = sessionData?.data;

    if (isLoading) {
        return (
            <div className="flex flex-col gap-4 px-6 pt-6 pb-8">
                <Skeleton className="h-8 w-64" />
                <Skeleton className="h-4 w-48" />
                <Skeleton className="mt-4 h-32 w-full" />
            </div>
        );
    }

    if (isError || !session) {
        return (
            <div className="flex items-center justify-center py-16">
                <div className="text-center">
                    <p className="text-destructive text-sm font-medium">
                        Failed to load session details.
                    </p>
                    <Button
                        variant="outline"
                        size="sm"
                        className="mt-4"
                        onClick={() => router.push("/assessment/sessions")}
                    >
                        Back to Sessions
                    </Button>
                </div>
            </div>
        );
    }

    const dateFormatted = session.date_administered
        ? new Date(session.date_administered).toLocaleDateString("en-US", {
              month: "long",
              day: "numeric",
              year: "numeric",
          })
        : "—";

    // Group results by student_id for a summary view
    const resultsByStudent = new Map<string, typeof session.results>();
    for (const r of session.results) {
        const existing = resultsByStudent.get(r.student_id) ?? [];
        existing.push(r);
        resultsByStudent.set(r.student_id, existing);
    }

    return (
        <div className="flex flex-1 flex-col px-6 pt-6 pb-8">
            {/* Back link */}
            <Button
                variant="ghost"
                size="sm"
                className="mb-4 w-fit"
                onClick={() => router.push("/assessment/sessions")}
            >
                <ArrowLeft className="mr-1.5 size-4" />
                Back to Sessions
            </Button>

            {/* Session metadata */}
            <div className="mb-6">
                <div className="flex items-start justify-between">
                    <div>
                        <h1 className="text-2xl font-semibold tracking-tight">Session Detail</h1>
                        <div className="mt-2 flex flex-wrap items-center gap-3">
                            <span className="text-muted-foreground text-sm">
                                Blueprint: {session.blueprint_id}
                            </span>
                            <span className="text-muted-foreground text-sm">•</span>
                            <span className="text-muted-foreground text-sm">
                                Date: {dateFormatted}
                            </span>
                            <span className="text-muted-foreground text-sm">•</span>
                            <span className="text-muted-foreground text-sm">
                                Results: {session.results.length}
                            </span>
                        </div>
                    </div>
                    <Button
                        variant="outline"
                        size="sm"
                        onClick={() => router.push(`/assessment/sessions/${id}/score`)}
                    >
                        <ClipboardCheck className="mr-1.5 size-4" />
                        Open Scoring Grid
                    </Button>
                </div>
            </div>

            {/* Results summary */}
            <div>
                <h2 className="mb-3 text-lg font-medium">Rubric Results</h2>
                {session.results.length === 0 ? (
                    <div className="bg-muted/30 flex items-center justify-center rounded-md px-4 py-8">
                        <div className="text-center">
                            <p className="text-muted-foreground text-sm font-medium">
                                No results recorded yet
                            </p>
                            <p className="text-muted-foreground mt-1 text-xs">
                                Use the scoring grid to record learner rubric results.
                            </p>
                            <Button
                                variant="outline"
                                size="sm"
                                className="mt-4"
                                onClick={() => router.push(`/assessment/sessions/${id}/score`)}
                            >
                                <ExternalLink className="mr-1.5 size-3.5" />
                                Go to Scoring Grid
                            </Button>
                        </div>
                    </div>
                ) : (
                    <div className="ring-foreground/10 rounded-lg ring-1">
                        <table className="w-full">
                            <thead>
                                <tr className="border-border/40 border-b">
                                    <th className="text-muted-foreground px-3 py-2 text-left text-xs font-medium tracking-wider uppercase">
                                        Student
                                    </th>
                                    <th className="text-muted-foreground px-3 py-2 text-left text-xs font-medium tracking-wider uppercase">
                                        Indicator
                                    </th>
                                    <th className="text-muted-foreground px-3 py-2 text-left text-xs font-medium tracking-wider uppercase">
                                        Rubric
                                    </th>
                                </tr>
                            </thead>
                            <tbody>
                                {session.results.map((r) => (
                                    <tr
                                        key={r.id}
                                        className="border-border/40 hover:bg-muted/30 border-b transition-colors"
                                    >
                                        <td className="px-3 py-2 text-sm font-medium">
                                            {r.student_id}
                                        </td>
                                        <td className="text-muted-foreground px-3 py-2 text-sm">
                                            {r.indicator_id}
                                        </td>
                                        <td className="px-3 py-2">
                                            <Badge
                                                variant="secondary"
                                                className={
                                                    RUBRIC_COLORS[r.rubric_level] ??
                                                    "bg-muted text-muted-foreground"
                                                }
                                            >
                                                {r.rubric_level}
                                            </Badge>
                                        </td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>
                )}
            </div>
        </div>
    );
}
