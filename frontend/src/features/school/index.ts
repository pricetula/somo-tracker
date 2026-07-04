/**
 * School feature — public API barrel.
 */

export { SchoolSwitcher } from "./components/school-switcher";
export { CreateSchoolDialog } from "./components/create-school-dialog";
export { CreateSchoolForm } from "./components/create-school-form";

export { useSchools, useCreateSchool, schoolKeys } from "./hooks/use-schools";

export type {
    SchoolWithMemberCount,
    ListSchoolsResponse,
    CreateSchoolPayload,
    CreateSchoolResponse,
} from "./types";
