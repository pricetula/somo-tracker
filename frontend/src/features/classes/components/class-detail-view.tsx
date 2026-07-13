/**
 * ClassDetailView — The main content for viewing a class's roster.
 *
 * Used by both the full-page render and the intercepted side sheet.
 * Shows class info header, roster table, and "Enroll Students" button.
 */

"use client";

import { useQuery } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { UserPlus, ArrowLeft, GraduationCap } from "lucide-react";

import { getClass } from "@/lib/api/classes";
import { ClassRoster, RosterSkeleton } from "./class-roster";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";

// ─── Props ─────────────────────────────────────────────────────────────────

interface ClassDetailViewProps {
    classId: string;
    /** When true, renders a back button for standalone page mode. */
    showBackButton?: boolean;
}

// ─── Hook ──────────────────────────────────────────────────────────────────

function useClassDetail(classId: string) {
    return useQuery({
        queryKey: ["class", classId],
        queryFn: () => getClass(classId),
        staleTime: 30_000,
    });
}

// ─── Skeleton ──────────────────────────────────────────────────────────────

export function ClassDetailSkeleton() {
    return (
        <div className="space-y-6">
            <div className="flex items-center gap-3">
                <Skeleton className="h-8 w-8 rounded-full" />
                <div className="space-y-1.5">
                    <Skeleton className="h-6 w-48" />
                    <Skeleton className="h-4 w-32" />
                </div>
            </div>
            <RosterSkeleton />
        </div>
    );
}

// ─── Component ─────────────────────────────────────────────────────────────

export function ClassDetailView({ classId, showBackButton }: ClassDetailViewProps) {
    const router = useRouter();
    const { data: classData, isLoading, isError } = useClassDetail(classId);

    if (isLoading) return <ClassDetailSkeleton />;

    if (isError || !classData) {
        return (
            <div className="space-y-4">
                {showBackButton && (
                    <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => router.back()}
                        className="-ml-2"
                    >
                        <ArrowLeft className="mr-2 h-4 w-4" />
                        Back
                    </Button>
                )}
                <p className="text-destructive py-8 text-center">
                    {isError ? "Failed to load class details." : "Class not found."}
                </p>
            </div>
        );
    }

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="flex items-start justify-between gap-4">
                <div>
                    {showBackButton && (
                        <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => router.back()}
                            className="mb-2 -ml-2"
                        >
                            <ArrowLeft className="mr-2 h-4 w-4" />
                            Back
                        </Button>
                    )}
                    <div className="flex items-center gap-2">
                        <GraduationCap className="text-muted-foreground h-5 w-5" />
                        <h1 className="text-lg font-semibold">{classData.display_label}</h1>
                    </div>
                    <p className="text-muted-foreground text-sm">
                        {classData.student_count ?? 0} student
                        {(classData.student_count ?? 0) !== 1 ? "s" : ""} enrolled
                    </p>
                </div>
                <Button size="sm" onClick={() => router.push(`/classes/${classId}/enroll`)}>
                    <UserPlus className="mr-2 h-4 w-4" />
                    Enroll Students
                </Button>
            </div>

            {/* Roster */}
            <div>
                <h2 className="text-muted-foreground mb-3 text-sm font-medium">Roster</h2>
                <ClassRoster classId={classId} />
            </div>
        </div>
    );
}
