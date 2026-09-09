/**
 * School feature — public API barrel.
 * Previous exports (useSchools, useCreateSchool, etc.) are deprecated.
 */

export { Onboarding } from "./components/onboarding";

export { useRegisterSchool, schoolRegistrationKeys } from "./hooks/use-schools";

export type { RegisterSchoolPayload, RegisterSchoolResponse } from "@/lib/api/schools";
