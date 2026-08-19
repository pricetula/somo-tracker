/**
 * Teacher Delivery API functions.
 *
 * Endpoints:
 *   GET /api/v1/teacher-delivery-summaries/breakdown — per-teacher Marked vs.
 *       Missed slot counts for the School Administrator dashboard
 */

import { api } from "./client";

// ─── Types ────────────────────────────────────────────────────────────────

/**
 * Per-teacher Marked vs. Missed rollup returned by
 * GET /api/v1/teacher-delivery-summaries/breakdown.
 *
 * Backend contract: backend/internal/teacherdeliverysummaries/repository.go —
 * ListDeliveryBreakdown. Items are ordered by missed_slots descending so
 * chronic non-compliant teachers surface first (compliance watch).
 */
export interface TeacherDeliveryBreakdownItem {
    teacher_id: string;
    teacher_name: string;
    /** Teachers Service Commission registration number (null when unassigned). */
    tsc_number: string | null;
    total_assigned_slots: number;
    marked_slots: number;
    missed_slots: number;
    on_time_submission_rate: number;
}

/** Wrapper returned by GET /api/v1/teacher-delivery-summaries/breakdown. */
export interface TeacherDeliveryBreakdownList {
    items: TeacherDeliveryBreakdownItem[];
    total: number;
}

// ─── API Functions ─────────────────────────────────────────────────────────

/**
 * Fetch per-teacher Marked vs. Missed slot counts for a school term.
 *
 * @param termId Academic term id (UUID) — the term to aggregate
 *               (teacher_delivery_summaries are per teacher × term).
 */
export async function getTeacherDeliveryBreakdown(
    termId: string
): Promise<TeacherDeliveryBreakdownList> {
    const searchParams = new URLSearchParams({ academic_term_id: termId });

    return api.get<TeacherDeliveryBreakdownList>(
        `/api/v1/teacher-delivery-summaries/breakdown?${searchParams.toString()}`
    );
}
