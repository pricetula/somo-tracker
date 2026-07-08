/**
 * Types for the crash-resistant student import wizard.
 */

import type { CreateStudentPayload } from "@/lib/api/students";

// ─── IndexedDB Record Types ───────────────────────────────────────────────

export interface StagedStudentRecord {
    id?: number;
    school_id?: string;
    payload: CreateStudentPayload;
    raw_row_data: Record<string, string>;
    status: "valid" | "error" | "duplicate" | "submitted";
    errors: string[];
}

export interface ImportSessionMeta {
    id: `session:${string}`;
    current_step: WizardStep;
    file_name: string;
    source_sheet_name?: string;
    total_rows: number;
    column_mappings: Record<string, string | string[]>;
    class_mappings: Record<string, string>;
    updated_at: string;
    school_id: string;
    /**
     * Size guard: if set, the parsed file row data was too large to persist,
     * so resuming into column_mapping or class_resolving is not supported.
     */
    parsed_file_too_large?: boolean;
}

/**
 * Storable subset of ParsedFileResult for IndexedDB persistence.
 * Used to resume the wizard from column_mapping or class_resolving.
 */
export interface StoredParsedFile {
    id: `parsed_file:${string}`;
    file_name: string;
    sheet_name?: string;
    headers: string[];
    rows: Record<string, string>[];
    total_rows: number;
}

/** Unresolved class entry — aggregated from raw column data. */
export interface UnresolvedClassEntry {
    raw_string: string;
    count: number;
    status: "ambiguous" | "unmatched" | "matched";
    candidates: ClassMatchCandidate[];
    resolved_id: string | null;
}

export interface ClassMatchCandidate {
    class_id: string;
    display_label: string;
    score: number;
}

// ─── Wizard Steps ─────────────────────────────────────────────────────────

export type WizardStep =
    | "upload"
    | "column_mapping"
    | "class_resolving"
    | "data_review"
    | "streaming";

/**
 * Terminal import job statuses recognised by the frontend.
 * When any of these is observed via polling, IndexedDB should be cleared.
 */
export const TERMINAL_JOB_STATUSES = [
    "completed",
    "completed_with_errors",
    "failed",
    "cancelled",
] as const;

/**
 * Staleness threshold for import sessions.
 * Sessions older than this are considered stale and will prompt the user
 * to resume or discard rather than auto-resuming silently.
 */
export const SESSION_STALE_MS = 24 * 60 * 60 * 1000; // 24 hours

/**
 * Maximum number of rows worth of parsed-file data to persist in IndexedDB.
 * Files larger than this will skip persisting full row data, making the
 * session non-resumable from column_mapping/class_resolving steps.
 *
 * Chosen as a conservative fraction of MaxImportRows (5000) to avoid
 * overloading IndexedDB storage while still covering the common case.
 */
export const MAX_PERSISTED_ROWS = 500;

/**
 * Approximate per-row byte cost for the size guard.
 * Accounts for the JSON-serialized raw_row_data + overhead.
 */
export const BYTES_PER_PERSISTED_ROW = 2048; // ~2 KB per row is a safe upper bound

// ─── Parsed File Result ───────────────────────────────────────────────────

export interface ParsedFileResult {
    file_name: string;
    sheet_name?: string;
    headers: string[];
    rows: Record<string, string>[];
    total_rows: number;
}

// ─── Column Mapping Helpers ───────────────────────────────────────────────

export interface ColumnMappingOption {
    target_key: string;
    label: string;
    required: boolean;
}

export const TARGET_FIELDS: ColumnMappingOption[] = [
    { target_key: "full_name", label: "Full Name", required: true },
    { target_key: "gender", label: "Gender", required: false },
    { target_key: "date_of_birth", label: "Date of Birth", required: false },
    { target_key: "upi_number", label: "UPI Number", required: false },
    { target_key: "knec_assessment_number", label: "KNEC Assessment Number", required: false },
    { target_key: "class_id", label: "Class/Stream", required: false },
];

/** Smart-matching dictionary: target -> array of English/Swahili variants. */
export const SMART_MATCH_DICT: Record<string, string[]> = {
    full_name: [
        "full name",
        "name",
        "jina kamili",
        "mwanafunzi",
        "student name",
        "student",
        "names",
        "jina",
        "learner name",
        "learner",
    ],
    gender: ["gender", "jinsia", "sex", "jeni"],
    date_of_birth: [
        "dob",
        "date of birth",
        "tarehe ya kuzaliwa",
        "birth date",
        "birthday",
        "tarehe ya kuzaliwa mwanafunzi",
    ],
    upi_number: [
        "upi",
        "unique identifier",
        "nambari ya usajili",
        "upi number",
        "upi no",
        "upi#",
        "unique pupil identifier",
    ],
    knec_assessment_number: [
        "knec assessment number",
        "knec number",
        "knec no",
        "assessment number",
        "knec#",
        "knec",
        "nambari ya kn",
        "nambari ya mtihani",
    ],
    class_id: [
        "class",
        "stream",
        "grade",
        "class/stream",
        "darasa",
        "grade level",
        "level",
        "form",
        "class name",
        "stream name",
        "daraja",
        "kidato",
    ],
};

// ─── Streaming State ──────────────────────────────────────────────────────

export interface StreamingProgress {
    total_batches: number;
    completed_batches: number;
    success_count: number;
    failed_count: number;
    current_batch: number;
    status: "idle" | "streaming" | "paused" | "completed" | "failed";
    error_message?: string;
}

// ─── Error types for quota handling ───────────────────────────────────────

/** Check whether an error is a QuotaExceededError from IndexedDB. */
export function isQuotaExceededError(err: unknown): boolean {
    if (!err) return false;
    if (err instanceof DOMException) {
        return err.name === "QuotaExceededError" || err.name === "NS_ERROR_DOM_QUOTA_REACHED";
    }
    return false;
}
