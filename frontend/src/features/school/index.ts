/**
 * School feature — public API barrel.
 * Previous exports (useSchools, useCreateSchool, etc.) are deprecated.
 */

export { SchoolSwitcher } from "./components/school-switcher";
export { CreateSchoolDialog } from "./components/create-school-dialog";
export { CreateSchoolForm } from "./components/create-school-form";
export { OnboardingForm } from "./components/onboarding";

export { useRegisterSchool, schoolRegistrationKeys } from "./hooks/use-schools";

export type { RegisterSchoolPayload, RegisterSchoolResponse } from "@/lib/api/schools";
