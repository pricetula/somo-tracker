/**
 * Behavior API functions.
 *
 * Endpoints:
 *   GET    /api/v1/behavior/categories          — list categories
 *   POST   /api/v1/behavior/categories          — create category
 *   GET    /api/v1/behavior/categories/:id      — get category
 *   PUT    /api/v1/behavior/categories/:id      — update category
 *   POST   /api/v1/behavior/notes               — create behavior note
 *   GET    /api/v1/behavior/notes/queue         — pending review queue
 *   GET    /api/v1/behavior/notes/:id           — get note detail
 *   POST   /api/v1/behavior/notes/:id/review    — approve/reject
 */

import { api } from "./client";

// ─── Types ────────────────────────────────────────────────────────────────

export type BehaviorNoteStatus = "PENDING_REVIEW" | "APPROVED" | "REJECTED" | "INCLUDED_IN_REPORT";

export type BehaviorSeverity = "MINOR" | "NEEDS_FOLLOW_UP";

export interface BehaviorCategory {
    id: string;
    tenant_id: string;
    school_id: string;
    name: string;
    default_severity?: BehaviorSeverity | null;
    is_active: boolean;
    created_at?: string;
}

export interface CreateCategoryPayload {
    name: string;
    default_severity?: BehaviorSeverity | null;
}

export interface UpdateCategoryPayload {
    name?: string;
    default_severity?: BehaviorSeverity | null;
    is_active?: boolean;
}

export interface BehaviorNote {
    id: string;
    tenant_id: string;
    school_id: string;
    student_id: string;
    timetable_slot_id: string;
    date: string;
    category_id: string;
    description: string;
    is_urgent: boolean;
    status: BehaviorNoteStatus;
    authored_by_id: string;
    reviewed_by_id?: string | null;
    reviewed_at?: string | null;
    created_at: string;
}

export interface CreateNotePayload {
    timetable_slot_id: string;
    student_id: string;
    date: string;
    category_id: string;
    description: string;
    is_urgent: boolean;
}

export interface PendingNoteItem {
    id: string;
    student_id: string;
    student_full_name: string;
    class_name: string;
    category_id: string;
    category_name: string;
    description: string;
    is_urgent: boolean;
    authored_by_id: string;
    authored_by_name: string;
    date: string;
    status: BehaviorNoteStatus;
}

export interface PendingNotesResponse {
    notes: PendingNoteItem[];
}

export interface ReviewDecisionPayload {
    decision: "APPROVED" | "REJECTED";
    admin_note?: string;
}

// ─── Categories API ───────────────────────────────────────────────────────

/** List behavior categories for the active school. */
export async function listBehaviorCategories(
    activeOnly?: boolean
): Promise<{ items: BehaviorCategory[]; total: number }> {
    const params = activeOnly ? "?active_only=true" : "";
    return api.get<{ items: BehaviorCategory[]; total: number }>(
        `/api/v1/behavior/categories${params}`
    );
}

/** Create a new behavior category. */
export async function createBehaviorCategory(
    payload: CreateCategoryPayload
): Promise<BehaviorCategory> {
    return api.post<BehaviorCategory>("/api/v1/behavior/categories", payload);
}

/** Get a single behavior category by ID. */
export async function getBehaviorCategory(id: string): Promise<BehaviorCategory> {
    return api.get<BehaviorCategory>(`/api/v1/behavior/categories/${id}`);
}

/** Update a behavior category. */
export async function updateBehaviorCategory(
    id: string,
    payload: UpdateCategoryPayload
): Promise<BehaviorCategory> {
    return api.put<BehaviorCategory>(`/api/v1/behavior/categories/${id}`, payload);
}

// ─── Notes API ────────────────────────────────────────────────────────────

/** Create a new behavior note. */
export async function createBehaviorNote(payload: CreateNotePayload): Promise<BehaviorNote> {
    return api.post<BehaviorNote>("/api/v1/behavior/notes", payload);
}

/** Get the pending review queue. */
export async function getBehaviorPendingQueue(): Promise<PendingNotesResponse> {
    return api.get<PendingNotesResponse>("/api/v1/behavior/notes/queue");
}

/** Get a single behavior note by ID. */
export async function getBehaviorNote(id: string): Promise<BehaviorNote> {
    return api.get<BehaviorNote>(`/api/v1/behavior/notes/${id}`);
}

/** Review (approve/reject) a behavior note. */
export async function reviewBehaviorNote(
    id: string,
    payload: ReviewDecisionPayload
): Promise<{ message: string; decision: string }> {
    return api.post<{ message: string; decision: string }>(
        `/api/v1/behavior/notes/${id}/review`,
        payload
    );
}
