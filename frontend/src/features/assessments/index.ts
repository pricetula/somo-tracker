/**
 * Assessments feature — public API barrel.
 *
 * External code must import ONLY from this file — never from internal paths.
 */

// Types
export type {
    ScaleProfile,
    ScaleRange,
    ScaleProfileWithRanges,
    AssessmentSession,
    StudentScore,
    OutcomeGrade,
    ParentAssessmentView,
    StudentTermGrade,
    AssessmentWeightConfig,
    CreateWeightConfigPayload,
    WeightConfigListResult,
    CreateScaleProfilePayload,
    ScaleRangePayload,
    BulkSetRangesPayload,
    CreateSessionPayload,
    StudentScorePayload,
    OutcomeGradePayload,
    BulkUpsertScoresPayload,
    BulkUpsertOutcomeGradesPayload,
    ListSessionsParams,
} from "./types";

export {
    PERFORMANCE_LEVEL_LABELS,
    SESSION_STATUS_LABELS,
    SESSION_STATUS_COLORS,
    EVALUATION_METHOD_LABELS,
    PERFORMANCE_LEVELS,
} from "./types";

// Hooks
export {
    useScaleProfileList,
    useScaleProfile,
    useScaleRanges,
    useCreateScaleProfile,
    useToggleScaleProfile,
    useDeleteScaleProfile,
    useBulkSetScaleRanges,
    useSessionList,
    useSession,
    useStudentScores,
    useOutcomeGrades,
    useCreateSession,
    useSubmitSession,
    useApproveSession,
    useRejectSession,
    useDeleteSession,
    useDeleteWeightConfig,
    useBulkUpsertScores,
    useBulkUpsertOutcomeGrades,
    useParentAssessments,
    useStudentTermGrades,
    useWeightConfigList,
    useCreateWeightConfig,
    assessmentKeys,
} from "./hooks/use-assessments";

// Components
export { PerformanceLevelBadge } from "./components/performance-level-badge";
export { StatusBadge } from "./components/status-badge";
export { GradingScaleProfilesList } from "./components/grading-scale-profiles-list";
export { CreateScaleProfileForm } from "./components/create-scale-profile-form";
export { ScaleProfileDetailView } from "./components/scale-profile-detail-view";
export { SetScaleRangesForm } from "./components/set-scale-ranges-form";
export { AssessmentSessionsList } from "./components/assessment-sessions-list";
export { CreateAssessmentSessionForm } from "./components/create-assessment-session-form";
export { AssessmentSessionDetailView } from "./components/assessment-session-detail-view";
export { ApprovalActions } from "./components/approval-actions";
export { GradingSheet } from "./components/grading-sheet";
export { RubricGradingMatrix } from "./components/rubric-grading-matrix";
export { ParentAssessmentsView } from "./components/parent-assessments-view";
export { WeightConfigsList } from "./components/weight-configs-list";
