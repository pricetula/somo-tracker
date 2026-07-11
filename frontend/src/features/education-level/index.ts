/**
 * Education Level feature — public API barrel.
 *
 * External code must import ONLY from this file — never from internal paths.
 */

export { EducationLevelPill } from "./components/education-level-pill";

export type { EducationLevel } from "./types";
export {
    EDUCATION_LEVEL_LABELS,
    EDUCATION_LEVEL_STYLES,
    EDUCATION_LEVEL_DOT_COLORS,
} from "./types";
export { getEducationLevelFilterSubmenu } from "./submenu";
