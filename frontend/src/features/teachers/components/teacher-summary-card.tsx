"use client";

import Link from "next/link";
import { Plus, TriangleAlert, Users } from "lucide-react";
import { Skeleton } from "@/components/ui/skeleton";
import { Help } from "@/components/help";
import { numberCompactor } from "@/lib/number-compactor";
import { useTeacherSummary } from "../hooks/use-teacher-summary";

export function TeacherSummaryCard() {
    const {
        data: { total_teachers = 0, teachers_without_assignment = 0 } = {},
        isLoading,
        isError,
    } = useTeacherSummary();

    if (isLoading)
        return (
            <div className="h-30 w-40 space-y-2">
                <div className="mb-6 space-y-2">
                    <Skeleton className="h-6 w-40" />
                    <Skeleton className="h-4 w-40" />
                </div>
                <Skeleton className="h-4 w-40" />
            </div>
        );

    if (isError) return <div>Error loading summary.</div>;

    const totalTeachers = total_teachers ? numberCompactor(total_teachers) : 0;
    const teachersWithoutAssignment = teachers_without_assignment
        ? numberCompactor(teachers_without_assignment)
        : 0;

    return (
        <article className="flex min-h-30 flex-col">
            <header className="mb-8 space-y-2">
                <h2 className="text-xl font-bold">
                    <Link href="/teachers">{totalTeachers} Teachers</Link>
                </h2>
            </header>
            <div className="mt-auto space-y-2">
                {!teachers_without_assignment && (
                    <Link href="/teachers/invite">
                        <Plus size="12" className="inline" />
                        <span>Invite teachers</span>
                        <Help>Invite new teachers to the school</Help>
                    </Link>
                )}
                {teachers_without_assignment > 0 && (
                    <Link href="/teachers" className="text-destructive space-x-2">
                        <TriangleAlert size="12" className="inline" />
                        <span>
                            <b>{teachersWithoutAssignment}</b> Teachers without assignment
                        </span>
                        <Help>Number of teachers not assigned to a timetable slot</Help>
                    </Link>
                )}
                {total_teachers > 0 && teachers_without_assignment === 0 && (
                    <Link href="/teachers" className="text-muted-foreground space-x-2">
                        <Users size="12" className="inline" />
                        <span>All teachers assigned</span>
                        <Help>All teachers are assigned to at least one timetable slot</Help>
                    </Link>
                )}
            </div>
        </article>
    );
}
