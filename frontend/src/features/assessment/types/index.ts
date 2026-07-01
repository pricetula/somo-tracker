/**
 * TypeScript interfaces for the Assessment feature.
 *
 * Maps to backend internal/assessment/domain.go
 */

// ─── Blueprints ───────────────────────────────────────────────────────────

export interface AssessmentBlueprint {
    id: string;
    school_id: string;
    title: string;
    type: string;
    grade_level: string;
    academic_year: number;
    term: number;
    created_at: string;
}

export interface LinkedIndicator {
    id: string;
    description: string;
}

export interface BlueprintDetail {
    id: string;
    school_id: string;
    title: string;
    type: string;
    grade_level: string;
    academic_year: number;
    term: number;
    created_at: string;
    indicators: LinkedIndicator[];
}

export interface CreateBlueprintPayload {
    title: string;
    type: string;
    grade_level: string;
    academic_year: number;
    term: number;
}

export interface ListBlueprintsResponse {
    data: AssessmentBlueprint[];
}

export interface BlueprintDetailResponse {
    data: BlueprintDetail;
}

export interface CreateBlueprintResponse {
    id: string;
}

// ─── Sessions ─────────────────────────────────────────────────────────────

export interface AssessmentSession {
    id: string;
    blueprint_id: string;
    class_id: string;
    assessed_by_user_id: string;
    date_administered: string;
    knec_upload_reference?: string | null;
    created_at: string;
}

export interface LearnerRubricResult {
    id: string;
    session_id: string;
    student_id: string;
    indicator_id: string;
    score_type: string;
    raw_score?: string | null;
    rubric_level: string;
    teacher_observation_notes?: string | null;
}

export interface SessionDetail {
    id: string;
    blueprint_id: string;
    class_id: string;
    assessed_by_user_id: string;
    date_administered: string;
    knec_upload_reference?: string | null;
    created_at: string;
    results: LearnerRubricResult[];
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
    data: AssessmentSession[];
}

export interface SessionDetailResponse {
    data: SessionDetail;
}

// ─── Results ──────────────────────────────────────────────────────────────

export interface BatchResultInput {
    student_id: string;
    indicator_id: string;
    score_type: string;
    raw_score?: string | null;
    rubric_level: string;
    teacher_observation_notes?: string | null;
}

export interface BatchUpsertResultsPayload {
    results: BatchResultInput[];
}

export interface ListResultsResponse {
    data: LearnerRubricResult[];
}

// ─── Weight Configs ───────────────────────────────────────────────────────

export interface AssessmentWeightConfig {
    id: string;
    grade_level: string;
    assessment_type_code: string;
    target_exam: string;
    weight_percent: string;
    effective_from: number;
}

export interface ListWeightConfigsResponse {
    data: AssessmentWeightConfig[];
}

// ─── Enums / Constants ────────────────────────────────────────────────────

export const ASSESSMENT_TYPES = [
    "Formative_Classroom",
    "KNEC_Written_Assessment",
    "KNEC_SBA_Project",
    "National_KPSEA",
    "National_KJSEA",
    "National_KSSEA",
] as const;

export const GRADE_LEVELS = [
    "PP1",
    "PP2",
    "G1",
    "G2",
    "G3",
    "G4",
    "G5",
    "G6",
    "G7",
    "G8",
    "G9",
    "G10",
    "G11",
    "G12",
] as const;

export const RUBRIC_LEVELS = ["EE", "ME", "AE", "BE"] as const;

export const SCORE_TYPES = ["Numeric_Raw", "Rubric_Direct"] as const;

export type AssessmentType = (typeof ASSESSMENT_TYPES)[number];
export type GradeLevel = (typeof GRADE_LEVELS)[number];
export type RubricLevel = (typeof RUBRIC_LEVELS)[number];
export type ScoreType = (typeof SCORE_TYPES)[number];
