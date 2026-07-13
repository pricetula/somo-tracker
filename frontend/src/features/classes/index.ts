/**
 * Classes feature — public API barrel.
 *
 * External code must import ONLY from this file — never from internal paths.
 */

export { AddClassForm } from "./components/add-class-form";
export { ClassCombobox } from "./components/class-combobox";
export { ClassDetailView, ClassDetailSkeleton } from "./components/class-detail-view";
export {
    ClassRoster,
    RosterTable,
    RosterSkeleton,
    useClassRoster,
} from "./components/class-roster";
export { EnrollStudentsPanel } from "./components/enroll-students-panel";
export { useClassList, classKeys } from "./hooks/use-classes";

export type { ClassComboboxProps } from "./components/class-combobox";
export type { ClassOption, Class } from "./types";
export type {
    RosterEntry,
    AvailableStudent,
    AvailableStudentsResponse,
    BatchEnrollResponse,
} from "@/lib/api/classes";
