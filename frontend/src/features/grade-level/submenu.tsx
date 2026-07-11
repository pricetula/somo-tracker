/**
 * Grade Level filter submenu — the single source of truth for grade-level
 * filter options used across all pages.
 *
 * Every data table that filters by grade level imports this function instead
 * of defining its own SubFilterItem array.
 *
 * The label renders a <GradeLevelPill /> so the colour definition lives
 * only inside the pill component — the filter dropdown never touches colour.
 */

import type { SubFilterItem } from "@/components/shared/data-table/types";
import { GRADE_LEVEL_LABELS, type GradeLevel } from "./types";
import { GradeLevelPill } from "./components/grade-level-pill";

/**
 * Returns the canonical list of SubFilterItem entries for all grade levels.
 *
 * Colour and display text are handled entirely by the GradeLevelPill
 * component — no dotColor or string label is passed to the filter.
 */
export function getGradeLevelFilterSubmenu(): SubFilterItem[] {
    return (Object.keys(GRADE_LEVEL_LABELS) as GradeLevel[]).map((grade) => ({
        id: grade.toLowerCase(),
        label: <GradeLevelPill grade={grade} />,
        value: grade,
    }));
}
