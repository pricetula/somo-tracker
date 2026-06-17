/**
 * Academic Calendar API client.
 *
 * Endpoints:
 *   GET  /api/v1/schools/current-calendar  → AcademicYear | null
 *   POST /api/v1/schools/current-calendar  → AcademicYear
 */

import { api } from "@/lib/api/client";
import type { AcademicYear, CreateAcademicCalendarPayload } from "@/features/calendar/types";

/** Fetch the current academic calendar for the authenticated school. */
export async function fetchCurrentCalendar(): Promise<AcademicYear | null> {
    try {
        return await api.get<AcademicYear>("/api/v1/schools/current-calendar");
    } catch (err) {
        // 404 means no calendar exists yet
        if (
            err &&
            typeof err === "object" &&
            "status" in err &&
            (err as { status: number }).status === 404
        ) {
            return null;
        }
        throw err;
    }
}

/** Create or update the academic calendar. */
export async function saveAcademicCalendar(
    payload: CreateAcademicCalendarPayload
): Promise<AcademicYear> {
    return await api.post<AcademicYear>("/api/v1/schools/current-calendar", payload);
}
