/**
 * Education Level feature — type definitions.
 */

/** Canonical education level values used across the system. */
export type EducationLevel = "Early_Years" | "Upper_Primary" | "Junior_Secondary" | "Senior_School";

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
