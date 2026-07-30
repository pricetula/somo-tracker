"use client";

import { WelcomeGreeting } from "./welcome-greeting";
import { QuickActions, SYSTEM_ADMIN_ACTIONS } from "..";
import { useAcademicTerms } from "@/features/academic-terms/hooks/use-academic-terms";
import { useClassList } from "@/features/classes/hooks/use-classes";
import {
    useAttendanceTermSummaries,
    useClassDailyAttendance,
    useTeacherDeliverySummaries,
    useTeacherPerformanceSummaries,
    AttendanceGauge,
    AttendanceGaugeSkeleton,
    DailyLineChart,
    DailyLineChartSkeleton,
    DeliveryGauge,
    DeliveryGaugeSkeleton,
    TeacherKpiCards,
    TeacherKpiCardsSkeleton,
    TeacherComparisonBar,
    TeacherComparisonBarSkeleton,
} from "@/features/analytics";

export function SystemAdminDashboardPage() {
    const attendance = useAttendanceTermSummaries();
    const { data: termsData } = useAcademicTerms();
    const { data: classesData } = useClassList();
    const currentTerm = (termsData?.items ?? []).find((t) => t.is_current);
    const termStart = currentTerm?.start_date;
    const termEnd = currentTerm?.end_date;
    const firstClassId = classesData?.items?.[0]?.value ?? "";
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

    const attendanceData = attendance.data ?? [];
    const teacherPerfData = teacherPerf.data ?? [];
    const deliveryData = delivery.data ?? [];
    const classDailyData = (classDaily.data ?? []).filter(
        (d) => d.daily_attendance_rate > 0 || d.total_enrolled > 0
    );

    const avgAttendance =
        attendanceData.reduce((s, r) => s + r.attendance_percentage, 0) /
        (attendanceData.length || 1);

    const dailyData = classDailyData.map((d) => ({
        date: d.date,
        rate: d.daily_attendance_rate,
    }));

    const perfKpis =
        teacherPerfData.length > 0
            ? [
                  {
                      label: "Cohort Mastery Rate",
                      value:
                          teacherPerfData.reduce((s, r) => s + r.cohort_mastery_rate, 0) /
                          teacherPerfData.length,
                      suffix: "%" as const,
                      trend: "stable" as const,
                  },
                  {
                      label: "Growth Rate",
                      value:
                          teacherPerfData.reduce((s, r) => s + r.student_growth_rate, 0) /
                          teacherPerfData.length,
                      suffix: "pts" as const,
                      trend: "up" as const,
                  },
                  {
                      label: "Timeliness",
                      value:
                          teacherPerfData.reduce((s, r) => s + r.assessment_timeliness_index, 0) /
                          teacherPerfData.length,
                      suffix: "%" as const,
                      trend: "up" as const,
                  },
                  {
                      label: "Strand Coverage",
                      value:
                          teacherPerfData.reduce((s, r) => s + r.strand_coverage_rate, 0) /
                          teacherPerfData.length,
                      suffix: "%" as const,
                      trend: "stable" as const,
                  },
                  {
                      label: "Subject Mean",
                      value:
                          teacherPerfData.reduce((s, r) => s + r.subject_mean_score, 0) /
                          teacherPerfData.length,
                      suffix: "%" as const,
                      trend: "up" as const,
                  },
              ]
            : [];

    const deliveryRate =
        deliveryData.reduce((s, r) => s + r.on_time_submission_rate, 0) /
        (deliveryData.length || 1);

    const teacherComparisons = teacherPerfData.map((t) => ({
        teacherName: t.teacher_name ?? "Unknown",
        subjectMeanScore: t.subject_mean_score,
        subjectName: t.learning_area_name,
    }));

    return (
        <div className="flex flex-1 flex-col gap-8 p-6">
            {/* Header */}
            <div>
                <h1 className="text-2xl font-semibold tracking-tight">
                    Welcome back
                    <WelcomeGreeting />
                </h1>
                <p className="text-muted-foreground mt-1">System-wide analytics overview.</p>
            </div>

            {/* ── Functional Sections ── */}

            <QuickActions actions={SYSTEM_ADMIN_ACTIONS} />

            {/* Attendance Overview */}
            <section>
                <h2 className="mb-4 text-lg font-medium">Attendance Overview</h2>
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
                                learningAreaName="Overall"
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

            {/* Daily Attendance Trend */}
            <section>
                {classDaily.isLoading ? (
                    <DailyLineChartSkeleton />
                ) : (
                    <DailyLineChart data={dailyData} />
                )}
            </section>

            {/* Teacher Performance */}
            <section>
                <h2 className="mb-4 text-lg font-medium">Teacher Performance</h2>
                <div className="grid gap-6 lg:grid-cols-2">
                    <div>
                        {teacherPerf.isLoading ? (
                            <TeacherKpiCardsSkeleton />
                        ) : (
                            <TeacherKpiCards data={perfKpis} />
                        )}
                    </div>
                    <div>
                        {delivery.isLoading ? (
                            <DeliveryGaugeSkeleton />
                        ) : (
                            <DeliveryGauge rate={deliveryRate} target={95} />
                        )}
                    </div>
                </div>
            </section>

            {/* Teacher Comparison */}
            <section>
                <h2 className="mb-4 text-lg font-medium">Teacher Subject Comparison</h2>
                {teacherPerf.isLoading ? (
                    <TeacherComparisonBarSkeleton />
                ) : (
                    <TeacherComparisonBar data={teacherComparisons} />
                )}
            </section>
        </div>
    );
}
