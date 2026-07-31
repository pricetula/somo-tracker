"use client";

import { useState, useMemo } from "react";
import { FileSpreadsheet, GraduationCap } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { getMyParentProfile } from "@/lib/api/parents";
import { getParentAssessments, getStudentTermGrades } from "@/lib/api/assessments";
import { useAcademicTerms } from "@/features/academic-terms/hooks/use-academic-terms";
import { getErrorMessage } from "@/lib/errors";
import { Badge } from "@/components/ui/badge";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

import { ChildSelector } from "./child-selector";
import { AssessmentResultCard } from "./assessment-result-card";
import { TermGradeCard } from "./term-grade-card";

export function ParentAssessmentsView() {
    // ── Fetch parent's linked children ──────────────────────────────
    const { data: parentProfile, isLoading: profileLoading } = useQuery({
        queryKey: ["parent", "me"],
        queryFn: () => getMyParentProfile(),
    });

    // ── Fetch terms ─────────────────────────────────────────────────
    const { data: termsData, isLoading: termsLoading } = useAcademicTerms();

    const children = parentProfile?.data?.linked_students ?? [];
    const [selectedStudentId, setSelectedStudentId] = useState<string>("");

    // Current term
    const currentTerm = useMemo(
        () => termsData?.items?.find((t) => t.is_current) ?? null,
        [termsData]
    );

    const effectiveStudentId = selectedStudentId || children[0]?.student_id || "";

    // ── Fetch assessments for selected child ────────────────────────
    const {
        data: assessmentsData,
        isLoading: assessmentsLoading,
        isError: assessmentsError,
        error: assessmentsErr,
    } = useQuery({
        queryKey: ["parent-assessments", effectiveStudentId, currentTerm?.id],
        queryFn: () => getParentAssessments(effectiveStudentId, currentTerm!.id),
        enabled: !!effectiveStudentId && !!currentTerm?.id,
    });

    // ── Fetch report card for selected child ────────────────────────
    const {
        data: reportCardData,
        isLoading: reportCardLoading,
        isError: reportCardError,
        error: reportCardErr,
    } = useQuery({
        queryKey: ["report-card", effectiveStudentId, currentTerm?.id],
        queryFn: () => getStudentTermGrades(effectiveStudentId, currentTerm!.id),
        enabled: !!effectiveStudentId && !!currentTerm?.id,
    });

    const assessments = assessmentsData?.items ?? [];
    const termGrades = reportCardData?.items ?? [];

    // ── Loading state ───────────────────────────────────────────────
    if (profileLoading || termsLoading) {
        return (
            <div className="space-y-4">
                <Skeleton className="h-8 w-64" />
                <Skeleton className="h-40 w-full" />
                <Skeleton className="h-40 w-full" />
            </div>
        );
    }

    // ── No children state ───────────────────────────────────────────
    if (children.length === 0) {
        return (
            <Alert>
                <AlertDescription>
                    No children linked to your account. Please contact the school administration.
                </AlertDescription>
            </Alert>
        );
    }

    // ── No active term state ────────────────────────────────────────
    if (!currentTerm) {
        return (
            <Alert>
                <AlertDescription>
                    No active academic term found. Please check back later.
                </AlertDescription>
            </Alert>
        );
    }

    const selectedChild = children.find((c) => c.student_id === effectiveStudentId);

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="flex flex-wrap items-center justify-between gap-4">
                <div>
                    <h1 className="text-2xl font-bold">Assessment Results</h1>
                    <p className="text-muted-foreground mt-0.5 text-sm">
                        View your child&apos;s published assessment results and term report card
                    </p>
                </div>
                <ChildSelector
                    studentList={children}
                    selectedId={effectiveStudentId}
                    onChange={setSelectedStudentId}
                />
            </div>

            {/* Selected child info */}
            {selectedChild && (
                <div className="bg-muted/30 flex items-center gap-2 rounded-md border px-4 py-2">
                    <GraduationCap className="text-muted-foreground h-5 w-5" />
                    <span className="font-medium">{selectedChild.full_name}</span>
                    <Badge variant="secondary" className="text-xs">
                        {currentTerm.name}
                    </Badge>
                </div>
            )}

            {/* Tabs */}
            <Tabs defaultValue="results">
                <TabsList>
                    <TabsTrigger value="results" className="flex items-center gap-1.5">
                        <FileSpreadsheet className="h-4 w-4" />
                        Assessment Results
                    </TabsTrigger>
                    <TabsTrigger value="report-card" className="flex items-center gap-1.5">
                        <GraduationCap className="h-4 w-4" />
                        Term Report Card
                    </TabsTrigger>
                </TabsList>

                {/* Results tab */}
                <TabsContent value="results" className="space-y-4 pt-4">
                    {assessmentsLoading ? (
                        <div className="space-y-4">
                            <Skeleton className="h-32 w-full" />
                            <Skeleton className="h-32 w-full" />
                        </div>
                    ) : assessmentsError ? (
                        <Alert variant="destructive">
                            <AlertDescription>{getErrorMessage(assessmentsErr)}</AlertDescription>
                        </Alert>
                    ) : assessments.length === 0 ? (
                        <Alert>
                            <AlertDescription>
                                No published assessment results found for {selectedChild?.full_name}{" "}
                                in this term.
                            </AlertDescription>
                        </Alert>
                    ) : (
                        <div className="grid gap-3 sm:grid-cols-2">
                            {assessments.map((assessment) => (
                                <AssessmentResultCard
                                    key={assessment.session_id}
                                    assessment={assessment}
                                />
                            ))}
                        </div>
                    )}
                </TabsContent>

                {/* Report Card tab */}
                <TabsContent value="report-card" className="space-y-4 pt-4">
                    {reportCardLoading ? (
                        <div className="space-y-4">
                            <Skeleton className="h-32 w-full" />
                            <Skeleton className="h-32 w-full" />
                        </div>
                    ) : reportCardError ? (
                        <Alert variant="destructive">
                            <AlertDescription>{getErrorMessage(reportCardErr)}</AlertDescription>
                        </Alert>
                    ) : termGrades.length === 0 ? (
                        <Alert>
                            <AlertDescription>
                                No term grades available for {selectedChild?.full_name}. Grades are
                                compiled after assessments are published.
                            </AlertDescription>
                        </Alert>
                    ) : (
                        <div className="grid gap-3 sm:grid-cols-2">
                            {termGrades.map((grade) => (
                                <TermGradeCard key={grade.learning_area_id} grade={grade} />
                            ))}
                        </div>
                    )}
                </TabsContent>
            </Tabs>
        </div>
    );
}
