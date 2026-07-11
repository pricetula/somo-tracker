/**
 * Education Level filter submenu — the single source of truth for education
 * level filter options used across all pages.
 *
 * Every data table that filters by education level imports this function
 * instead of defining its own SubFilterItem array.
 *
 * The label renders an <EducationLevelPill /> so the colour definition lives
 * only inside the pill component — the filter dropdown never touches colour.
 */

import type { SubFilterItem } from "@/components/shared/data-table/types";
import { EDUCATION_LEVEL_LABELS, type EducationLevel } from "./types";
import { EducationLevelPill } from "./components/education-level-pill";

/**
 * Returns the canonical list of SubFilterItem entries for all education levels.
 *
 * Colour and display text are handled entirely by the EducationLevelPill
 * component — no dotColor or string label is passed to the filter.
 */
export function getEducationLevelFilterSubmenu(): SubFilterItem[] {
    return (Object.keys(EDUCATION_LEVEL_LABELS) as EducationLevel[]).map((level) => ({
        id: level.toLowerCase(),
        label: <EducationLevelPill level={level} />,
        value: level,
    }));
}
