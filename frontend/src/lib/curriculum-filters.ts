/**
 * Shared curriculum filter definitions used across listing pages.
 *
 * Provides the multi-select filter groups for Education Level and Grade Level
 * (matching the multi-select architecture from the Curriculum page), plus
 * human-readable label formatters.
 */

import type { FilterGroup } from "@/components/shared/data-table/types";

// ─── Education Level Labels ───────────────────────────────────────────────

export const EDUCATION_LEVEL_LABELS: Record<string, string> = {
    Early_Years: "Early Years",
    Upper_Primary: "Upper Primary",
    Junior_Secondary: "Junior Secondary",
    Senior_School: "Senior School",
};

export function formatEducationLevel(level: string): string {
    return EDUCATION_LEVEL_LABELS[level] ?? level;
}

// ─── Grade Level Labels ───────────────────────────────────────────────────

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

export function formatGradeLevel(level: string): string {
    return GRADE_LEVEL_LABELS[level] ?? level;
}

// ─── Filter Groups ────────────────────────────────────────────────────────

const EDUCATION_LEVEL_SUBMENU = [
    { id: "early_years", label: "Early Years", value: "Early_Years" },
    { id: "upper_primary", label: "Upper Primary", value: "Upper_Primary" },
    { id: "junior_secondary", label: "Junior Secondary", value: "Junior_Secondary" },
    { id: "senior_school", label: "Senior School", value: "Senior_School" },
];

const GRADE_LEVEL_SUBMENU = [
    { id: "pp1", label: "PP1", value: "PP1" },
    { id: "pp2", label: "PP2", value: "PP2" },
    { id: "g1", label: "Grade 1", value: "G1" },
    { id: "g2", label: "Grade 2", value: "G2" },
    { id: "g3", label: "Grade 3", value: "G3" },
    { id: "g4", label: "Grade 4", value: "G4" },
    { id: "g5", label: "Grade 5", value: "G5" },
    { id: "g6", label: "Grade 6", value: "G6" },
    { id: "g7", label: "Grade 7", value: "G7" },
    { id: "g8", label: "Grade 8", value: "G8" },
    { id: "g9", label: "Grade 9", value: "G9" },
    { id: "g10", label: "Grade 10", value: "G10" },
    { id: "g11", label: "Grade 11", value: "G11" },
    { id: "g12", label: "Grade 12", value: "G12" },
];

/**
 * Curriculum filter group: multi-select dropdown for Education Levels or Grade Levels.
 * Matches the multi-select architecture from the Curriculum page.
 *
 * Use in combination with the Lifecycle Filter Group for the students page.
 */
export const CURRICULUM_FILTER_GROUPS: FilterGroup[] = [
    {
        id: "curriculum_filters",
        label: "Filter by",
        items: [
            {
                id: "education_level",
                label: "Education Level",
                type: "sub_menu_multi",
                submenu: EDUCATION_LEVEL_SUBMENU,
            },
            {
                id: "grade_level",
                label: "Grade",
                type: "sub_menu_multi",
                submenu: GRADE_LEVEL_SUBMENU,
            },
        ],
    },
];
