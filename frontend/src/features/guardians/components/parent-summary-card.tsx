"use client";

import Link from "next/link";
import { Plus, TriangleAlert, Users } from "lucide-react";
import { Skeleton } from "@/components/ui/skeleton";
import { Help } from "@/components/help";
import { numberCompactor } from "@/lib/number-compactor";
import { useGuardianSummary } from "../hooks/use-guardian-summary";

export function ParentSummaryCard() {
    const {
        data: { total_guardians = 0, guardians_without_student = 0 } = {},
        isLoading,
        isError,
    } = useGuardianSummary();

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

    const totalGuardians = total_guardians ? numberCompactor(total_guardians) : 0;
    const guardiansWithoutStudent = guardians_without_student
        ? numberCompactor(guardians_without_student)
        : 0;

    return (
        <article className="flex min-h-30 flex-col">
            <header className="mb-8 space-y-2">
                <h2 className="text-xl font-bold">
                    <Link href="/guardians">{totalGuardians} Guardians</Link>
                </h2>
            </header>
            <div className="mt-auto space-y-2">
                {!guardians_without_student && (
                    <Link href="/guardians/invite">
                        <Plus size="12" className="inline" />
                        <span>Invite guardians</span>
                        <Help>Invite new guardians to the school</Help>
                    </Link>
                )}
                {guardians_without_student > 0 && (
                    <Link href="/guardians" className="text-destructive space-x-2">
                        <TriangleAlert size="12" className="inline" />
                        <span>
                            <b>{guardiansWithoutStudent}</b> Guardians without students
                        </span>
                        <Help>Number of guardians not linked to any student</Help>
                    </Link>
                )}
                {total_guardians > 0 && guardians_without_student === 0 && (
                    <Link href="/guardians" className="text-muted-foreground space-x-2">
                        <Users size="12" className="inline" />
                        <span>All guardians linked</span>
                        <Help>All guardians are linked to at least one student</Help>
                    </Link>
                )}
            </div>
        </article>
    );
}
