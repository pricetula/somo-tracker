/**
 * Academic Years feature — public API barrel.
 *
 * External code must import ONLY from this file — never from internal paths.
 */

export { AcademicYearsList } from "./components/academic-years-list";
export { AcademicYearForm } from "./components/academic-year-form";
export { AcademicYearDetail } from "./components/academic-year-detail";

export {
    useAcademicYearsManage,
    useAcademicYearMap,
    useAcademicYearDetail,
    useTermsManage,
    useCreateAcademicYear,
    useUpdateAcademicYear,
    useSetCurrentYear,
    useDeleteAcademicYear,
    useCreateTerm,
    useUpdateTerm,
    academicYearKeys,
    academicTermKeys,
} from "./hooks/use-academic-years";

export type {
    AcademicYear,
    AcademicTerm,
    CreateAcademicYearPayload,
    UpdateAcademicYearPayload,
    CreateTermPayload,
    UpdateTermPayload,
} from "./types";
