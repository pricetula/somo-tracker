/**
 * Enrollment Timeline — shows a reverse-chronological list of term enrollments.
 *
 * Each entry: Term name, Class name, Status badge, Academic year.
 */

"use client";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { UserPlus } from "lucide-react";

import type { Enrollment } from "../types";

// ─── Status colors ────────────────────────────────────────────────────────

const STATUS_COLORS: Record<string, string> = {
    ACTIVE: "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400",
    COMPLETED: "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400",
    TRANSFERRED: "bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400",
    WITHDRAWN: "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400",
};

// ─── Props ─────────────────────────────────────────────────────────────────

interface EnrollmentTimelineProps {
    enrollments: Enrollment[];
    isLoading: boolean;
    onEnrollClick: () => void;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function EnrollmentTimeline({
    enrollments,
    isLoading,
    onEnrollClick,
}: EnrollmentTimelineProps) {
    if (isLoading) {
        return (
            <div className="space-y-3">
                <Skeleton className="h-6 w-48" />
                {Array.from({ length: 3 }).map((_, i) => (
                    <Skeleton key={i} className="h-16 w-full" />
                ))}
            </div>
        );
    }

    return (
        <div>
            <div className="mb-3 flex items-center justify-between">
                <h2 className="text-lg font-medium">Enrollment History</h2>
                <Button variant="outline" size="sm" onClick={onEnrollClick}>
                    <UserPlus className="mr-1.5 size-3.5" />
                    Enroll in New Term
                </Button>
            </div>

            {enrollments.length === 0 ? (
                <div className="bg-muted/30 flex items-center justify-center rounded-md px-4 py-8">
                    <div className="text-center">
                        <p className="text-muted-foreground text-sm font-medium">
                            No enrollment history
                        </p>
                        <p className="text-muted-foreground mt-1 text-xs">
                            Enroll this student in a class to get started.
                        </p>
                    </div>
                </div>
            ) : (
                <div className="space-y-0">
                    {enrollments.map((enrollment, idx) => {
                        const isLast = idx === enrollments.length - 1;
                        return (
                            <div key={enrollment.id} className="relative flex gap-4 pb-4">
                                {/* Timeline connector */}
                                <div className="flex flex-col items-center">
                                    <div className="bg-primary z-10 size-2.5 rounded-full" />
                                    {!isLast && (
                                        <div className="border-border/40 mt-1 w-px flex-1 border-l" />
                                    )}
                                </div>

                                {/* Content */}
                                <div className="flex-1 pb-2">
                                    <div className="flex flex-wrap items-center gap-2">
                                        <span className="text-sm font-medium">
                                            {enrollment.term_name}
                                        </span>
                                        {enrollment.academic_year && (
                                            <span className="text-muted-foreground text-xs">
                                                {enrollment.academic_year}
                                            </span>
                                        )}
                                    </div>
                                    <div className="mt-0.5 flex flex-wrap items-center gap-2">
                                        <span className="text-muted-foreground text-xs">
                                            {enrollment.class_name || "—"}
                                        </span>
                                        {enrollment.status && (
                                            <Badge
                                                variant="secondary"
                                                className={
                                                    STATUS_COLORS[enrollment.status] ??
                                                    "bg-muted text-muted-foreground"
                                                }
                                            >
                                                {enrollment.status}
                                            </Badge>
                                        )}
                                    </div>
                                </div>
                            </div>
                        );
                    })}
                </div>
            )}
        </div>
    );
}
