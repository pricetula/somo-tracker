/**
 * ClassDetailView — The main content for viewing a class's roster.
 *
 * Used by both the full-page render and the intercepted side sheet.
 * Shows class info header, academic year/term selector comboboxes,
 * roster table, and "Enroll Students" button.
 */

"use client";

import { useQuery } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { GraduationCap, Trash2 } from "lucide-react";

import { getClass } from "@/lib/api/classes";
import { STALE_TIMES } from "@/lib/query-config";
import { useAcademicYears, useAcademicTerms } from "@/features/academic-terms";
import { ClassRoster } from "./class-roster";
import { ClassDetailSkeleton } from "./class-detail-skeleton";
import { Button } from "@/components/ui/button";
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
    AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { useDeleteClasses } from "../hooks/use-classes";

// ─── Props ─────────────────────────────────────────────────────────────────

interface ClassDetailViewProps {
    classId: string;
}

// ─── Hook ──────────────────────────────────────────────────────────────────

function useClassDetail(classId: string) {
    return useQuery({
        queryKey: ["class", classId],
        queryFn: () => getClass(classId),
        staleTime: STALE_TIMES.FREQUENT,
    });
}

// ─── Skeleton ──────────────────────────────────────────────────────────────

// ─── Component ─────────────────────────────────────────────────────────────

export function ClassDetailView({ classId }: ClassDetailViewProps) {
    const router = useRouter();
    const { data: classData, isLoading, isError } = useClassDetail(classId);
    const { data: yearsData } = useAcademicYears();
    const deleteMutation = useDeleteClasses();
    const { data: termsData } = useAcademicTerms();

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
                <AlertDialog>
                    <AlertDialogTrigger
                        render={
                            <Button variant="outline" size="sm" className="text-destructive">
                                <Trash2 className="mr-1.5 size-3.5" />
                                Delete
                            </Button>
                        }
                    />
                    <AlertDialogContent>
                        <AlertDialogHeader>
                            <AlertDialogTitle>Delete Class</AlertDialogTitle>
                            <AlertDialogDescription>
                                Are you sure you want to delete &ldquo;
                                {classData.display_label}&rdquo;? This action cannot be undone.
                            </AlertDialogDescription>
                        </AlertDialogHeader>
                        <AlertDialogFooter>
                            <AlertDialogCancel>Cancel</AlertDialogCancel>
                            <AlertDialogAction
                                variant="destructive"
                                onClick={() => {
                                    deleteMutation.mutate([classId], {
                                        onSuccess: () => router.push("/classes"),
                                    });
                                }}
                                disabled={deleteMutation.isPending}
                            >
                                {deleteMutation.isPending ? "Deleting…" : "Delete"}
                            </AlertDialogAction>
                        </AlertDialogFooter>
                    </AlertDialogContent>
                </AlertDialog>
            </header>

            {/* Current academic year / term (resolved server-side) */}
            <div className="flex flex-wrap items-end gap-3">
                <div className="flex flex-col gap-1.5">
                    <label className="text-muted-foreground font-medium">Academic Year</label>
                    <span className="text-foreground w-56 truncate py-1.5 text-sm">
                        {(Array.isArray(yearsData?.data)
                            ? yearsData.data.find((y) => y.is_current)?.name
                            : null) ?? "—"}
                    </span>
                </div>
                <div className="flex flex-col gap-1.5">
                    <label className="text-muted-foreground font-medium">Academic Term</label>
                    <span className="text-foreground w-56 truncate py-1.5 text-sm">
                        {(Array.isArray(termsData?.items)
                            ? termsData.items.find((t) => t.is_current)?.name
                            : null) ?? "—"}
                    </span>
                </div>
            </div>

            {/* Roster for the current active term (resolved server-side) */}
            <div>
                <h2 className="text-muted-foreground mb-3 font-medium">Roster</h2>
                <ClassRoster classId={classId} />
            </div>
        </article>
    );
}
