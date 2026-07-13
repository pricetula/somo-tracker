/**
 * Academic Terms feature — public API barrel.
 *
 * External code must import ONLY from this file — never from internal paths.
 */

export { AcademicYearCombobox } from "./components/academic-year-combobox";
export { useAcademicYears, academicYearKeys } from "./hooks/use-academic-terms";

export type { AcademicYearComboboxProps } from "./components/academic-year-combobox";
export type { AcademicYear, AcademicTerm } from "./types";
