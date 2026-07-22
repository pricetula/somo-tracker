"use client";

import { useState } from "react";
import { WelcomeGreeting } from "./welcome-greeting";
import { QuickActions, PARENT_ACTIONS } from "..";
import { useMyParentProfile } from "@/features/parents";
import {
    useAttendanceTermSummaries,
    useStudentTermOverallSummaries,
    useStudentTermSubjectSummaries,
    useStudentBehaviorSummaries,
    useStudentCohortPosition,
    useStudentProjections,
    OverallGauge,
    OverallGaugeSkeleton,
    AttendanceGauge,
    AttendanceGaugeSkeleton,
    SubjectRadarChartView,
    SubjectRadarChartSkeleton,
    SubjectComparisonBar,
    SubjectComparisonBarSkeleton,
    BehaviorTrendLine,
    BehaviorTrendLineSkeleton,
    BehaviorPieChart,
    BehaviorPieChartSkeleton,
    PercentileGauge,
    PercentileGaugeSkeleton,
    DistributionCurve,
    DistributionCurveSkeleton,
    ProjectionScatterTrend,
    ProjectionScatterTrendSkeleton,
} from "@/features/analytics";

export function ParentDashboardPage() {
    const { data: parentProfile, isLoading: profileLoading } = useMyParentProfile();
    const linkedStudents = parentProfile?.data?.linked_students ?? [];

    const [selectedStudentId, setSelectedStudentId] = useState<string | null>(null);

    // Default to first linked student if none selected
    const activeStudentId = selectedStudentId ?? linkedStudents[0]?.student_id ?? null;
    const activeStudentName =
        linkedStudents.find((s) => s.student_id === activeStudentId)?.full_name ?? "My Child";

    const filter = activeStudentId ? { student_id: activeStudentId } : {};

    const overall = useStudentTermOverallSummaries({ filter });
    const subjects = useStudentTermSubjectSummaries({ filter });
    const attendance = useAttendanceTermSummaries({ filter });
    const behavior = useStudentBehaviorSummaries({ filter });
    const cohort = useStudentCohortPosition({ filter });
    const projections = useStudentProjections({ filter });

    const overallData = overall.data?.[0];
    const cohortData = cohort.data?.[0];

    const subjectRadar = (subjects.data ?? []).map((s) => ({
        subject: s.learning_area_name,
        score: s.average_percentage,
    }));

    const subjectBars = (subjects.data ?? []).map((s) => ({
        learningAreaName: s.learning_area_name,
        averagePercentage: s.average_percentage,
        level: s.mapped_performance_level,
    }));

    const behaviorTrend = [
        {
            termName: "Current Term",
            commendations: (behavior.data ?? []).reduce((s, b) => s + b.commendations_count, 0),
            disciplinary: (behavior.data ?? []).reduce((s, b) => s + b.disciplinary_count, 0),
        },
    ];
    const totalCommendations = (behavior.data ?? []).reduce((s, b) => s + b.commendations_count, 0);
    const totalDisciplinary = (behavior.data ?? []).reduce((s, b) => s + b.disciplinary_count, 0);

    const projectionData = (projections.data ?? []).map((p) => ({
        historicalScores: p.historical_scores.map((h) => ({
            termIndex: h.term_index,
            termName: h.term_name,
            score: h.score,
        })),
        projectedScore: p.projected_score,
        momentumScore: p.momentum_score,
        lastTermScore: p.last_term_score,
        learningAreaName: p.learning_area_name ?? "Overall",
    }));

    const distributionData = cohortData
        ? {
              studentScore: cohortData.overall_mean_percentage,
              gradeAverage: cohortData.grade_average,
          }
        : null;

    const childrenCount = linkedStudents.length;

    return (
        <div className="flex flex-1 flex-col gap-8 p-6">
            {/* Header */}
            <div>
                <h1 className="text-2xl font-semibold tracking-tight">
                    Welcome back
                    <WelcomeGreeting />
                </h1>
                <p className="text-muted-foreground mt-1">
                    Track your child&apos;s academic progress.
                </p>
            </div>

            {/* ── Functional Sections ── */}

            <QuickActions actions={PARENT_ACTIONS} />

            {!profileLoading && childrenCount > 0 && (
                <section>
                    <h2 className="mb-3 text-lg font-medium">At a Glance</h2>
                    <div className="flex flex-wrap gap-x-10 gap-y-4">
                        <div className="space-y-1">
                            <p className="text-muted-foreground text-sm">Linked Children</p>
                            <p className="text-3xl font-semibold tracking-tight tabular-nums">
                                {childrenCount}
                            </p>
                        </div>
                    </div>
                </section>
            )}

            {/* Student Selector */}
            {profileLoading ? (
                <div className="bg-muted h-10 w-64 animate-pulse rounded-md" />
            ) : linkedStudents.length === 0 ? (
                <div className="bg-muted/20 rounded-lg border px-4 py-6 text-center">
                    <p className="text-muted-foreground text-sm">
                        No linked students yet. Contact the school to link your account.
                    </p>
                </div>
            ) : (
                <div className="flex flex-wrap items-center gap-3">
                    <span className="text-muted-foreground text-sm font-medium">Child:</span>
                    {linkedStudents.map((s) => (
                        <button
                            key={s.student_id}
                            type="button"
                            onClick={() => setSelectedStudentId(s.student_id)}
                            className={`rounded-full px-4 py-1.5 text-sm font-medium transition-colors ${
                                activeStudentId === s.student_id
                                    ? "bg-foreground text-background"
                                    : "bg-muted text-muted-foreground hover:bg-muted/80"
                            }`}
                        >
                            {s.full_name}
                        </button>
                    ))}
                </div>
            )}

            {!activeStudentId && !profileLoading && (
                <p className="text-muted-foreground py-12 text-center text-sm">
                    No student selected.
                </p>
            )}

            {activeStudentId && (
                <>
                    {/* Overall Performance */}
                    <section>
                        <h2 className="mb-4 text-lg font-medium">
                            {activeStudentName}&apos;s Overall Performance
                        </h2>
                        <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
                            {overall.isLoading ? (
                                <>
                                    <OverallGaugeSkeleton />
                                    <PercentileGaugeSkeleton />
                                    <DistributionCurveSkeleton />
                                    <div />
                                </>
                            ) : (
                                <>
                                    {overallData ? (
                                        <OverallGauge
                                            overallMeanPercentage={
                                                overallData.overall_mean_percentage
                                            }
                                            mappedPerformanceLevel={
                                                overallData.mapped_performance_level
                                            }
                                            studentName={overallData.student_name}
                                        />
                                    ) : (
                                        <p className="text-muted-foreground text-sm">
                                            No overall data.
                                        </p>
                                    )}
                                    {cohortData ? (
                                        <PercentileGauge
                                            classPercentile={cohortData.class_percentile}
                                            gradePercentile={cohortData.grade_percentile}
                                            studentName={cohortData.student_name}
                                        />
                                    ) : (
                                        <p className="text-muted-foreground text-sm">
                                            No position data.
                                        </p>
                                    )}
                                    {distributionData ? (
                                        <DistributionCurve data={distributionData} />
                                    ) : (
                                        <p className="text-muted-foreground text-sm">
                                            No distribution data.
                                        </p>
                                    )}
                                </>
                            )}
                        </div>
                    </section>

                    {/* Subject Performance */}
                    <section>
                        <h2 className="mb-4 text-lg font-medium">Subject Performance</h2>
                        <div className="grid gap-6 lg:grid-cols-2">
                            {subjects.isLoading ? (
                                <>
                                    <SubjectRadarChartSkeleton />
                                    <SubjectComparisonBarSkeleton />
                                </>
                            ) : (
                                <>
                                    {subjectRadar.length > 0 ? (
                                        <SubjectRadarChartView data={subjectRadar} />
                                    ) : (
                                        <p className="text-muted-foreground text-sm">
                                            No subject data yet.
                                        </p>
                                    )}
                                    {subjectBars.length > 0 ? (
                                        <SubjectComparisonBar data={subjectBars} />
                                    ) : (
                                        <p className="text-muted-foreground text-sm">
                                            No subject data yet.
                                        </p>
                                    )}
                                </>
                            )}
                        </div>
                    </section>

                    {/* Attendance */}
                    <section>
                        <h2 className="mb-4 text-lg font-medium">Attendance by Subject</h2>
                        <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
                            {attendance.isLoading ? (
                                <>
                                    <AttendanceGaugeSkeleton />
                                    <AttendanceGaugeSkeleton />
                                    <AttendanceGaugeSkeleton />
                                    <AttendanceGaugeSkeleton />
                                </>
                            ) : attendance.data && attendance.data.length > 0 ? (
                                attendance.data.map((a) => (
                                    <AttendanceGauge
                                        key={a.id}
                                        percentage={a.attendance_percentage}
                                        learningAreaName={a.learning_area_name}
                                    />
                                ))
                            ) : (
                                <p className="text-muted-foreground col-span-full text-center text-sm">
                                    No attendance data yet.
                                </p>
                            )}
                        </div>
                    </section>

                    {/* Behaviour */}
                    <section>
                        <h2 className="mb-4 text-lg font-medium">Behaviour</h2>
                        <div className="grid gap-6 lg:grid-cols-2">
                            {behavior.isLoading ? (
                                <>
                                    <BehaviorTrendLineSkeleton />
                                    <BehaviorPieChartSkeleton />
                                </>
                            ) : (
                                <>
                                    <BehaviorTrendLine data={behaviorTrend} />
                                    <BehaviorPieChart
                                        commendationsCount={totalCommendations}
                                        disciplinaryCount={totalDisciplinary}
                                    />
                                </>
                            )}
                        </div>
                    </section>

                    {/* Performance Projections */}
                    {projectionData.length > 0 && (
                        <section>
                            <h2 className="mb-4 text-lg font-medium">Performance Projections</h2>
                            <div className="grid gap-6 lg:grid-cols-2">
                                {projections.isLoading ? (
                                    <>
                                        <ProjectionScatterTrendSkeleton />
                                        <ProjectionScatterTrendSkeleton />
                                    </>
                                ) : (
                                    projectionData
                                        .slice(0, 4)
                                        .map((p) => (
                                            <ProjectionScatterTrend
                                                key={p.learningAreaName}
                                                data={p}
                                            />
                                        ))
                                )}
                            </div>
                        </section>
                    )}
                </>
            )}
        </div>
    );
}
