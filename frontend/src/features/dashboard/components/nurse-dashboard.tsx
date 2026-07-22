"use client";

import { WelcomeGreeting } from "./welcome-greeting";
import { QuickActions, QuickStats, NURSE_ACTIONS } from "..";
import {
    useStudentBehaviorSummaries,
    useAttendanceTermSummaries,
    BehaviorTrendLine,
    BehaviorTrendLineSkeleton,
    BehaviorPieChart,
    BehaviorPieChartSkeleton,
    CategoryBreakdownBar,
    CategoryBreakdownBarSkeleton,
    BehaviorAlertBadge,
    BehaviorAlertBadgeSkeleton,
    NetSentimentScore,
    NetSentimentScoreSkeleton,
    AttendanceGauge,
    AttendanceGaugeSkeleton,
} from "@/features/analytics";

export function NurseDashboardPage() {
    const behavior = useStudentBehaviorSummaries();
    const attendance = useAttendanceTermSummaries();

    const behaviorData = behavior.data ?? [];
    const attendanceData = attendance.data ?? [];

    const behaviorTrend = [
        {
            termName: "Current Term",
            commendations: behaviorData.reduce((s, b) => s + b.commendations_count, 0),
            disciplinary: behaviorData.reduce((s, b) => s + b.disciplinary_count, 0),
        },
    ];

    const totalCommendations = behaviorData.reduce((s, b) => s + b.commendations_count, 0);
    const totalDisciplinary = behaviorData.reduce((s, b) => s + b.disciplinary_count, 0);
    const totalUrgent = behaviorData.reduce((s, b) => s + b.urgent_count, 0);
    const totalIncidents = behaviorData.reduce((s, b) => s + b.total_incidents, 0);

    // Derive category breakdown from behavior data
    const categoryData = behaviorData.flatMap((b) =>
        b.primary_category_name
            ? [
                  {
                      categoryName: b.primary_category_name,
                      categoryType: b.primary_category_type ?? "DISCIPLINARY",
                      count: b.disciplinary_count + b.urgent_count,
                  },
              ]
            : []
    );

    const avgAttendance =
        attendanceData.reduce((s, r) => s + r.attendance_percentage, 0) /
        (attendanceData.length || 1);

    return (
        <div className="flex flex-1 flex-col gap-8 p-6">
            {/* Header */}
            <div>
                <h1 className="text-2xl font-semibold tracking-tight">
                    Welcome back
                    <WelcomeGreeting />
                </h1>
                <p className="text-muted-foreground mt-1">Student health and wellbeing overview.</p>
            </div>

            {/* ── Functional Sections ── */}

            <QuickActions actions={NURSE_ACTIONS} />

            <QuickStats
                stats={[
                    { label: "Total Incidents", value: totalIncidents },
                    { label: "Urgent", value: totalUrgent },
                    { label: "Disciplinary", value: totalDisciplinary },
                    { label: "Commendations", value: totalCommendations },
                ]}
                isLoading={behavior.isLoading}
            />

            {/* Alert Banner */}
            {totalUrgent > 0 && (
                <div className="bg-destructive/10 border-destructive/20 flex items-center gap-3 rounded-lg border px-4 py-3">
                    <span className="text-destructive text-sm font-medium">
                        {totalUrgent} urgent incident{totalUrgent !== 1 ? "s" : ""} require
                        {totalUrgent === 1 ? "s" : ""} attention
                    </span>
                    {behavior.isLoading ? (
                        <BehaviorAlertBadgeSkeleton />
                    ) : (
                        <BehaviorAlertBadge
                            totalIncidents={totalIncidents}
                            urgentCount={totalUrgent}
                            disciplinaryCount={totalDisciplinary}
                        />
                    )}
                </div>
            )}

            {/* Behaviour Overview */}
            <section>
                <h2 className="mb-4 text-lg font-medium">Behaviour Overview</h2>
                <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
                    {behavior.isLoading ? (
                        <>
                            <BehaviorTrendLineSkeleton />
                            <BehaviorPieChartSkeleton />
                            <NetSentimentScoreSkeleton />
                        </>
                    ) : (
                        <>
                            <BehaviorTrendLine data={behaviorTrend} />
                            <BehaviorPieChart
                                commendationsCount={totalCommendations}
                                disciplinaryCount={totalDisciplinary}
                            />
                            <NetSentimentScore
                                commendationsCount={totalCommendations}
                                disciplinaryCount={totalDisciplinary}
                            />
                        </>
                    )}
                </div>
            </section>

            {/* Behaviour Categories */}
            <section>
                <h2 className="mb-4 text-lg font-medium">Incident Categories</h2>
                <div className="grid gap-6 lg:grid-cols-2">
                    {behavior.isLoading ? (
                        <CategoryBreakdownBarSkeleton />
                    ) : (
                        <CategoryBreakdownBar data={categoryData} />
                    )}
                    {behavior.isLoading ? (
                        <BehaviorPieChartSkeleton />
                    ) : (
                        <BehaviorPieChart
                            commendationsCount={totalCommendations}
                            disciplinaryCount={totalDisciplinary}
                        />
                    )}
                </div>
            </section>

            {/* Wellbeing Context */}
            <section>
                <h2 className="mb-4 text-lg font-medium">Wellbeing Context</h2>
                <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
                    {attendance.isLoading ? (
                        <>
                            <AttendanceGaugeSkeleton />
                            <AttendanceGaugeSkeleton />
                            <AttendanceGaugeSkeleton />
                            <AttendanceGaugeSkeleton />
                        </>
                    ) : (
                        <>
                            <AttendanceGauge
                                percentage={avgAttendance}
                                learningAreaName="Overall Attendance"
                            />
                            {attendanceData.slice(0, 3).map((a) => (
                                <AttendanceGauge
                                    key={a.id}
                                    percentage={a.attendance_percentage}
                                    learningAreaName={a.learning_area_name}
                                />
                            ))}
                        </>
                    )}
                </div>
            </section>
        </div>
    );
}
