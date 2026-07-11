/**
 * Grade Level filter submenu — the single source of truth for grade-level
 * filter options used across all pages.
 *
 * Every data table that filters by grade level imports this function instead
 * of defining its own SubFilterItem array (which would risk duplicating
 * colour values already defined in ./types).
 */

import type { SubFilterItem } from "@/components/shared/data-table/types";
import { GRADE_LEVEL_DOT_COLORS, GRADE_LEVEL_LABELS, type GradeLevel } from "./types";

/**
 * Returns the canonical list of SubFilterItem entries for all grade levels.
 *
 * The dotColor values are derived from GRADE_LEVEL_DOT_COLORS so the filter
 * dropdown stays in sync with the GradeLevelPill component.
 */
export function getGradeLevelFilterSubmenu(): SubFilterItem[] {
    return (Object.keys(GRADE_LEVEL_LABELS) as GradeLevel[]).map((grade) => ({
        id: grade.toLowerCase(),
        label: GRADE_LEVEL_LABELS[grade],
        value: grade,
        dotColor: GRADE_LEVEL_DOT_COLORS[grade],
    }));
}
