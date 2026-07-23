/**
 * Assessment feature types.
 *
 * Re-exports from the API client for convenience. Feature-internal types
 * (not related to backend shapes) go here.
 */

export type {
    ScaleProfile,
    ScaleRange,
    ScaleProfileWithRanges,
    AssessmentSession,
    StudentScore,
    OutcomeGrade,
    ParentAssessmentView,
    StudentTermGrade,
    CreateScaleProfilePayload,
    ScaleRangePayload,
    BulkSetRangesPayload,
    CreateSessionPayload,
    StudentScorePayload,
    OutcomeGradePayload,
    BulkUpsertScoresPayload,
    BulkUpsertOutcomeGradesPayload,
    RejectSessionPayload,
    ScaleProfileListResult,
    ScaleRangesListResult,
    SessionListResult,
    StudentScoresResult,
    OutcomeGradesResult,
    ParentAssessmentsResult,
    StudentTermGradesResult,
    ListSessionsParams,
} from "@/lib/api/assessments";

/** Human-readable labels for CBC performance levels. */
export const PERFORMANCE_LEVEL_LABELS: Record<string, string> = {
    EE: "Exceeding Expectation",
    ME: "Meeting Expectation",
    AE: "Approaching Expectation",
    BE: "Below Expectation",
};

/** Human-readable labels for assessment session statuses. */
export const SESSION_STATUS_LABELS: Record<string, string> = {
    DRAFT: "Draft",
    PENDING_APPROVAL: "Pending Approval",
    PUBLISHED: "Published",
};

/** Colours for status badges. */
export const SESSION_STATUS_COLORS: Record<string, string> = {
    DRAFT: "bg-muted text-muted-foreground",
    PENDING_APPROVAL: "bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400",
    PUBLISHED: "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400",
};

/** Human-readable labels for evaluation methods. */
export const EVALUATION_METHOD_LABELS: Record<string, string> = {
    QUANTITATIVE: "Marks-Based",
    RUBRIC: "Rubric (Indicator-Level)",
};

/** Valid CBC performance levels in order (lowest → highest). */
export const PERFORMANCE_LEVELS = ["BE", "AE", "ME", "EE"] as const;
