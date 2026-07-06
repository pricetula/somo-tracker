/**
 * Assessment feature — TypeScript types.
 *
 * Mirrors the backend response/payload shapes for blueprints, sessions, results, and weight configs.
 * Used by the API client (src/lib/api/assessment.ts) and any future assessment feature components.
 */

// ─── Blueprints ───────────────────────────────────────────────────────────

export interface Blueprint {
    id: string;
    school_id: string;
    title: string;
    type: string;
    grade_level: string;
    academic_year: number;
    term: number;
    created_at: string;
    updated_at: string;
}

export interface BlueprintLinkedIndicator {
    id: string;
    performance_indicator_id: string;
    description: string;
    sub_strand: string;
    strand: string;
    learning_area: string;
}

export interface CreateBlueprintPayload {
    title: string;
    type: string;
    grade_level: string;
    academic_year: number;
    term: number;
}

export interface CreateBlueprintResponse {
    id: string;
}

export interface ListBlueprintsResponse {
    items: Blueprint[];
    total: number;
}

export interface BlueprintDetailResponse {
    blueprint: Blueprint;
    indicators: BlueprintLinkedIndicator[];
}

// ─── Sessions ─────────────────────────────────────────────────────────────

export interface AssessmentSession {
    id: string;
    school_id: string;
    blueprint_id: string;
    class_id: string;
    date_administered: string;
    status: string;
    knec_upload_reference?: string | null;
    created_at: string;
    updated_at: string;
}

export interface CreateSessionPayload {
    blueprint_id: string;
    class_id: string;
    date_administered: string;
}

export interface CreateSessionResponse {
    id: string;
}

export interface ListSessionsResponse {
    items: AssessmentSession[];
    total: number;
}

export interface SessionDetailResponse {
    session: AssessmentSession;
    blueprint: Blueprint;
}

// ─── Results ──────────────────────────────────────────────────────────────

export interface RubricResult {
    id: string;
    session_id: string;
    student_id: string;
    performance_indicator_id: string;
    score: number;
    max_score: number;
    remark?: string;
    created_at: string;
    updated_at: string;
}

export interface BatchUpsertResultsPayload {
    results: Array<{
        student_id: string;
        performance_indicator_id: string;
        score: number;
        max_score: number;
        remark?: string;
    }>;
}

export interface ListResultsResponse {
    items: RubricResult[];
    total: number;
}

// ─── Weight Configs ───────────────────────────────────────────────────────

export interface WeightConfig {
    id: string;
    grade_level: string;
    target_exam: string;
    learning_area_id: string;
    weight: number;
}

export interface ListWeightConfigsResponse {
    items: WeightConfig[];
    total: number;
}
