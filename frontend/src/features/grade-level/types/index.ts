/**
 * Grade Level feature — type definitions.
 *
 * The GradeLevel union type is sourced from lib/api/generated.ts (the
 * canonical API-generated type) — never redefine it here.
 */

import type { GradeLevel } from "@/lib/api/generated";

export type { GradeLevel };

/** Human-readable labels for each grade level. */
export const GRADE_LEVEL_LABELS: Record<GradeLevel, string> = {
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
 * Dot colour (Tailwind class) for each grade level — 14 independent colour variations.
 *
 * Used by the GradeLevelPill component. Each grade gets its own hue so they
 * can be distinguished independently of education-level grouping.
 */
export const GRADE_LEVEL_STYLES: Record<GradeLevel, { dot: string }> = {
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
