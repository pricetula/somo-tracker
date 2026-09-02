/**
 * Types for the crash-resistant staff bulk-invite wizard.
 * Mirrors the student import types pattern.
 */

// ─── IndexedDB Record Types ───────────────────────────────────────────────

export interface StagedInviteRecord {
    id?: number;
    school_id?: string;
    email: string;
    full_name: string;
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
    updated_at: string;
    school_id: string;
    role: string;
    /**
     * Size guard: if set, the parsed file row data was too large to persist,
     * so resuming into column_mapping is not supported.
     */
    parsed_file_too_large?: boolean;
}

/**
 * Storable subset of ParsedFileResult for IndexedDB persistence.
 * Used to resume the wizard from column_mapping.
 */
export interface StoredParsedFile {
    id: `parsed_file:${string}`;
    file_name: string;
    sheet_name?: string;
    headers: string[];
    rows: Record<string, string>[];
    total_rows: number;
}

// ─── Wizard Steps ─────────────────────────────────────────────────────────

export type WizardStep = "upload" | "column_mapping" | "data_review" | "streaming";

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
 */
export const SESSION_STALE_MS = 24 * 60 * 60 * 1000; // 24 hours

/**
 * Maximum number of rows worth of parsed-file data to persist in IndexedDB.
 */
export const MAX_PERSISTED_ROWS = 500;

/**
 * Approximate per-row byte cost for the size guard.
 */
export const BYTES_PER_PERSISTED_ROW = 2048; // ~2 KB per row

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
    { target_key: "email", label: "Email", required: true },
    { target_key: "full_name", label: "Full Name", required: false },
];

/** Smart-matching dictionary: target -> array of English/Swahili variants. */
export const SMART_MATCH_DICT: Record<string, string[]> = {
    email: ["email", "email address", "e-mail", "mail", "barua pepe"],
    full_name: [
        "full name",
        "name",
        "names",
        "jina kamili",
        "jina",
        "staff name",
        "employee name",
        "teacher name",
        "learner name",
    ],
};

// ─── Error types for quota handling ───────────────────────────────────────

export function isQuotaExceededError(err: unknown): boolean {
    if (!err) return false;
    if (err instanceof DOMException) {
        return err.name === "QuotaExceededError" || err.name === "NS_ERROR_DOM_QUOTA_REACHED";
    }
    return false;
}
