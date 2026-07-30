"use client";

import { WelcomeGreeting } from "./welcome-greeting";
import { QuickActions, FINANCE_ACTIONS } from "..";
import { useAcademicTerms } from "@/features/academic-terms/hooks/use-academic-terms";
import { useClassList } from "@/features/classes/hooks/use-classes";
import {
    useTeacherWorkloadSummaries,
    useAttendanceTermSummaries,
    useClassDailyAttendance,
    WorkloadComparisonBar,
    WorkloadComparisonBarSkeleton,
    WorkloadUtilizationGauge,
    WorkloadUtilizationGaugeSkeleton,
    AttendanceGauge,
    AttendanceGaugeSkeleton,
    DailyLineChart,
    DailyLineChartSkeleton,
    AttendanceSubjectComparisonBar,
    AttendanceSubjectComparisonBarSkeleton,
} from "@/features/analytics";

export function FinanceDashboardPage() {
    const { data: termsData } = useAcademicTerms();
    const { data: classesData } = useClassList();
    const currentTerm = (termsData?.items ?? []).find((t) => t.is_current);
    const workload = useTeacherWorkloadSummaries({
        filter: { academic_year_id: currentTerm?.academic_year_id },
    });
    const attendance = useAttendanceTermSummaries();
    const termStart = currentTerm?.start_date;
    const termEnd = currentTerm?.end_date;
    const firstClassId = classesData?.items?.[0]?.value ?? "";
    const classDaily = useClassDailyAttendance({
        filter: { class_id: firstClassId || undefined },
        startDate: termStart,
        endDate: termEnd,
    });

    const workloadData = workload.data ?? [];
    const attendanceData = attendance.data ?? [];
    const classDailyData = (classDaily.data ?? []).filter(
        (d) => d.daily_attendance_rate > 0 || d.total_enrolled > 0
    );

    const workloadEntries = workloadData.map((w) => ({
        teacherName: w.teacher_name ?? "Unknown",
        utilization: w.utilization_percentage,
        isOvercapacity: w.is_overcapacity,
        periods: w.total_assigned_periods,
    }));

    const overcapacityTeachers = workloadEntries.filter((w) => w.isOvercapacity);
    const avgUtilization =
        workloadData.reduce((s, w) => s + w.utilization_percentage, 0) / (workloadData.length || 1);

    const dailyData = classDailyData.map((d) => ({
        date: d.date,
        rate: d.daily_attendance_rate,
    }));

    const subjectAttendance = attendanceData.map((a) => ({
        learningAreaName: a.learning_area_name ?? "Unknown",
        percentage: a.attendance_percentage,
    }));

    const overallAttendance =
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
                <p className="text-muted-foreground mt-1">Finance dashboard.</p>
            </div>

            {/* ── Functional Sections ── */}

            <QuickActions actions={FINANCE_ACTIONS} />

            {/* Overcapacity Alert */}
            {overcapacityTeachers.length > 0 && (
                <div className="bg-destructive/10 border-destructive/20 rounded-lg border px-4 py-3">
                    <p className="text-destructive text-sm font-medium">
                        {overcapacityTeachers.length} teacher
                        {overcapacityTeachers.length !== 1 ? "s" : ""} overcapacity —{" "}
                        {overcapacityTeachers.map((t) => t.teacherName).join(", ")}
                    </p>
                </div>
            )}

            {/* Teacher Workload */}
            <section>
                <h2 className="mb-4 text-lg font-medium">Teacher Workload Analysis</h2>
                <div className="grid gap-6 sm:grid-cols-2">
                    {workload.isLoading ? (
                        <>
                            <WorkloadComparisonBarSkeleton />
                            <WorkloadUtilizationGaugeSkeleton />
                        </>
                    ) : (
                        <>
                            <WorkloadComparisonBar data={workloadEntries} />
                            <WorkloadUtilizationGauge
                                utilization={avgUtilization}
                                isOvercapacity={overcapacityTeachers.length > 0}
                            />
                        </>
                    )}
                </div>
            </section>

            {/* Attendance Context */}
            <section>
                <h2 className="mb-4 text-lg font-medium">Attendance Overview</h2>
                <div className="grid gap-6 lg:grid-cols-2">
                    {classDaily.isLoading ? (
                        <DailyLineChartSkeleton />
                    ) : (
                        <DailyLineChart data={dailyData} />
                    )}
                    {attendance.isLoading ? (
                        <AttendanceSubjectComparisonBarSkeleton />
                    ) : (
                        <AttendanceSubjectComparisonBar data={subjectAttendance} />
                    )}
                </div>
                <div className="mt-4 grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
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
                                percentage={overallAttendance}
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
        </div>
    );
}
