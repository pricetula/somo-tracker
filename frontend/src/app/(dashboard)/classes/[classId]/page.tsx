/**
 * Class Detail Page — CBC tabbed view
 *
 * Surfaces:
 *   1. Timetable builder (admin, school_admin — weekly grid)
 *   2. Attendance recording (teacher, school_admin — per-period)
 *
 * Tab-based layout: Overview | Timetable | Attendance
 */

"use client";

import * as React from "react";
import { useParams, useSearchParams, useRouter } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { ArrowLeft } from "lucide-react";

import { useMe } from "@/hooks/use-auth";

import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";

import { CbcTimetablePage } from "@/features/cbc/components/timetable/cbc-timetable-page";
import { CbcAttendancePage } from "@/features/cbc/components/attendance/cbc-attendance-page";

// ─── Types for the class detail fetch ──────────────────────────────────────

interface ClassDetailData {
    id: string;
    name: string;
    stream: string;
    grade_id: string;
    grade_name: string;
    school_id: string;
    academic_year_id: string;
    academic_term_id: string;
    education_system_id: string;
    teacher_count: number;
    student_count: number;
    has_class_teachers: boolean;
}

// ─── Component ────────────────────────────────────────────────────────────

export default function ClassDetailPage() {
    const params = useParams<{ classId: string }>();
    const searchParams = useSearchParams();
    const router = useRouter();
    const { data: me } = useMe();
    const classId = params.classId;

    const tabFromUrl = searchParams.get("tab") ?? "timetable";
    const validTabs = ["timetable", "attendance", "overview"] as const;
    const activeTab = validTabs.includes(tabFromUrl as (typeof validTabs)[number])
        ? tabFromUrl
        : "timetable";

    // ── Fetch class detail ─────────────────────────────────────────────
    const {
        data: classDetail,
        isLoading,
        error,
    } = useQuery<ClassDetailData>({
        queryKey: ["class", "detail", classId],
        queryFn: async () => {
            const res = await fetch(`/api/v1/schools/classes/${classId}`);
            if (!res.ok) throw new Error("Failed to load class");
            return res.json();
        },
        enabled: !!classId,
    });

    // ── Handle tab change ──────────────────────────────────────────────
    const handleTabChange = (value: string) => {
        router.push(`/classes/${classId}?tab=${value}`);
    };

    // ── Loading state ──────────────────────────────────────────────────
    if (isLoading) {
        return (
            <div className="flex flex-1 flex-col gap-4 p-6">
                <Skeleton className="h-6 w-48" />
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-96 w-full" />
            </div>
        );
    }

    // ── Error state ────────────────────────────────────────────────────
    if (error || !classDetail) {
        return (
            <div className="flex flex-1 items-center justify-center p-6">
                <div className="text-center">
                    <h2 className="text-lg font-medium">Class not found</h2>
                    <p className="text-muted-foreground mt-1 text-sm">
                        This class could not be loaded. It may have been removed or you may not have
                        access.
                    </p>
                    <Button
                        variant="outline"
                        size="sm"
                        className="mt-4"
                        onClick={() => router.push("/classes")}
                    >
                        Back to classes
                    </Button>
                </div>
            </div>
        );
    }

    return (
        <div className="flex flex-1 flex-col gap-4 p-6">
            {/* ── Back navigation ────────────────────────────────────── */}
            <div className="flex items-center gap-2">
                <Button
                    variant="ghost"
                    size="sm"
                    className="h-7 px-2 text-xs"
                    onClick={() => router.push("/classes")}
                >
                    <ArrowLeft className="mr-1 size-3.5" />
                    Classes
                </Button>
            </div>

            {/* ── Class header ────────────────────────────────────────── */}
            <div>
                <h1 className="text-2xl font-semibold tracking-tight">
                    {classDetail.name}
                    {classDetail.stream && (
                        <span className="text-muted-foreground ml-2 text-lg font-normal">
                            {classDetail.stream}
                        </span>
                    )}
                </h1>
                <p className="text-muted-foreground mt-0.5 text-sm">
                    {classDetail.grade_name}
                    {classDetail.teacher_count > 0 &&
                        ` · ${classDetail.teacher_count} teacher${classDetail.teacher_count !== 1 ? "s" : ""}`}
                    {classDetail.student_count > 0 &&
                        ` · ${classDetail.student_count} student${classDetail.student_count !== 1 ? "s" : ""}`}
                </p>
            </div>

            {/* ── Tabs ────────────────────────────────────────────────── */}
            <Tabs value={activeTab} onValueChange={handleTabChange}>
                <TabsList className="bg-muted w-fit">
                    <TabsTrigger value="overview" className="text-xs">
                        Overview
                    </TabsTrigger>
                    <TabsTrigger value="timetable" className="text-xs">
                        Timetable
                    </TabsTrigger>
                    <TabsTrigger value="attendance" className="text-xs">
                        Attendance
                    </TabsTrigger>
                </TabsList>

                {/* Tab: Overview */}
                <TabsContent value="overview" className="mt-4">
                    <div className="rounded-lg border p-6">
                        <h3 className="text-base font-medium">Class overview</h3>
                        <dl className="mt-3 grid grid-cols-2 gap-4 text-sm">
                            <div>
                                <dt className="text-muted-foreground text-xs">Grade</dt>
                                <dd className="font-medium">{classDetail.grade_name}</dd>
                            </div>
                            <div>
                                <dt className="text-muted-foreground text-xs">Stream</dt>
                                <dd className="font-medium">{classDetail.stream || "—"}</dd>
                            </div>
                            <div>
                                <dt className="text-muted-foreground text-xs">Teachers</dt>
                                <dd className="font-medium">{classDetail.teacher_count}</dd>
                            </div>
                            <div>
                                <dt className="text-muted-foreground text-xs">Students</dt>
                                <dd className="font-medium">{classDetail.student_count}</dd>
                            </div>
                        </dl>
                    </div>
                </TabsContent>

                {/* Tab: Timetable */}
                <TabsContent value="timetable" className="mt-4">
                    <CbcTimetablePage
                        classId={classId}
                        schoolId={classDetail.school_id}
                        academicYearId={classDetail.academic_year_id}
                        gradeId={classDetail.grade_id}
                        className={classDetail.name}
                        hasClassTeachers={classDetail.has_class_teachers}
                    />
                </TabsContent>

                {/* Tab: Attendance */}
                <TabsContent value="attendance" className="mt-4">
                    <CbcAttendancePage
                        classId={classId}
                        schoolId={classDetail.school_id}
                        academicTermId={classDetail.academic_term_id}
                        gradeId={classDetail.grade_id}
                        className={classDetail.name}
                        userId={me?.user_id}
                        userRole={
                            me?.role as
                                | "SYSTEM_ADMIN"
                                | "SCHOOL_ADMIN"
                                | "TEACHER"
                                | "SUPPORT_STAFF"
                                | undefined
                        }
                    />
                </TabsContent>
            </Tabs>
        </div>
    );
}
