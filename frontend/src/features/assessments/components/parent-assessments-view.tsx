/**
 * ParentAssessmentsView — Shows published assessment results and term report
 * card for a parent's linked children.
 *
 * Multi-child selector. Shows the current term's data by default.
 * Displays both QUANTITATIVE (raw score + level) and RUBRIC (per-indicator grades).
 */

"use client";

import { useState, useMemo } from "react";
import { FileSpreadsheet, FileCheck, GraduationCap, User, Calendar, BookOpen } from "lucide-react";

import { useQuery } from "@tanstack/react-query";
import { getMyParentProfile } from "@/lib/api/parents";
import { getParentAssessments, getStudentTermGrades } from "@/lib/api/assessments";
import type { ParentAssessmentView, StudentTermGrade } from "@/lib/api/assessments";
import { useAcademicTerms } from "@/features/academic-terms/hooks/use-academic-terms";
import { PerformanceLevelBadge } from "./performance-level-badge";
import { getErrorMessage } from "@/lib/errors";
import { Badge } from "@/components/ui/badge";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

// ── Child Selector ──────────────────────────────────────────────────────

function ChildSelector({
    studentList,
    selectedId,
    onChange,
}: {
    studentList: { student_id: string; full_name: string }[];
    selectedId: string;
    onChange: (id: string) => void;
}) {
    if (studentList.length <= 1) return null;

    return (
        <div className="flex items-center gap-2">
            <User className="text-muted-foreground h-4 w-4" />
            <Select value={selectedId} onValueChange={onChange}>
                <SelectTrigger className="w-64">
                    <SelectValue placeholder="Select a child..." />
                </SelectTrigger>
                <SelectContent>
                    {studentList.map((child) => (
                        <SelectItem key={child.student_id} value={child.student_id}>
                            {child.full_name}
                        </SelectItem>
                    ))}
                </SelectContent>
            </Select>
        </div>
    );
}

// ── Assessment Result Card ──────────────────────────────────────────────

function AssessmentResultCard({ assessment }: { assessment: ParentAssessmentView }) {
    const isQuant = assessment.evaluation_method === "QUANTITATIVE";

    return (
        <Card>
            <CardHeader className="pb-2">
                <div className="flex items-start justify-between gap-2">
                    <div>
                        <CardTitle className="text-sm font-semibold">
                            {assessment.session_name}
                        </CardTitle>
                        <CardDescription className="mt-0.5 flex items-center gap-1 text-xs">
                            {isQuant ? (
                                <FileSpreadsheet className="h-3 w-3" />
                            ) : (
                                <FileCheck className="h-3 w-3" />
                            )}
                            {isQuant ? "Marks-Based" : "Rubric (Indicator-Level)"}
                            {assessment.scheduled_date && (
                                <>
                                    <span className="text-muted-foreground">\u00b7</span>
                                    <Calendar className="h-3 w-3" />
                                    {assessment.scheduled_date}
                                </>
                            )}
                        </CardDescription>
                    </div>
                </div>
            </CardHeader>
            <CardContent>
                {isQuant ? (
                    <div className="flex items-center gap-4">
                        <div className="flex flex-col">
                            <span className="text-muted-foreground text-xs">Score</span>
                            <span className="text-2xl font-bold tabular-nums">
                                {assessment.raw_score != null ? assessment.raw_score : "-"}
                            </span>
                            {assessment.max_points != null && (
                                <span className="text-muted-foreground text-xs">
                                    out of {assessment.max_points}
                                </span>
                            )}
                        </div>
                        <div className="flex flex-col">
                            <span className="text-muted-foreground text-xs">Level</span>
                            <PerformanceLevelBadge level={assessment.performance_level} showLabel />
                        </div>
                    </div>
                ) : (
                    <div className="space-y-2">
                        <span className="text-muted-foreground text-xs font-medium">
                            Performance Indicators
                        </span>
                        {assessment.outcome_grades && assessment.outcome_grades.length > 0 ? (
                            <div className="flex flex-wrap gap-2">
                                {assessment.outcome_grades.map((grade, idx) => (
                                    <div
                                        key={idx}
                                        className="flex items-center gap-1.5 rounded-md border px-2 py-1"
                                    >
                                        <span className="text-muted-foreground text-xs">
                                            {grade.performance_indicator_id.length > 20
                                                ? `${grade.performance_indicator_id.slice(0, 20)}...`
                                                : grade.performance_indicator_id}
                                        </span>
                                        <PerformanceLevelBadge level={grade.awarded_level} />
                                    </div>
                                ))}
                            </div>
                        ) : (
                            <span className="text-muted-foreground text-xs">
                                No grades available.
                            </span>
                        )}
                    </div>
                )}
            </CardContent>
        </Card>
    );
}

// ── Term Grade Card ─────────────────────────────────────────────────────

function TermGradeCard({ grade }: { grade: StudentTermGrade }) {
    return (
        <Card>
            <CardHeader className="pb-2">
                <div className="flex items-center justify-between">
                    <div>
                        <CardTitle className="text-sm font-semibold">
                            {grade.learning_area_name}
                        </CardTitle>
                        <CardDescription className="text-xs">
                            {grade.learning_area_code}
                        </CardDescription>
                    </div>
                    <PerformanceLevelBadge level={grade.final_level} showLabel />
                </div>
            </CardHeader>
            <CardContent>
                <div className="text-muted-foreground flex items-center gap-1.5 text-xs">
                    <BookOpen className="h-3 w-3" />
                    <span>
                        Based on <strong>{grade.assessment_count}</strong> assessment
                        {grade.assessment_count !== 1 ? "s" : ""}
                    </span>
                </div>
            </CardContent>
        </Card>
    );
}

// ── Main Component ──────────────────────────────────────────────────────

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
