/**
 * Academic Years feature — public API barrel.
 *
 * External code must import ONLY from this file — never from internal paths.
 */

export { AcademicYearsList } from "./components/academic-years-list";
export { AcademicYearDetail } from "./components/academic-year-detail";

export {
    useAcademicYearsManage,
    useAcademicYearMap,
    useAcademicYearDetail,
    useTermsManage,
    useCreateTerm,
    useUpdateTerm,
    useActivateTerm,
    useDeleteTerm,
    academicYearKeys,
    academicTermKeys,
} from "./hooks/use-academic-years";

export type { AcademicYear, AcademicTerm, CreateTermPayload, UpdateTermPayload } from "./types";
