/**
 * Attendance Term Summaries — Section 2.1 components barrel.
 */

export { AttendanceGauge, AttendanceGaugeSkeleton } from "./attendance-gauge";

export {
    AttendanceWeeklyStackedBar,
    AttendanceWeeklyStackedBarSkeleton,
} from "./attendance-weekly-stacked-bar";
export type { WeeklyAttendanceRow } from "./attendance-weekly-stacked-bar";

export { AttendanceHeatmap, AttendanceHeatmapSkeleton } from "./attendance-heatmap";
export type { HeatmapCell, AttendanceHeatmapData } from "./attendance-heatmap";

export {
    AttendanceTermTrendLine,
    AttendanceTermTrendLineSkeleton,
} from "./attendance-term-trend-line";
export type { TermAttendancePoint } from "./attendance-term-trend-line";

export {
    AttendanceSubjectComparisonBar,
    AttendanceSubjectComparisonBarSkeleton,
} from "./attendance-subject-comparison-bar";
export type { SubjectAttendanceEntry } from "./attendance-subject-comparison-bar";

export {
    AttendanceVsOverallScatter,
    AttendanceVsOverallScatterSkeleton,
} from "./attendance-vs-overall-scatter";
export type { AttendanceVsPerformancePoint } from "./attendance-vs-overall-scatter";
