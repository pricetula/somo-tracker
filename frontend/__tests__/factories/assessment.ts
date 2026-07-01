/**
 * Factory for building assessment-related test objects.
 */

import type {
    AssessmentBlueprint,
    AssessmentSession,
    LearnerRubricResult,
    BlueprintDetail,
} from "@/features/assessment/types";

let counter = 0;

export function buildBlueprint(overrides?: Partial<AssessmentBlueprint>): AssessmentBlueprint {
    counter++;
    return {
        id: `blueprint-${String(counter).padStart(3, "0")}`,
        school_id: "school-xyz",
        title: `Assessment Blueprint ${counter}`,
        type: "Formative_Classroom",
        grade_level: "G4",
        academic_year: 2026,
        term: 1,
        created_at: "2026-06-30T08:00:00Z",
        ...overrides,
    };
}

export function buildBlueprintDetail(
    overrides?: Partial<BlueprintDetail>,
    indicatorCount = 3
): BlueprintDetail {
    const blueprint = buildBlueprint(overrides);
    return {
        ...blueprint,
        indicators: Array.from({ length: indicatorCount }, (_, i) => ({
            id: `indicator-${i + 1}`,
            description: `Performance indicator ${i + 1}: demonstrates understanding of key concepts`,
        })),
        ...overrides,
    };
}

export function buildSession(overrides?: Partial<AssessmentSession>): AssessmentSession {
    counter++;
    return {
        id: `session-${String(counter).padStart(3, "0")}`,
        blueprint_id: "blueprint-001",
        class_id: "class-001",
        assessed_by_user_id: "user-xyz",
        date_administered: "2026-06-30",
        knec_upload_reference: null,
        created_at: "2026-06-30T10:00:00Z",
        ...overrides,
    };
}

export function buildResult(overrides?: Partial<LearnerRubricResult>): LearnerRubricResult {
    counter++;
    const levels = ["EE", "ME", "AE", "BE"];
    return {
        id: `result-${String(counter).padStart(3, "0")}`,
        session_id: "session-001",
        student_id: "student-001",
        indicator_id: "indicator-001",
        score_type: "Rubric_Direct",
        raw_score: null,
        rubric_level: levels[Math.floor(Math.random() * levels.length)],
        teacher_observation_notes: null,
        ...overrides,
    };
}

export function resetCounters() {
    counter = 0;
}
