"use client";

import { AttendanceCalendar } from "@/features/attendance";
import { WelcomeGreeting } from "./welcome-greeting";
import {
    QuickActions,
    QuickStats,
    PendingItems,
    SetupChecklist,
    ActivityFeed,
    SCHOOL_ADMIN_ACTIONS,
} from "..";
import { useDashboardCounts, useDashboardPendingItems } from "..";
import {
    useClassDailyAttendance,
    useTeacherDeliverySummaries,
    useTeacherPerformanceSummaries,
    useTeacherWorkloadSummaries,
    useStudentBehaviorSummaries,
    DailyLineChart,
    DailyLineChartSkeleton,
    DayOfWeekBar,
    DayOfWeekBarSkeleton,
    DeliveryGauge,
    DeliveryGaugeSkeleton,
    DeliveryComparisonBar,
    DeliveryComparisonBarSkeleton,
    TeacherKpiCards,
    TeacherKpiCardsSkeleton,
    TeacherComparisonBar,
    TeacherComparisonBarSkeleton,
    WorkloadComparisonBar,
    WorkloadComparisonBarSkeleton,
    BehaviorTrendLine,
    BehaviorTrendLineSkeleton,
    BehaviorPieChart,
    BehaviorPieChartSkeleton,
} from "@/features/analytics";

export function SchoolAdminDashboardPage() {
    const classDaily = useClassDailyAttendance();
    const delivery = useTeacherDeliverySummaries();
    const teacherPerf = useTeacherPerformanceSummaries();
    const workload = useTeacherWorkloadSummaries();
    const behavior = useStudentBehaviorSummaries();
    const counts = useDashboardCounts();
    const pending = useDashboardPendingItems();

    const classDailyData = classDaily.data ?? [];

    const dailyData = classDailyData.map((d) => ({
        date: d.date,
        rate: d.daily_attendance_rate,
    }));
    const dayShort = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];
    const dayOfWeekData = dayShort.slice(1, 6).map((day) => {
        const dayIndex = dayShort.indexOf(day);
        const entries = classDailyData.filter((d) => new Date(d.date).getDay() === dayIndex);
        const avg =
            entries.length > 0
                ? entries.reduce((s, e) => s + e.daily_attendance_rate, 0) / entries.length
                : 0;
        return { dayName: day, averageRate: avg };
    });

    const teacherPerfData = teacherPerf.data ?? [];
    const deliveryData = delivery.data ?? [];
    const behaviorData = behavior.data ?? [];

    const deliveryRates = deliveryData
        .filter((d) => d.teacher_name)
        .map((d) => ({
            teacherName: d.teacher_name!,
            submissionRate: d.on_time_submission_rate,
        }));

    const avgDeliveryRate =
        deliveryData.reduce((s, d) => s + d.on_time_submission_rate, 0) /
        (deliveryData.length || 1);

    const perfKpis =
        teacherPerfData.length > 0
            ? [
                  {
                      label: "Avg Subject Mean",
                      value:
                          teacherPerfData.reduce((s, r) => s + r.subject_mean_score, 0) /
                          teacherPerfData.length,
                      suffix: "%" as const,
                      trend: "up" as const,
                  },
                  {
                      label: "Avg Mastery Rate",
                      value:
                          teacherPerfData.reduce((s, r) => s + r.cohort_mastery_rate, 0) /
                          teacherPerfData.length,
                      suffix: "%" as const,
                      trend: "stable" as const,
                  },
                  {
                      label: "Avg Growth",
                      value:
                          teacherPerfData.reduce((s, r) => s + r.student_growth_rate, 0) /
                          teacherPerfData.length,
                      suffix: "pts" as const,
                      trend: "up" as const,
                  },
                  {
                      label: "Avg Timeliness",
                      value:
                          teacherPerfData.reduce((s, r) => s + r.assessment_timeliness_index, 0) /
                          teacherPerfData.length,
                      suffix: "%" as const,
                      trend: "up" as const,
                  },
              ]
            : [];

    const teacherComparisons = teacherPerfData.map((t) => ({
        teacherName: t.teacher_name ?? "Unknown",
        subjectMeanScore: t.subject_mean_score,
        subjectName: t.learning_area_name,
    }));

    const workloadData = workload.data ?? [];
    const workloadEntries = workloadData.map((w) => ({
        teacherName: w.teacher_name ?? "Unknown",
        utilization: w.utilization_percentage,
        isOvercapacity: w.is_overcapacity,
        periods: w.total_assigned_periods,
    }));

    const behaviorTrend = [
        {
            termName: "Current Term",
            commendations: behaviorData.reduce((s, b) => s + b.commendations_count, 0),
            disciplinary: behaviorData.reduce((s, b) => s + b.disciplinary_count, 0),
        },
    ];

    const totalCommendations = behaviorData.reduce((s, b) => s + b.commendations_count, 0);
    const totalDisciplinary = behaviorData.reduce((s, b) => s + b.disciplinary_count, 0);

    return (
        <div className="flex flex-1 flex-col gap-8 p-6">
            {/* Header */}
            <div>
                <h1 className="text-2xl font-semibold tracking-tight">
                    Dashboard
                    <WelcomeGreeting />
                </h1>
                <p className="text-muted-foreground mt-1">School-wide overview and quick access.</p>
            </div>

            {/* ── Functional Sections ── */}

            <QuickActions actions={SCHOOL_ADMIN_ACTIONS} />

            <QuickStats
                stats={
                    counts.data
                        ? [
                              { label: "Students", value: counts.data.students },
                              { label: "Teachers", value: counts.data.teachers },
                              { label: "Classes", value: counts.data.classes },
                              {
                                  label: "Pending Invitations",
                                  value: counts.data.pendingInvitations,
                              },
                          ]
                        : []
                }
                isLoading={counts.isLoading}
            />

            <PendingItems
                items={
                    pending.data?.map((item) => ({
                        type: item.type,
                        label: item.label,
                        description: item.description,
                        href: item.href,
                    })) ?? []
                }
                isLoading={pending.isLoading}
            />

            <SetupChecklist />

            {/* ── Analytics Sections ── */}

            {/* Daily Attendance Trend */}
            <section>
                <h2 className="mb-4 text-lg font-medium">Daily Attendance</h2>
                <div className="grid gap-6 lg:grid-cols-2">
                    {classDaily.isLoading ? (
                        <>
                            <DailyLineChartSkeleton />
                            <DayOfWeekBarSkeleton />
                        </>
                    ) : (
                        <>
                            <DailyLineChart data={dailyData} />
                            <DayOfWeekBar data={dayOfWeekData} />
                        </>
                    )}
                </div>
                <div className="mt-4">
                    {classDaily.isLoading ? (
                        <div className="bg-muted h-72 animate-pulse rounded" />
                    ) : (
                        <AttendanceCalendar
                            attendanceRateMap={Object.fromEntries(
                                classDailyData.map((d) => [d.date, d.daily_attendance_rate])
                            )}
                        />
                    )}
                </div>
            </section>

            {/* Teacher Delivery */}
            <section>
                <h2 className="mb-4 text-lg font-medium">Teacher Delivery</h2>
                <div className="grid gap-6 lg:grid-cols-2">
                    {delivery.isLoading ? (
                        <>
                            <DeliveryGaugeSkeleton />
                            <DeliveryComparisonBarSkeleton />
                        </>
                    ) : (
                        <>
                            <DeliveryGauge rate={avgDeliveryRate} target={95} />
                            <DeliveryComparisonBar data={deliveryRates} />
                        </>
                    )}
                </div>
            </section>

            {/* Teacher Performance */}
            <section>
                <h2 className="mb-4 text-lg font-medium">Teacher Performance</h2>
                <div className="grid gap-6 lg:grid-cols-2">
                    {teacherPerf.isLoading ? (
                        <>
                            <TeacherKpiCardsSkeleton />
                            <TeacherComparisonBarSkeleton />
                        </>
                    ) : (
                        <>
                            <TeacherKpiCards data={perfKpis} />
                            <TeacherComparisonBar data={teacherComparisons} />
                        </>
                    )}
                </div>
            </section>

            {/* Teacher Workload */}
            <section>
                <h2 className="mb-4 text-lg font-medium">Teacher Workload</h2>
                {workload.isLoading ? (
                    <WorkloadComparisonBarSkeleton />
                ) : (
                    <WorkloadComparisonBar data={workloadEntries} />
                )}
            </section>

            {/* Student Behavior Overview */}
            <section>
                <h2 className="mb-4 text-lg font-medium">Student Behaviour Overview</h2>
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

            {/* Recent Activity (at the bottom) */}
            <ActivityFeed />
        </div>
    );
}
