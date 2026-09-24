"use client";

import Link from "next/link";
import { Plus, TriangleAlert, Users } from "lucide-react";
import { Skeleton } from "@/components/ui/skeleton";
import { Help } from "@/components/help";
import { numberCompactor } from "@/lib/number-compactor";
import { useStudentSummary } from "../hooks/use-student-summary";

export function StudentSummaryCard() {
    const {
        data: {
            male_count = 0,
            female_count = 0,
            total_students = 0,
            unassigned_count = 0,
            unlinked_guardians_count = 0,
        } = {},
        isLoading,
        isError,
    } = useStudentSummary();

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

    const malePercentage = total_students > 0 ? (male_count / total_students) * 100 : 0;
    const femalePercentage = total_students > 0 ? (female_count / total_students) * 100 : 0;
    const totalStudents = total_students ? numberCompactor(total_students) : 0;
    const unassignedCount = unassigned_count ? numberCompactor(unassigned_count) : 0;
    const unlinkedGuardiansCount = unlinked_guardians_count
        ? numberCompactor(unlinked_guardians_count)
        : 0;

    return (
        <article className="flex min-h-30 flex-col">
            <header className="mb-8 space-y-2">
                <h2 className="text-xl font-bold">
                    <Link href="/students">{totalStudents} Students</Link>
                </h2>
                {total_students > 0 && (
                    <div className="flex items-center">
                        <div className="mr-2">Male</div>
                        <div className="flex w-20 items-center">
                            <div
                                className={`h-1 rounded-l-2xl bg-blue-400`}
                                style={{ width: malePercentage }}
                            ></div>
                            <div
                                className={`h-1 rounded-r-2xl bg-blue-600`}
                                style={{ width: femalePercentage }}
                            ></div>
                        </div>
                        <div className="ml-2">Female</div>
                        <Help>
                            <div>
                                <p>Represents the male to female student percentage ratio</p>
                                <span>Male {malePercentage.toFixed(1)}%</span>
                                <span>Female {femalePercentage.toFixed(1)}%</span>
                            </div>
                        </Help>
                    </div>
                )}
            </header>
            <div className="mt-auto space-y-2">
                {!unassigned_count && !unlinked_guardians_count && (
                    <Link href="/students/add">
                        <Plus size="12" className="inline" />
                        <span>Add students</span>
                        <Help>Male to Female student percentage (%) ratio</Help>
                    </Link>
                )}
                {unassigned_count > 0 && (
                    <Link
                        href="/students?without_class=true"
                        className="text-destructive space-x-2"
                    >
                        <TriangleAlert size="12" className="inline" />
                        <span>
                            <b>{unassignedCount}</b> Unassigned
                        </span>
                        <Help>Number of students without a class</Help>
                    </Link>
                )}
                {unlinked_guardians_count > 0 && (
                    <Link
                        href="/students?without_guardian=true"
                        className="space-x-2 text-amber-600"
                    >
                        <Users size="12" className="inline" />
                        <span>
                            <b>{unlinkedGuardiansCount}</b> Unlinked Guardians
                        </span>
                        <Help>Number of students without a guardian/parent</Help>
                    </Link>
                )}
            </div>
        </article>
    );
}
