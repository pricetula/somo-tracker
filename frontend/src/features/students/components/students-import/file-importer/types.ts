/**
 * Types for the crash-resistant student import wizard.
 */

import type { CreateStudentPayload } from "@/lib/api/students";

// ─── IndexedDB Record Types ───────────────────────────────────────────────

export interface StagedStudentRecord {
    id?: number;
    payload: CreateStudentPayload;
    raw_row_data: Record<string, string>;
    status: "valid" | "error" | "duplicate" | "submitted";
    errors: string[];
    batch_id?: string;
}

export interface ImportSessionMeta {
    id: "current_session";
    current_step: WizardStep;
    file_name: string;
    source_sheet_name?: string;
    total_rows: number;
    column_mappings: Record<string, string | string[]>;
    class_mappings: Record<string, string>;
    completed_batch_ids: string[];
    last_active_tab_id?: string;
    updated_at: string;
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
