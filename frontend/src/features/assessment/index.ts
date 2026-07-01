/**
 * Assessment feature — public API barrel.
 */

export { BlueprintsTable } from "./components/blueprints-table";
export { BlueprintForm } from "./components/blueprint-form";
export { IndicatorLinker } from "./components/indicator-linker";
export { SessionsTable } from "./components/sessions-table";
export { SessionForm } from "./components/session-form";
export { ScoringGrid } from "./components/scoring-grid";

export {
    useBlueprints,
    useBlueprintDetail,
    useCreateBlueprint,
    useDeleteBlueprint,
    useLinkIndicators,
    useUnlinkIndicator,
    useSessions,
    useSessionDetail,
    useCreateSession,
    useDeleteSession,
    useSessionResults,
    useBatchUpsertResults,
    useWeightConfigs,
    assessmentKeys,
} from "./hooks/use-assessment";

export type {
    AssessmentBlueprint,
    BlueprintDetail,
    LinkedIndicator,
    CreateBlueprintPayload,
    AssessmentSession,
    LearnerRubricResult,
    SessionDetail,
    CreateSessionPayload,
    BatchResultInput,
    BatchUpsertResultsPayload,
    AssessmentWeightConfig,
    AssessmentType,
    GradeLevel,
    RubricLevel,
    ScoreType,
} from "./types";

export { ASSESSMENT_TYPES, GRADE_LEVELS, RUBRIC_LEVELS, SCORE_TYPES } from "./types";
