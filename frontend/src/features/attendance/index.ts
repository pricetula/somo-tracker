/**
 * Attendance feature — public API barrel.
 *
 * Import only from this barrel, never from internal paths.
 */

// Components
export { AttendanceSummary } from "./components/attendance-summary";
export { ClassAttendanceBreakdownChart } from "./components/class-attendance-breakdown-chart";
export { LearningAreaAbsenteeismChart } from "./components/learning-area-absenteeism-chart";

// Hooks
export { useSchoolAttendanceKPIs } from "./hooks/use-school-attendance-kpis";
export {
    useClassAttendanceBreakdowns,
    useCurrentTermId,
} from "./hooks/use-class-attendance-breakdowns";
export {
    useLearningAreaAttendanceBreakdowns,
    useCurrentTermId as useLearningAreaCurrentTermId,
} from "./hooks/use-learning-area-attendance-breakdowns";
