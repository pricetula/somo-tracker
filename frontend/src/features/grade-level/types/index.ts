/**
 * Grade Level feature — type definitions.
 */

/** Canonical grade level values used across the system. */
export type GradeLevel =
    | "PP1"
    | "PP2"
    | "G1"
    | "G2"
    | "G3"
    | "G4"
    | "G5"
    | "G6"
    | "G7"
    | "G8"
    | "G9"
    | "G10"
    | "G11"
    | "G12";

/** Human-readable labels for each grade level. */
export const GRADE_LEVEL_LABELS: Record<string, string> = {
    PP1: "PP1",
    PP2: "PP2",
    G1: "Grade 1",
    G2: "Grade 2",
    G3: "Grade 3",
    G4: "Grade 4",
    G5: "Grade 5",
    G6: "Grade 6",
    G7: "Grade 7",
    G8: "Grade 8",
    G9: "Grade 9",
    G10: "Grade 10",
    G11: "Grade 11",
    G12: "Grade 12",
};

/**
 * Dot colours for each grade level — 14 independent colour variations.
 *
 * Only the leading dot is coloured; the pill label remains neutral.
 * Each grade gets its own hue so they can be distinguished independently
 * of education-level grouping.
 */
export const GRADE_LEVEL_STYLES: Record<string, { dot: string }> = {
    PP1: { dot: "bg-emerald-100" },
    PP2: { dot: "bg-emerald-200" },
    G1: { dot: "bg-emerald-300" },
    G2: { dot: "bg-emerald-400" },
    G3: { dot: "bg-emerald-600" },
    G4: { dot: "bg-sky-300" },
    G5: { dot: "bg-sky-400" },
    G6: { dot: "bg-sky-600" },
    G7: { dot: "bg-amber-300" },
    G8: { dot: "bg-amber-400" },
    G9: { dot: "bg-amber-600" },
    G10: { dot: "bg-violet-300" },
    G11: { dot: "bg-violet-400" },
    G12: { dot: "bg-violet-600" },
};
