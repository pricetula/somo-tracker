/**
 * IndexedDB wrapper for persisting bulk import drafts.
 * Uses idb-keyval underneath for simple key-value semantics,
 * namespaced by tenant_id + user_id + page context.
 */

import { get, set, del } from "idb-keyval";

// ─── Types ─────────────────────────────────────────────────────────────────

export interface ImportDraftRow {
    temp_id: string;
    email: string;
    first_name: string;
    last_name: string;
    phone: string;
    registration_number: string;
}

export interface ImportDraft {
    rows: ImportDraftRow[];
    createdAt: number; // Unix ms
    updatedAt: number; // Unix ms
    totalRows: number;
}

// ─── Constants ─────────────────────────────────────────────────────────────

const DRAFT_TTL_MS = 48 * 60 * 60 * 1000; // 48 hours

// ─── Key helpers ───────────────────────────────────────────────────────────

function draftKey(tenantID: string, userID: string, context: string): string {
    return `import_draft:${tenantID}:${userID}:${context}`;
}

// ─── Public API ────────────────────────────────────────────────────────────

/** Save an import draft to IndexedDB. */
export async function saveDraft(
    tenantID: string,
    userID: string,
    context: string,
    rows: ImportDraftRow[]
): Promise<void> {
    const draft: ImportDraft = {
        rows,
        createdAt: Date.now(),
        updatedAt: Date.now(),
        totalRows: rows.length,
    };
    await set(draftKey(tenantID, userID, context), draft);
}

/** Load a non-expired draft from IndexedDB. Returns null if absent or stale. */
export async function loadDraft(
    tenantID: string,
    userID: string,
    context: string
): Promise<ImportDraft | null> {
    const draft = await get<ImportDraft | undefined>(draftKey(tenantID, userID, context));
    if (!draft) return null;

    // Check TTL
    const age = Date.now() - draft.updatedAt;
    if (age > DRAFT_TTL_MS) {
        // Silently clear stale drafts
        await del(draftKey(tenantID, userID, context));
        return null;
    }

    return draft;
}

/** Delete a draft from IndexedDB. */
export async function clearDraft(tenantID: string, userID: string, context: string): Promise<void> {
    await del(draftKey(tenantID, userID, context));
}
