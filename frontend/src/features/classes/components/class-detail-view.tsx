/**
 * ClassDetailView — The main content for viewing a class's roster.
 *
 * Used by both the full-page render and the intercepted side sheet.
 * Shows class info header, academic year/term selector comboboxes,
 * roster table, and "Enroll Students" button.
 */

"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { GraduationCap, Trash2 } from "lucide-react";

import { getClass } from "@/lib/api/classes";
import { STALE_TIMES } from "@/lib/query-config";
import {
    useAcademicYears,
    useAcademicTerms,
    AcademicYearCombobox,
    AcademicTermCombobox,
} from "@/features/academic-terms";
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

    // Selected academic year and term (controlled comboboxes)
    const [selectedYearId, setSelectedYearId] = useState("");
    const [selectedTermId, setSelectedTermId] = useState("");

    // Derive effective values — user selection, or auto-select current year/term as default
    const yearId = selectedYearId || (yearsData?.items?.find((y) => y.is_current)?.id ?? "");
    const termId = selectedTermId || (termsData?.items?.find((t) => t.is_current)?.id ?? "");

    // Reset term when year changes
    const handleYearChange = (nextYearId: string) => {
        setSelectedYearId(nextYearId);
        setSelectedTermId("");
    };

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
                    <AlertDialogTrigger asChild>
                        <Button variant="outline" size="sm" className="text-destructive">
                            <Trash2 className="mr-1.5 size-3.5" />
                            Delete
                        </Button>
                    </AlertDialogTrigger>
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
                                onClick={async () => {
                                    try {
                                        await deleteMutation.mutateAsync([classId]);
                                        router.push("/classes");
                                    } catch {
                                        // handled by hook onError
                                    }
                                }}
                                disabled={deleteMutation.isPending}
                            >
                                {deleteMutation.isPending ? "Deleting…" : "Delete"}
                            </AlertDialogAction>
                        </AlertDialogFooter>
                    </AlertDialogContent>
                </AlertDialog>
            </header>

            {/* Academic year / term selector */}
            <div className="flex flex-wrap items-end gap-3">
                <div className="flex flex-col gap-1.5">
                    <label className="text-muted-foreground text-xs font-medium">
                        Academic Year
                    </label>
                    <AcademicYearCombobox
                        value={yearId}
                        onChange={handleYearChange}
                        placeholder="Select year..."
                        className="w-56"
                    />
                </div>
                <div className="flex flex-col gap-1.5">
                    <label className="text-muted-foreground text-xs font-medium">
                        Academic Term
                    </label>
                    <AcademicTermCombobox
                        value={termId}
                        onChange={setSelectedTermId}
                        placeholder="Select term..."
                        className="w-56"
                    />
                </div>
            </div>

            {/* Roster for the selected term */}
            <div>
                <h2 className="text-muted-foreground mb-3 font-medium">
                    Roster
                    {termId && termsData?.items && (
                        <span className="ml-2 font-normal">
                            &mdash;{" "}
                            {termsData.items.find((t) => t.id === termId)?.name ?? "selected term"}
                        </span>
                    )}
                </h2>
                <ClassRoster classId={classId} academicYearId={yearId} academicTermId={termId} />
            </div>
        </article>
    );
}
