"use client";

import { FileSpreadsheet, FileCheck, Calendar } from "lucide-react";
import { type ParentAssessmentView } from "@/lib/api/assessments";
import { PerformanceLevelBadge } from "./performance-level-badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

export function AssessmentResultCard({ assessment }: { assessment: ParentAssessmentView }) {
    const isQuant = assessment.evaluation_method === "QUANTITATIVE";

    return (
        <Card>
            <CardHeader className="pb-2">
                <div className="flex items-start justify-between gap-2">
                    <div>
                        <CardTitle className="text-sm font-semibold">
                            {assessment.session_name}
                        </CardTitle>
                        <CardDescription className="mt-0.5 flex items-center gap-1 text-xs">
                            {isQuant ? (
                                <FileSpreadsheet className="h-3 w-3" />
                            ) : (
                                <FileCheck className="h-3 w-3" />
                            )}
                            {isQuant ? "Marks-Based" : "Rubric (Indicator-Level)"}
                            {assessment.scheduled_date && (
                                <>
                                    <span className="text-muted-foreground">\u00b7</span>
                                    <Calendar className="h-3 w-3" />
                                    {assessment.scheduled_date}
                                </>
                            )}
                        </CardDescription>
                    </div>
                </div>
            </CardHeader>
            <CardContent>
                {isQuant ? (
                    <div className="flex items-center gap-4">
                        <div className="flex flex-col">
                            <span className="text-muted-foreground text-xs">Score</span>
                            <span className="text-2xl font-bold tabular-nums">
                                {assessment.raw_score != null ? assessment.raw_score : "-"}
                            </span>
                            {assessment.max_points != null && (
                                <span className="text-muted-foreground text-xs">
                                    out of {assessment.max_points}
                                </span>
                            )}
                        </div>
                        <div className="flex flex-col">
                            <span className="text-muted-foreground text-xs">Level</span>
                            <PerformanceLevelBadge level={assessment.performance_level} showLabel />
                        </div>
                    </div>
                ) : (
                    <div className="space-y-2">
                        <span className="text-muted-foreground text-xs font-medium">
                            Performance Indicators
                        </span>
                        {assessment.outcome_grades && assessment.outcome_grades.length > 0 ? (
                            <div className="flex flex-wrap gap-2">
                                {assessment.outcome_grades.map((grade, idx) => (
                                    <div
                                        key={idx}
                                        className="flex items-center gap-1.5 rounded-md border px-2 py-1"
                                    >
                                        <span className="text-muted-foreground text-xs">
                                            {grade.performance_indicator_id.length > 20
                                                ? `${grade.performance_indicator_id.slice(0, 20)}...`
                                                : grade.performance_indicator_id}
                                        </span>
                                        <PerformanceLevelBadge level={grade.awarded_level} />
                                    </div>
                                ))}
                            </div>
                        ) : (
                            <span className="text-muted-foreground text-xs">
                                No grades available.
                            </span>
                        )}
                    </div>
                )}
            </CardContent>
        </Card>
    );
}
