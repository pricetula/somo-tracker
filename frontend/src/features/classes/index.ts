/**
 * Classes feature — public API barrel.
 *
 * External code must import ONLY from this file — never from internal paths.
 */

export { ClassCombobox } from "./components/class-combobox";
export { useClassList, classKeys } from "./hooks/use-classes";

export type { ClassComboboxProps } from "./components/class-combobox";
export type { ClassOption, Class } from "./types";
