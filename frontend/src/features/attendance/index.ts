/**
 * Attendance feature — public API barrel.
 *
 * Import only from this barrel, never from internal paths.
 */

// Components
export { SchoolAttendanceKPIs } from "./components/school-attendance-kpis";
export { ClassAttendanceBreakdownChart } from "./components/class-attendance-breakdown-chart";

// Hooks
export { useSchoolAttendanceKPIs } from "./hooks/use-school-attendance-kpis";
export {
    useClassAttendanceBreakdowns,
    useCurrentTermId,
} from "./hooks/use-class-attendance-breakdowns";
