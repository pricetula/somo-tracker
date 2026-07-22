/**
 * Student Performance Projections — Section 2.7 components barrel.
 */

export { ProjectionScatterTrend, ProjectionScatterTrendSkeleton } from "./projection-scatter-trend";
export type { TrendProjectionData } from "./projection-scatter-trend";

export {
    ReportCardForecastCard,
    ReportCardForecastCardSkeleton,
} from "./report-card-forecast-card";
export type { ForecastCardData } from "./report-card-forecast-card";

export { RiskIndicator, RiskIndicatorSkeleton } from "./risk-indicator";
export type { RiskData } from "./risk-indicator";

export { GapBarChart, GapBarChartSkeleton } from "./gap-bar-chart";

export { MomentumArrow, MomentumArrowList, MomentumArrowSkeleton } from "./momentum-arrow";
export type { MomentumData } from "./momentum-arrow";

export {
    ProjectionTableSparklines,
    ProjectionTableSparklinesSkeleton,
} from "./projection-table-sparklines";
export type { ProjectionTableRow } from "./projection-table-sparklines";

export { ConfidenceBadge, ConfidenceBadgeSkeleton } from "./confidence-badge";
export type { ConfidenceData } from "./confidence-badge";

export { ComparisonGrid, ComparisonGridSkeleton } from "./comparison-grid";
export type { ComparisonGridItem } from "./comparison-grid";

export {
    ActualToProjectedWaterfall,
    ActualToProjectedWaterfallSkeleton,
} from "./actual-to-projected-waterfall";
export type { ActualToProjectedData } from "./actual-to-projected-waterfall";
