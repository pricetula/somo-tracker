"use client";

import * as React from "react";
import { useClasses } from "@/features/classes/hooks/use-classes";
import { useAcademicCalendar } from "@/features/calendar/hooks/use-academic-calendar";
import type { ClassStreamState } from "@/features/classes/types";

/**
 * Evaluate whether Step 2 of the onboarding lifecycle should mount.
 *
 * Decision Tree:
 *   CASE 1: Academic calendar is loading → return loading
 *   CASE 2: No academic calendar configured → return loading (prerequisite not met)
 *   CASE 3: Classes query is loading → return loading
 *   CASE 4: Classes array is empty → return "setup" (mount Step 2)
 *   CASE 5: Classes exist → return "ready" (hide Step 2)
 */
export function useClassStreamEvaluator(): ClassStreamState {
    const { data: calendar, isLoading: calendarLoading } = useAcademicCalendar();
    const { data: classes, isLoading: classesLoading } = useClasses();

    return React.useMemo(() => {
        // Prerequisite: academic calendar must be configured
        if (calendarLoading) return { type: "loading" };
        if (!calendar || !calendar.periods || calendar.periods.length === 0) {
            // Calendar not configured yet — wait (Step 1 must complete before Step 2)
            return { type: "loading" };
        }

        // Classes query in progress
        if (classesLoading) return { type: "loading" };

        // CASE 4: Empty classes list → mount the stream generator
        if (!classes || classes.length === 0) {
            return { type: "setup" };
        }

        // CASE 5: Classes exist → collapse Step 2
        return { type: "ready" };
    }, [calendar, calendarLoading, classes, classesLoading]);
}
