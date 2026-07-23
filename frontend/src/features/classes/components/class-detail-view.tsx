/**
 * ClassDetailView — The main content for viewing a class's roster.
 *
 * Used by both the full-page render and the intercepted side sheet.
 * Shows class info header, roster table, and "Enroll Students" button.
 */

"use client";

import { useQuery } from "@tanstack/react-query";
import { GraduationCap } from "lucide-react";

import { getClass } from "@/lib/api/classes";
import { ClassRoster } from "./class-roster";
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
            <div className="rounded-md border">
                <Skeleton className="h-10 w-full rounded-none border-b" />
                <Skeleton className="h-10 w-full rounded-none" />
                <Skeleton className="h-10 w-full rounded-none" />
                <Skeleton className="h-10 w-3/4 rounded-none" />
            </div>
        </div>
    );
}

// ─── Component ─────────────────────────────────────────────────────────────

export function ClassDetailView({ classId }: ClassDetailViewProps) {
    const { data: classData, isLoading, isError } = useClassDetail(classId);

    if (isLoading) return <ClassDetailSkeleton />;

    if (isError || !classData) {
        return (
            <div className="space-y-4">
                <p className="text-destructive py-8 text-center">
                    {isError ? "Failed to load class details." : "Class not found."}
                </p>
            </div>
        );
    }

    return (
        <article className="space-y-6">
            {/* Header */}
            <header className="flex items-start justify-between gap-4">
                <div className="flex items-center gap-2">
                    <GraduationCap className="text-muted-foreground h-5 w-5" />
                    <h1 className="text-lg font-semibold">{classData.display_label}</h1>
                </div>
                <p className="text-muted-foreground">
                    {classData.student_count ?? 0} student
                    {(classData.student_count ?? 0) !== 1 ? "s" : ""} enrolled
                </p>
            </header>

            {/* Roster */}
            <div>
                <h2 className="text-muted-foreground mb-3 font-medium">Roster</h2>
                <ClassRoster classId={classId} />
            </div>
        </article>
    );
}
