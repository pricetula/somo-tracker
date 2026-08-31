/**
 * Attendance feature — public API barrel.
 *
 * Import only from this barrel, never from internal paths.
 */

// Components
export { AttendanceSummary } from "./components/attendance-summary";
export { AttendanceCalendar } from "./components/attendance-calendar";
export { LowestAttendanceStudents } from "./components/lowest-attendance-students";
export { ClassAttendanceBreakdownChart } from "./components/class-attendance-breakdown-chart";
export { LearningAreaAbsenteeismChart } from "./components/learning-area-absenteeism-chart";
export { WeekdayAttendanceExceptionsChart } from "./components/weekday-attendance-exceptions-chart";

// Hooks
export { useSchoolAttendanceKPIs } from "./hooks/use-school-attendance-kpis";
export { useClassAttendanceBreakdowns } from "./hooks/use-class-attendance-breakdowns";
export { useLearningAreaAttendanceBreakdowns } from "./hooks/use-learning-area-attendance-breakdowns";
export { useDayOfWeekSummaries } from "./hooks/use-day-of-week-summaries";
export { useLowestAttendanceStudents } from "./hooks/use-lowest-attendance-students";
