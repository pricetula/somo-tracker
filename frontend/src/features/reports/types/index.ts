/**
 * Reports feature types.
 *
 * These mirror the backend API response shapes. The canonical definitions
 * live in src/lib/api/reports.ts; this barrel re-exports them so feature
 * consumers can import from @/features/reports/types.
 */

export type {
    TermReportBehaviorNote,
    TermReportAttendance,
    TermReportCompetency,
    TermReport,
    TermReportGenerateResponse,
} from "@/lib/api/reports";
