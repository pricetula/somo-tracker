/**
 * Education Level filter submenu — the single source of truth for education
 * level filter options used across all pages.
 *
 * Every data table that filters by education level imports this function
 * instead of defining its own SubFilterItem array.
 */

import type { SubFilterItem } from "@/components/shared/data-table/types";
import { EDUCATION_LEVEL_DOT_COLORS, EDUCATION_LEVEL_LABELS, type EducationLevel } from "./types";

/**
 * Returns the canonical list of SubFilterItem entries for all education levels.
 *
 * The dotColor values are derived from EDUCATION_LEVEL_DOT_COLORS so the filter
 * dropdown stays in sync with the EducationLevelPill component.
 */
export function getEducationLevelFilterSubmenu(): SubFilterItem[] {
    return (Object.keys(EDUCATION_LEVEL_LABELS) as EducationLevel[]).map((level) => ({
        id: level.toLowerCase(),
        label: EDUCATION_LEVEL_LABELS[level],
        value: level,
        dotColor: EDUCATION_LEVEL_DOT_COLORS[level],
    }));
}
