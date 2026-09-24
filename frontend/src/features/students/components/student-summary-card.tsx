"use client";

import Link from "next/link";
import { Plus, TriangleAlert, Users } from "lucide-react";
import { Skeleton } from "@/components/ui/skeleton";
import { numberCompactor } from "@/lib/number-compactor";
import { useStudentSummary } from "../hooks/use-student-summary";

export function StudentSummaryCard() {
    const { data: summary, isLoading, isError } = useStudentSummary();

    if (isLoading)
        return (
            <div className="h-30 w-40 space-y-2">
                <div className="mb-6 space-y-2">
                    <Skeleton className="h-6 w-40" />
                    <Skeleton className="h-4 w-40" />
                </div>
                <Skeleton className="h-4 w-40" />
                <Skeleton className="h-4 w-40" />
            </div>
        );

    if (isError) return <div>Error loading summary.</div>;

    if (!summary) return null;

    const malePercentage = (summary.male_count / summary.total_students) * 100;
    const femalePercentage = (summary.female_count / summary.total_students) * 100;
    const totalStudents = summary.total_students ? numberCompactor(summary.total_students) : 0;
    const unassignedCount = summary.unassigned_count
        ? numberCompactor(summary.unassigned_count)
        : 0;
    const unlinkedGuardiansCount = summary.unlinked_guardians_count
        ? numberCompactor(summary.unlinked_guardians_count)
        : 0;

    return (
        <article className="h-30 w-40 space-y-2">
            <header className="mb-6 space-y-2">
                <h2 className="text-xl font-bold">
                    <Link href="/students">{totalStudents} Students</Link>
                </h2>
                <div className="flex w-40 items-center">
                    <div className="mr-2">Male</div>
                    <div
                        className={`h-1 rounded-l-2xl bg-blue-400`}
                        style={{ width: malePercentage }}
                    ></div>
                    <div
                        className={`h-1 rounded-r-2xl bg-blue-600`}
                        style={{ width: femalePercentage }}
                    ></div>
                    <div className="ml-2">Female</div>
                </div>
            </header>
            {!summary.unassigned_count && !summary.unlinked_guardians_count && (
                <Link href="/students/add" className="block">
                    <Plus size="12" className="inline" />
                    <span>Add students</span>
                </Link>
            )}
            {summary.unassigned_count && (
                <Link
                    href="/students?without_class=true"
                    className="text-destructive block space-x-2"
                >
                    <TriangleAlert size="12" className="inline" />
                    <span>
                        <b>{unassignedCount}</b> Unassigned
                    </span>
                </Link>
            )}
            {summary.unlinked_guardians_count && (
                <Link
                    href="/students?without_guardian=true"
                    className="block space-x-2 text-amber-600"
                >
                    <Users size="12" className="inline" />
                    <span>
                        <b>{unlinkedGuardiansCount}</b> Unlinked Guardians
                    </span>
                </Link>
            )}
        </article>
    );
}
