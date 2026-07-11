/**
 * Shared curriculum filter definitions used across listing pages.
 *
 * Provides the FilterGroup for Education Level and Grade Level
 * (matching the multi-select architecture from the Curriculum page).
 *
 * Colour and label definitions are sourced from the dedicated feature
 * modules (features/education-level and features/grade-level) which
 * serve as the single source of truth — never duplicate them here.
 */

import type { FilterGroup } from "@/components/shared/data-table/types";
import { getEducationLevelFilterSubmenu } from "@/features/education-level";
import { getGradeLevelFilterSubmenu } from "@/features/grade-level";
import { GraduationCap, BookOpen } from "lucide-react";

// ─── Education Level Labels (re-exported for convenience) ───────────────

export { EDUCATION_LEVEL_LABELS } from "@/features/education-level/types";

// ─── Grade Level Labels (re-exported for convenience) ───────────────────

export { GRADE_LEVEL_LABELS } from "@/features/grade-level/types";

// ─── Filter Groups ────────────────────────────────────────────────────────

/**
 * Curriculum filter group: multi-select dropdown for Education Levels or Grade Levels.
 *
 * Submenu items (labels, dot colours) are sourced from the canonical feature
 * modules — never define them inline.
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
                icon: BookOpen,
                type: "sub_menu_multi",
                submenu: getEducationLevelFilterSubmenu(),
            },
            {
                id: "grade_level",
                label: "Grade",
                icon: GraduationCap,
                type: "sub_menu_multi",
                submenu: getGradeLevelFilterSubmenu(),
            },
        ],
    },
];
