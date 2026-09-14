/**
 * School feature — public API barrel.
 * Previous exports (useSchools, useCreateSchool, etc.) are deprecated.
 */

export { Onboarding } from "./components/onboarding";
export { CreateStreams } from "./components/create-streams";
export { SchoolsTable } from "./components/schools-list";
export { CreateSchoolForm } from "./components/create-school-form";

export { useRegisterSchool, schoolRegistrationKeys } from "./hooks/use-schools";
export { useCreateStreams, useGrades, streamKeys } from "./hooks/use-streams";

export type { RegisterSchoolPayload, RegisterSchoolResponse } from "@/lib/api/schools";
