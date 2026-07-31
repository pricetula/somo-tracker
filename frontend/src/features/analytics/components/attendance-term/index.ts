/**
 * Attendance Term Summaries — Section 2.1 components barrel.
 */

export { AttendanceGauge } from "./attendance-gauge";
export { AttendanceGaugeSkeleton } from "./attendance-gauge-skeleton";

export { AttendanceWeeklyStackedBar } from "./attendance-weekly-stacked-bar";
export { AttendanceWeeklyStackedBarSkeleton } from "./attendance-weekly-stacked-bar-skeleton";
export type { WeeklyAttendanceRow } from "./attendance-weekly-stacked-bar";

export { AttendanceHeatmap } from "./attendance-heatmap";
export { AttendanceHeatmapSkeleton } from "./attendance-heatmap-skeleton";
export type { HeatmapCell, AttendanceHeatmapData } from "./attendance-heatmap";

export { AttendanceTermTrendLine } from "./attendance-term-trend-line";
export { AttendanceTermTrendLineSkeleton } from "./attendance-term-trend-line-skeleton";
export type { TermAttendancePoint } from "./attendance-term-trend-line";

export { AttendanceSubjectComparisonBar } from "./attendance-subject-comparison-bar";
export { AttendanceSubjectComparisonBarSkeleton } from "./attendance-subject-comparison-bar-skeleton";
export type { SubjectAttendanceEntry } from "./attendance-subject-comparison-bar";

export { AttendanceVsOverallScatter } from "./attendance-vs-overall-scatter";
export { AttendanceVsOverallScatterSkeleton } from "./attendance-vs-overall-scatter-skeleton";
export type { AttendanceVsPerformancePoint } from "./attendance-vs-overall-scatter";
