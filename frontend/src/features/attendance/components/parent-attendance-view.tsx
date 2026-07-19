/**
 * ParentAttendanceView — shows attendance for the parent's linked children.
 * Pure shadcn: no borders, no cards, no hardcoded colours.
 */

"use client";

import { useState, useMemo } from "react";
import { useRouter } from "next/navigation";
import { UserX, AlertCircle } from "lucide-react";
import { useMe } from "@/hooks/use-auth";
import { getMyParentProfile } from "@/lib/api/parents";
import { getChildAttendanceSummary } from "@/lib/api/attendance";
import { STALE_TIMES } from "@/lib/query-config";
import { useAcademicTerms } from "@/features/academic-terms";
import { useQuery, useQueries } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { ParentAttendanceSummary } from "./parent-attendance-summary";
import { AttendanceEmptyState } from "./attendance-empty-state";

export function ParentAttendanceView() {
    const router = useRouter();
    const { data: me } = useMe();

    const { data: parentProfile, isLoading: profileLoading } = useQuery({
        queryKey: ["parent", "me"],
        queryFn: () => getMyParentProfile(),
        enabled: !!me,
    });

    const { data: termsData, isLoading: termsLoading } = useAcademicTerms();

    const [selectedStudentId, setSelectedStudentId] = useState<string>("");

    const children = parentProfile?.data?.linked_students ?? [];

    const currentTerm = useMemo(
        () => termsData?.items?.find((t) => t.is_current) ?? null,
        [termsData]
    );

    const effectiveStudentId = selectedStudentId || children[0]?.student_id || "";

    const childSummaries = useQueries({
        queries: (children ?? []).map((child) => ({
            queryKey: ["attendance", "child", child.student_id, currentTerm?.id],
            queryFn: () => getChildAttendanceSummary(child.student_id, currentTerm!.id),
            enabled: !!currentTerm && !!child.student_id,
            staleTime: STALE_TIMES.STANDARD,
        })),
    });

    if (!me) {
        router.replace("/logout");
        return null;
    }

    if (profileLoading || termsLoading) {
        return (
            <div className="space-y-4">
                <Skeleton className="h-8 w-48" />
                <Skeleton className="h-24 w-full" />
                <Skeleton className="h-8 w-full" />
                <Skeleton className="h-8 w-full" />
            </div>
        );
    }

    if (children.length === 0) {
        return (
            <AttendanceEmptyState
                icon={UserX}
                title="No children linked to your account"
                description="To view attendance, your school admin needs to link your child's record to your parent account."
            >
                <Button variant="outline" size="sm">
                    Contact school admin
                </Button>
            </AttendanceEmptyState>
        );
    }

    return (
        <div className="space-y-6">
            <p className="text-foreground text-2xl font-bold">Attendance</p>

            {currentTerm && children.length > 0 && (
                <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
                    {children.map((child, idx) => {
                        const summary = childSummaries[idx]?.data;
                        const loading = childSummaries[idx]?.isLoading;
                        const isSelected = child.student_id === effectiveStudentId;
                        return (
                            <button
                                key={child.student_id}
                                type="button"
                                onClick={() => setSelectedStudentId(child.student_id)}
                                className={`bg-muted/30 hover:bg-muted/50 cursor-pointer rounded-md p-3 text-left transition-colors ${
                                    isSelected ? "ring-primary ring-2" : ""
                                }`}
                            >
                                <p className="text-foreground truncate font-medium">
                                    {child.full_name}
                                </p>
                                {loading ? (
                                    <Skeleton className="mt-2 h-4 w-24" />
                                ) : summary ? (
                                    <>
                                        <div className="mt-1 flex items-baseline gap-1">
                                            <span className="text-foreground text-lg font-bold">
                                                {summary.attendance_percentage.toFixed(1)}%
                                            </span>
                                        </div>
                                        <Progress
                                            value={summary.attendance_percentage}
                                            className="mt-2 h-1.5"
                                        />
                                    </>
                                ) : (
                                    <p className="text-muted-foreground mt-1 text-xs">
                                        No data yet
                                    </p>
                                )}
                            </button>
                        );
                    })}
                </div>
            )}

            {children.length > 1 && (
                <div className="w-64">
                    <Select value={effectiveStudentId} onValueChange={setSelectedStudentId}>
                        <SelectTrigger>
                            <SelectValue placeholder="Select child..." />
                        </SelectTrigger>
                        <SelectContent>
                            {children.map((child) => (
                                <SelectItem key={child.student_id} value={child.student_id}>
                                    {child.full_name}
                                </SelectItem>
                            ))}
                        </SelectContent>
                    </Select>
                </div>
            )}

            {effectiveStudentId && currentTerm ? (
                <ParentAttendanceSummary studentId={effectiveStudentId} termId={currentTerm.id} />
            ) : effectiveStudentId ? (
                <AttendanceEmptyState
                    icon={AlertCircle}
                    title="No active term"
                    description="The current academic term has not been set. Attendance data will appear once a term is activated."
                >
                    <Button variant="outline" size="sm">
                        Contact school admin
                    </Button>
                </AttendanceEmptyState>
            ) : null}
        </div>
    );
}
