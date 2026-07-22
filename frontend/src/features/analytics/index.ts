/**
 * Analytics feature — public API barrel.
 *
 * All components, hooks, and types for summary table visualisations.
 * Import only from this barrel, never from internal paths.
 */

// Types
export type {
    PerformanceLevel,
    RiskLevel,
    AttendanceTermSummary,
    ClassDailyAttendanceSummary,
    StudentTermSubjectSummary,
    StudentTermOverallSummary,
    StudentCohortPositionSummary,
    StudentSubjectStrandSummary,
    StudentPerformanceProjection,
    HistoricalScore,
    AnalyticsFilter,
} from "./types";

// Hooks
export {
    useAttendanceTermSummaries,
    useClassDailyAttendance,
    useStudentTermSubjectSummaries,
    useStudentTermOverallSummaries,
    useStudentCohortPosition,
    useStudentSubjectStrandSummaries,
    useStudentProjections,
} from "./hooks";

// Components — Section 2.1 (Attendance Term)
export {
    AttendanceGauge,
    AttendanceGaugeSkeleton,
    AttendanceWeeklyStackedBar,
    AttendanceWeeklyStackedBarSkeleton,
    AttendanceHeatmap,
    AttendanceHeatmapSkeleton,
    AttendanceTermTrendLine,
    AttendanceTermTrendLineSkeleton,
    AttendanceSubjectComparisonBar,
    AttendanceSubjectComparisonBarSkeleton,
    AttendanceVsOverallScatter,
    AttendanceVsOverallScatterSkeleton,
} from "./components/attendance-term";

// Components — Section 2.2 (Class Daily Attendance)
export {
    DailyCalendarHeatmap,
    DailyCalendarHeatmapSkeleton,
    DailyLineChart,
    DailyLineChartSkeleton,
    DayOfWeekBar,
    DayOfWeekBarSkeleton,
    ClassSparklineGrid,
    ClassSparklineGridSkeleton,
    ThresholdAlertBadge,
    ThresholdAlertList,
    ThresholdAlertBadgeSkeleton,
    WeekOverWeekComparison,
    WeekOverWeekComparisonSkeleton,
} from "./components/class-daily-attendance";

// Components — Section 2.3 (Student Term Subject)
export {
    SubjectRadarChartView,
    SubjectRadarChartSkeleton,
    SubjectComparisonBar,
    SubjectComparisonBarSkeleton,
    SubjectTreemap,
    SubjectTreemapSkeleton,
    SubjectDotPlot,
    SubjectDotPlotSkeleton,
    SourceCompositionStackedBar,
    SourceCompositionStackedBarSkeleton,
    SubjectProgressBar,
    SubjectProgressBarSkeleton,
} from "./components/student-term-subject";

// Components — Section 2.4 (Student Term Overall)
export {
    OverallGauge,
    OverallGaugeSkeleton,
    LevelDonutChart,
    LevelDonutChartSkeleton,
    LevelDistributionBar,
    LevelDistributionBarSkeleton,
    WeightedToggle,
    WeightedToggleSkeleton,
    TermOverTermComparison,
    TermOverTermComparisonSkeleton,
    WaterfallContribution,
    WaterfallContributionSkeleton,
    HeadteacherRemarkBadge,
    HeadteacherRemarkBadgeSkeleton,
} from "./components/student-term-overall";

// Components — Section 2.5 (Cohort Position)
export {
    DistributionCurve,
    DistributionCurveSkeleton,
    RankOverTermsLine,
    RankOverTermsLineSkeleton,
    ClassVsGradeBar,
    ClassVsGradeBarSkeleton,
    PercentileGauge,
    PercentileGaugeSkeleton,
    ClassScoreScatter,
    ClassScoreScatterSkeleton,
    StreamComparisonHeatmap,
    StreamComparisonHeatmapSkeleton,
    VarianceBar,
    VarianceBarSkeleton,
    TopBottomList,
    TopBottomListSkeleton,
} from "./components/cohort-position";

// Components — Section 2.6 (Subject Strand)
export {
    StrandHeatmap,
    StrandHeatmapSkeleton,
    StrandMasteryBar,
    StrandMasteryBarSkeleton,
    StrandTreemap,
    StrandTreemapSkeleton,
    StrandGauge,
    StrandGaugeSkeleton,
    RemediationAlertList,
    RemediationAlertListSkeleton,
    SkillRadar,
    SkillRadarSkeleton,
    LevelPieChartView,
    LevelPieChartSkeleton,
    BeforeAfterComparison,
    BeforeAfterComparisonSkeleton,
} from "./components/subject-strand";

// Components — Section 2.7 (Performance Projections)
export {
    ProjectionScatterTrend,
    ProjectionScatterTrendSkeleton,
    ReportCardForecastCard,
    ReportCardForecastCardSkeleton,
    RiskIndicator,
    RiskIndicatorSkeleton,
    GapBarChart,
    GapBarChartSkeleton,
    MomentumArrow,
    MomentumArrowList,
    MomentumArrowSkeleton,
    ProjectionTableSparklines,
    ProjectionTableSparklinesSkeleton,
    ConfidenceBadge,
    ConfidenceBadgeSkeleton,
    ComparisonGrid,
    ComparisonGridSkeleton,
    ActualToProjectedWaterfall,
    ActualToProjectedWaterfallSkeleton,
} from "./components/performance-projections";

// Types for component props
export type {
    WeeklyAttendanceRow,
    HeatmapCell,
    AttendanceHeatmapData,
    TermAttendancePoint,
    SubjectAttendanceEntry,
    AttendanceVsPerformancePoint,
    CalendarDayData,
    DailyAttendancePoint,
    DayOfWeekAverage,
    ClassSparklineData,
    ThresholdAlert,
    WeekOverWeekData,
    SubjectRadarEntry,
    SubjectBarEntry,
    SubjectTreemapEntry,
    SubjectDotEntry,
    SourceCompositionEntry,
    ProgressBarData,
} from "./components";
