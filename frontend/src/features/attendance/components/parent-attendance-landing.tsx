/**
 * ParentAttendanceLanding — parent's view of linked children's attendance.
 * Pure shadcn: no borders, no cards, flat layout.
 *
 * Fetches the parent profile to discover linked students, then renders
 * an attendance summary for each child.
 */

"use client";

import { useMemo } from "react";
import { Users } from "lucide-react";

import { Skeleton } from "@/components/ui/skeleton";

import { AttendanceEmptyState } from "./attendance-empty-state";
import { ParentAttendanceSummary } from "./parent-attendance-summary";
import { useAcademicTerms } from "@/features/academic-terms/hooks/use-academic-terms";
import { useMyParentProfile } from "@/features/parents/hooks/use-parents";

export function ParentAttendanceLanding() {
    const {
        data: profileData,
        isLoading: profileLoading,
        isError: profileError,
    } = useMyParentProfile();
    const { data: termsData, isLoading: termsLoading } = useAcademicTerms();

    const currentTerm = useMemo(() => termsData?.items?.find((t) => t.is_current), [termsData]);

    const isLoading = profileLoading || termsLoading;

    if (isLoading) {
        return (
            <div className="space-y-6">
                <Skeleton className="h-8 w-64" />
                <Skeleton className="h-32 w-full" />
                <Skeleton className="h-32 w-full" />
            </div>
        );
    }

    if (profileError || !profileData?.data) {
        return (
            <div className="text-destructive bg-destructive/10 p-4">
                Failed to load your profile. Please try again.
            </div>
        );
    }

    const linkedStudents = profileData.data.linked_students ?? [];

    if (linkedStudents.length === 0) {
        return (
            <AttendanceEmptyState
                icon={Users}
                title="No linked children"
                description="Your account is not linked to any students yet. Contact the school to link your children."
            />
        );
    }

    if (!currentTerm) {
        return (
            <AttendanceEmptyState
                icon={Users}
                title="No active term"
                description="There is no current academic term. Attendance summaries will be available once a term is active."
            />
        );
    }

    return (
        <div className="space-y-8">
            <h1 className="text-foreground text-2xl font-bold">Attendance Overview</h1>
            {linkedStudents.map((student) => (
                <section key={student.student_id}>
                    <div className="space-y-4">
                        <h2 className="text-foreground text-lg font-semibold">
                            {student.full_name}
                        </h2>
                        {student.relationship && (
                            <p className="text-muted-foreground -mt-3 text-xs">
                                {student.relationship}
                            </p>
                        )}
                        <ParentAttendanceSummary
                            studentId={student.student_id}
                            termId={currentTerm.id}
                        />
                    </div>
                </section>
            ))}
        </div>
    );
}
