// ─── Domain Types ───────────────────────────────────────────────────────────

/** A single classroom entity from the backend. */
export interface ClassItem {
    id?: string;
    tenant_id?: string;
    school_id?: string;
    academic_year_id?: string;
    education_system_id?: string;
    grade_id?: string;
    name: string;
    stream?: string;
    is_active: boolean;
    created_at?: string;
}

/** A grade record from the backend. */
export interface Grade {
    id: string;
    name: string;
    sequence_order: number;
}

/** Params for listing classes with filters. */
export interface ClassListParams {
    grade_ids?: string[];
    search?: string;
    is_active?: boolean;
}

/** Payload sent to POST /api/v1/schools/classes/generate. */
export interface GeneratePayload {
    streams: string[];
}

/** Response from the generate endpoint. */
export interface GenerateResult {
    classes: ClassItem[];
    total_created: number;
    streams: string[];
    grade_names: string[];
}

// ─── Evaluator Result ─────────────────────────────────────────────────────

export type ClassStreamState =
    | { type: "loading" }
    | { type: "setup" } // No classes exist → show the generator
    | { type: "ready" }; // Classes exist → collapse Step 2

// ─── Static CBE Defaults ──────────────────────────────────────────────────

/**
 * Standard Kenyan CBC baseline tiers used during onboarding step 2.
 * The cross-multiplication engine combines these with user-defined
 * stream tags to generate the full classroom grid.
 */
export const CBE_GRADE_TIERS = ["Grade 1", "Grade 2", "Grade 3", "Grade 4"];
