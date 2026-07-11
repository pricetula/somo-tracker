/**
 * Grade Level feature — public API barrel.
 *
 * External code must import ONLY from this file — never from internal paths.
 */

export { GradeLevelPill } from "./components/grade-level-pill";

export type { GradeLevel } from "./types";
export { GRADE_LEVEL_LABELS, GRADE_LEVEL_STYLES, GRADE_LEVEL_DOT_COLORS } from "./types";
export { getGradeLevelFilterSubmenu } from "./submenu";
