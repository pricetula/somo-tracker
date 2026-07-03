/**
 * IndexedDB schema types for the bulk student import pipeline.
 *
 * See spec §IndexedDB Schema  for the full schema definition.
 */

// ─── Stage enum ───────────────────────────────────────────────────────────

export type ImportStage = "MAPPING" | "PREVIEW" | "READY" | "SUBMITTING";

// ─── Column mapping shape ────────────────────────────────────────────────

export interface ColumnMapping {
    full_name: string[]; // ordered — concatenation order == selection order
    gender: string | null;
    date_of_birth: string | null;
    class_room: string | null;
    nemis_number: string | null;
    assessment_number: string | null;
    birth_certificate_number: string | null;
}

// ─── ImportMeta (single record, keyed by school_id) ──────────────────────

export interface ImportMeta {
    school_id: string;
    current_stage: ImportStage;
    column_mapping: ColumnMapping;
    academic_year_id: string;
    academic_term_id: string;
    total_rows: number;
    schema_version: number;
    created_at: string; // ISO timestamp
    classes_last_fetched_at: string | null;
    idempotency_key: string | null; // generated once at Stage 3 → Stage 4 transition
    import_job_id: string | null; // set once Stage 4 dispatches
    file_name?: string; // original file name for display
}

export const CURRENT_SCHEMA_VERSION = 2; // v2: academic_term_id, idempotency_key, submitted, server_error_type

// ─── StagedRow ───────────────────────────────────────────────────────────

export interface StagedRow {
    row_number: number; // stable client-side identity — sent as client_row_ref
    raw_data: Record<string, unknown>;
    processed_data: {
        full_name: string;
        gender: "M" | "F" | "";
        date_of_birth: string | null;
        class_id: string | null;
        grade_level: string;
        stream_name: string;
        nemis_number: string | null;
        assessment_number: string | null;
        birth_certificate_number: string | null;
    };
    ui_meta: {
        has_error: boolean;
        skipped: boolean; // true = excluded from final payload
        submitted: boolean; // true once confirmed accepted by a completed job
        errors: {
            missing_required: string | null;
            invalid_class: string | null;
            invalid_date: string | null;
            server_rejected: string | null;
            server_error_type: string | null; // SCHEMA_VALIDATION | DATABASE_CONSTRAINT | BUSINESS_RULE_VIOLATION
        };
    };
}

// ─── Class lookup cache ─────────────────────────────────────────────────

export interface ClassCacheEntry {
    id: string;
    grade_level: string;
    stream_name: string;
    display_label: string;
}
