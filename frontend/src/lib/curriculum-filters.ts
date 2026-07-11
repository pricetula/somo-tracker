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
    { id: "early_years", label: "Early Years", value: "Early_Years", dotColor: "#059669" },
    { id: "upper_primary", label: "Upper Primary", value: "Upper_Primary", dotColor: "#0284c7" },
    {
        id: "junior_secondary",
        label: "Junior Secondary",
        value: "Junior_Secondary",
        dotColor: "#d97706",
    },
    { id: "senior_school", label: "Senior School", value: "Senior_School", dotColor: "#7c3aed" },
];

const GRADE_LEVEL_SUBMENU = [
    { id: "pp1", label: "PP1", value: "PP1", dotColor: "#d1fae5" },
    { id: "pp2", label: "PP2", value: "PP2", dotColor: "#a7f3d0" },
    { id: "g1", label: "Grade 1", value: "G1", dotColor: "#6ee7b7" },
    { id: "g2", label: "Grade 2", value: "G2", dotColor: "#34d399" },
    { id: "g3", label: "Grade 3", value: "G3", dotColor: "#059669" },
    { id: "g4", label: "Grade 4", value: "G4", dotColor: "#7dd3fc" },
    { id: "g5", label: "Grade 5", value: "G5", dotColor: "#38bdf8" },
    { id: "g6", label: "Grade 6", value: "G6", dotColor: "#0284c7" },
    { id: "g7", label: "Grade 7", value: "G7", dotColor: "#fcd34d" },
    { id: "g8", label: "Grade 8", value: "G8", dotColor: "#fbbf24" },
    { id: "g9", label: "Grade 9", value: "G9", dotColor: "#d97706" },
    { id: "g10", label: "Grade 10", value: "G10", dotColor: "#c4b5fd" },
    { id: "g11", label: "Grade 11", value: "G11", dotColor: "#a78bfa" },
    { id: "g12", label: "Grade 12", value: "G12", dotColor: "#7c3aed" },
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
