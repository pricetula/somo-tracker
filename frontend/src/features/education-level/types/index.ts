/**
 * Education Level feature — type definitions.
 *
 * The EducationLevel union type is sourced from lib/api/generated.ts (the
 * canonical API-generated type) — never redefine it here.
 */

import type { EducationLevel } from "@/lib/api/generated";

export type { EducationLevel };

/** Human-readable labels for each education level. */
export const EDUCATION_LEVEL_LABELS: Record<EducationLevel, string> = {
    Early_Years: "Early Years",
    Upper_Primary: "Upper Primary",
    Junior_Secondary: "Junior Secondary",
    Senior_School: "Senior School",
};

/**
 * Dot colour for each education level.
 *
 * Only the leading dot is coloured — the pill label remains neutral.
 */
export const EDUCATION_LEVEL_STYLES: Record<EducationLevel, { dot: string }> = {
    Early_Years: { dot: "bg-emerald-600" },
    Upper_Primary: { dot: "bg-sky-600" },
    Junior_Secondary: { dot: "bg-amber-600" },
    Senior_School: { dot: "bg-violet-600" },
};
