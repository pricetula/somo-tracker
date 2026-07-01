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

import { SessionForm } from "@/features/assessment";
import { listClasses } from "@/lib/api/classes";
import type { Class } from "@/lib/api/classes";

export default function NewSessionPage() {
    const [classes, setClasses] = React.useState<Class[]>([]);
    const [classesLoading, setClassesLoading] = React.useState(true);

    React.useEffect(() => {
        // For now, we fetch classes with empty academic_year_id as a placeholder.
        // In production, the active academic year/term should come from user context or settings.
        async function load() {
            try {
                // Try fetching classes; if it fails due to missing year params,
                // we'll show an empty list with an appropriate message.
                const result = await listClasses({
                    academic_year_id: "",
                    academic_term_id: "",
                });
                setClasses(result.data ?? []);
            } catch {
                // Classes endpoint requires academic_year_id and academic_term_id.
                // For now, return empty — the form handles the empty state.
                setClasses([]);
            } finally {
                setClassesLoading(false);
            }
        }
        load();
    }, []);

    return (
        <div className="mx-auto flex max-w-xl flex-col px-6 pt-6 pb-8">
            <div className="mb-6">
                <h1 className="text-2xl font-semibold tracking-tight">New Assessment Session</h1>
                <p className="text-muted-foreground mt-1 text-sm">
                    Create an assessment session by selecting a blueprint and a class. After
                    creation, you&apos;ll be taken to the scoring grid to record learner results.
                </p>
            </div>

            <SessionForm classes={classes} classesLoading={classesLoading} />
        </div>
    );
}
