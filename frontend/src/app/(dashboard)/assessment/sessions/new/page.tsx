/**
 * Create Session page.
 *
 * Step 1: Select blueprint
 * Step 2: Select class, choose date
 * Submit → creates session, navigates to score page
 *
 * Maps to POST /api/v1/assessment/sessions.
 */

"use client";

import * as React from "react";
import { useQuery } from "@tanstack/react-query";

import { SessionForm, useActiveAcademicYear, useActiveTerm } from "@/features/assessment";
import { listClasses } from "@/lib/api/classes";
import type { Class } from "@/lib/api/classes";

export default function NewSessionPage() {
    // Fetch the active (current) academic year
    const {
        data: activeYear,
        isLoading: yearLoading,
        isError: yearError,
    } = useActiveAcademicYear();

    // Fetch the active term within that year
    const { data: activeTerm, isLoading: termLoading } = useActiveTerm(activeYear?.id);

    // Fetch classes using the resolved academic year & term IDs
    const {
        data: classesData,
        isLoading: classesLoading,
        isError: classesError,
    } = useQuery({
        queryKey: ["classes", "for-session", activeYear?.id, activeTerm?.id],
        queryFn: async (): Promise<Class[]> => {
            const result = await listClasses({
                academic_year_id: activeYear!.id,
                academic_term_id: activeTerm!.id,
            });
            return result.data ?? [];
        },
        enabled: !!activeYear?.id && !!activeTerm?.id,
        staleTime: 30_000,
    });

    const classes = classesData ?? [];
    const isLoading = yearLoading || termLoading || classesLoading;
    const hasError = yearError || classesError;

    return (
        <div className="mx-auto flex max-w-xl flex-col px-6 pt-6 pb-8">
            <div className="mb-6">
                <h1 className="text-2xl font-semibold tracking-tight">New Assessment Session</h1>
                <p className="text-muted-foreground mt-1 text-sm">
                    Create an assessment session by selecting a blueprint and a class. After
                    creation, you&apos;ll be taken to the scoring grid to record learner results.
                </p>
                {activeYear && activeTerm && (
                    <p className="text-muted-foreground mt-2 text-xs">
                        Active period: {activeYear.name}, {activeTerm.name}
                    </p>
                )}
                {hasError && (
                    <p className="text-destructive mt-2 text-xs">
                        Failed to load academic period or classes. Check your school configuration.
                    </p>
                )}
            </div>

            <SessionForm classes={classes} classesLoading={isLoading} />
        </div>
    );
}
