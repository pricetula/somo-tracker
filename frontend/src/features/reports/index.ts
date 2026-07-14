/**
 * Reports feature — public API barrel.
 *
 * External code must import ONLY from this file — never from internal paths.
 */

// Components
export { ParentTermReport } from "./components/parent-term-report";
export { AdminTermReportManager } from "./components/admin-term-report-manager";

// Hooks
export {
    useTermReport,
    useTermReportList,
    useGenerateTermReport,
    useGenerateClassReports,
    usePublishTermReport,
    reportKeys,
} from "./hooks/use-reports";

// Types
export type {
    TermReportBehaviorNote,
    TermReportAttendance,
    TermReportCompetency,
    TermReport,
    TermReportGenerateResponse,
} from "./types";
