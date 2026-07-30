"use client";

import { AttendanceCalendar } from "@/features/attendance";
import { WelcomeGreeting } from "./welcome-greeting";
import { QuickActions, TEACHER_ACTIONS } from "..";
import { useMe } from "@/hooks/use-auth";
import { useAcademicTerms } from "@/features/academic-terms/hooks/use-academic-terms";
import { useClassTeachersByTeacher } from "@/features/classteachers/hooks/use-classteachers";
import {
    useClassDailyAttendance,
    useTeacherDeliverySummaries,
    useTeacherPerformanceSummaries,
    useTeacherWorkloadSummaries,
    useStudentBehaviorSummaries,
    useStudentTermSubjectSummaries,
    DailyLineChart,
    DailyLineChartSkeleton,
    DayOfWeekBar,
    DayOfWeekBarSkeleton,
    DeliveryGauge,
    DeliveryGaugeSkeleton,
    TeacherPerformanceRadar,
    TeacherPerformanceRadarSkeleton,
    TeacherKpiCards,
    TeacherKpiCardsSkeleton,
    WorkloadUtilizationGauge,
    WorkloadUtilizationGaugeSkeleton,
    BehaviorTrendLine,
    BehaviorTrendLineSkeleton,
    CategoryBreakdownBar,
    CategoryBreakdownBarSkeleton,
    SubjectRadarChartView,
    SubjectRadarChartSkeleton,
    SubjectComparisonBar,
    SubjectComparisonBarSkeleton,
} from "@/features/analytics";

export function TeacherDashboardPage() {
    const { data: me } = useMe();
    const { data: termsData } = useAcademicTerms();
    const teacherUserId = me?.user_id ?? "";
    const { data: teacherClasses } = useClassTeachersByTeacher(teacherUserId);

    // Derive the first assigned class_id and current term's date range
    const firstClassId = teacherClasses?.items?.[0]?.class_id ?? "";
    const currentTerm = (termsData?.items ?? []).find((t) => t.is_current);
    const termStart = currentTerm?.start_date;
    const termEnd = currentTerm?.end_date;

    const classDaily = useClassDailyAttendance({
        filter: { class_id: firstClassId || undefined },
        startDate: termStart,
        endDate: termEnd,
    });
    const delivery = useTeacherDeliverySummaries({
        filter: { academic_term_id: currentTerm?.id },
    });
    const teacherPerf = useTeacherPerformanceSummaries({
        filter: { academic_term_id: currentTerm?.id },
    });
    const workload = useTeacherWorkloadSummaries({
        filter: { academic_year_id: currentTerm?.academic_year_id },
    });
    const behavior = useStudentBehaviorSummaries();
    const subjects = useStudentTermSubjectSummaries();

    const classDailyData = (classDaily.data ?? []).filter(
        (d) => d.daily_attendance_rate > 0 || d.total_enrolled > 0
    );
    const deliveryData = delivery.data ?? [];
    const teacherPerfData = teacherPerf.data ?? [];
    const workloadData = workload.data ?? [];
    const behaviorData = behavior.data ?? [];
    const subjectsData = subjects.data ?? [];

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

    const myDelivery = deliveryData.find((d) => d.user_id === teacherUserId);
    const myPerf = teacherPerfData.find((p) => p.user_id === teacherUserId);

    const radarData = myPerf
        ? [
              { metric: "Subject Mean", value: myPerf.subject_mean_score },
              { metric: "Mastery Rate", value: myPerf.cohort_mastery_rate },
              { metric: "Growth Rate", value: Math.min(myPerf.student_growth_rate * 20, 100) },
              { metric: "Timeliness", value: myPerf.assessment_timeliness_index },
              { metric: "Coverage", value: myPerf.strand_coverage_rate },
          ]
        : [];

    const perfKpis: {
        label: string;
        value: number;
        suffix: string;
        trend: "up" | "down" | "stable";
    }[] = myPerf
        ? [
              {
                  label: "Subject Mean",
                  value: myPerf.subject_mean_score,
                  suffix: "%",
                  trend: myPerf.subject_mean_score >= 70 ? "up" : "down",
              },
              {
                  label: "Mastery Rate",
                  value: myPerf.cohort_mastery_rate,
                  suffix: "%",
                  trend: myPerf.cohort_mastery_rate >= 60 ? "up" : "down",
              },
              {
                  label: "Growth",
                  value: myPerf.student_growth_rate,
                  suffix: "pts",
                  trend: myPerf.student_growth_rate >= 2 ? "up" : "down",
              },
              {
                  label: "Timeliness",
                  value: myPerf.assessment_timeliness_index,
                  suffix: "%",
                  trend: myPerf.assessment_timeliness_index >= 85 ? "up" : "down",
              },
          ]
        : [];

    const myWorkload = workloadData.find((w) => w.user_id === teacherUserId);
    const myUtilization = myWorkload?.utilization_percentage ?? 0;
    const overcapacity = myWorkload?.is_overcapacity ?? false;

    const behaviorTrend = [
        {
            termName: "Current Term",
            commendations: behaviorData.reduce((s, b) => s + b.commendations_count, 0),
            disciplinary: behaviorData.reduce((s, b) => s + b.disciplinary_count, 0),
        },
    ];

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

    const subjectRadar = subjectsData.map((s) => ({
        subject: s.learning_area_name,
        score: s.average_percentage,
    }));

    const subjectBars = subjectsData.map((s) => ({
        learningAreaName: s.learning_area_name,
        averagePercentage: s.average_percentage,
        level: s.mapped_performance_level,
    }));

    return (
        <div className="flex flex-1 flex-col gap-8 p-6">
            {/* Header */}
            <div>
                <h1 className="text-2xl font-semibold tracking-tight">
                    Welcome back
                    <WelcomeGreeting />
                </h1>
                <p className="text-muted-foreground mt-1">Your teaching dashboard.</p>
            </div>

            {/* ── Functional Sections ── */}

            <QuickActions actions={TEACHER_ACTIONS} />

            {/* ── Analytics Sections ── */}

            {/* Class Daily Attendance */}
            <section>
                <h2 className="mb-4 text-lg font-medium">Class Attendance</h2>
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

                {classDaily.isLoading ? (
                    <div className="bg-muted mt-4 h-72 w-full max-w-md animate-pulse rounded" />
                ) : (
                    <div className="mt-4">
                        <AttendanceCalendar
                            attendanceRateMap={Object.fromEntries(
                                classDailyData.map((d) => [d.date, d.daily_attendance_rate])
                            )}
                        />
                    </div>
                )}
            </section>

            {/* My Performance */}
            <section>
                <h2 className="mb-4 text-lg font-medium">My Performance</h2>
                <div className="grid gap-6 lg:grid-cols-2">
                    {teacherPerf.isLoading ? (
                        <>
                            <TeacherPerformanceRadarSkeleton />
                            <TeacherKpiCardsSkeleton />
                        </>
                    ) : (
                        <>
                            {radarData.length > 0 && <TeacherPerformanceRadar data={radarData} />}
                            {perfKpis.length > 0 && <TeacherKpiCards data={perfKpis} />}
                            {radarData.length === 0 && perfKpis.length === 0 && (
                                <p className="text-muted-foreground text-sm">
                                    No performance data yet.
                                </p>
                            )}
                        </>
                    )}
                </div>
            </section>

            {/* Subject Performance Overview */}
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
                            {subjectRadar.length > 0 && (
                                <SubjectRadarChartView data={subjectRadar} />
                            )}
                            {subjectBars.length > 0 && <SubjectComparisonBar data={subjectBars} />}
                            {subjectRadar.length === 0 && (
                                <p className="text-muted-foreground text-sm">
                                    No subject data yet.
                                </p>
                            )}
                        </>
                    )}
                </div>
            </section>

            {/* Delivery & Workload */}
            <section>
                <h2 className="mb-4 text-lg font-medium">Delivery & Workload</h2>
                <div className="grid gap-6 sm:grid-cols-2">
                    {delivery.isLoading ? (
                        <DeliveryGaugeSkeleton />
                    ) : (
                        <DeliveryGauge
                            rate={myDelivery?.on_time_submission_rate ?? 0}
                            target={95}
                            teacherName={myDelivery?.teacher_name}
                        />
                    )}
                    {workload.isLoading ? (
                        <WorkloadUtilizationGaugeSkeleton />
                    ) : (
                        <WorkloadUtilizationGauge
                            utilization={myUtilization}
                            isOvercapacity={overcapacity}
                            teacherName={myWorkload?.teacher_name}
                        />
                    )}
                </div>
            </section>

            {/* Student Behaviour */}
            <section>
                <h2 className="mb-4 text-lg font-medium">Student Behaviour</h2>
                <div className="grid gap-6 lg:grid-cols-2">
                    {behavior.isLoading ? (
                        <>
                            <BehaviorTrendLineSkeleton />
                            <CategoryBreakdownBarSkeleton />
                        </>
                    ) : (
                        <>
                            <BehaviorTrendLine data={behaviorTrend} />
                            <CategoryBreakdownBar data={categoryData} />
                        </>
                    )}
                </div>
            </section>
        </div>
    );
}
