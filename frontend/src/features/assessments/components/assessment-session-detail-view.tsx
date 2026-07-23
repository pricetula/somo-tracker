/**
 * AssessmentSessionDetailView — View and manage a single assessment session.
 *
 * Shows session info, current scores/grades, and context-appropriate actions:
 *   - DRAFT:         Grading sheet input + Submit button
 *   - PENDING_APPROVAL: Admin approve/reject actions (read-only for teachers)
 *   - PUBLISHED:     Read-only results with performance levels
 */

"use client";

import { useQuery } from "@tanstack/react-query";
import { Calendar, FileSpreadsheet, FileCheck, AlertTriangle } from "lucide-react";

import { getSession, getStudentScores, getOutcomeGrades } from "@/lib/api/assessments";
import { useSubmitSession } from "../hooks/use-assessments";
import { PerformanceLevelBadge } from "./performance-level-badge";
import { StatusBadge } from "./status-badge";
import { ApprovalActions } from "./approval-actions";
import { EVALUATION_METHOD_LABELS } from "../types";
import { getErrorMessage } from "@/lib/errors";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { StaticTable } from "@/components/shared/static-table";
import { toast } from "sonner";

interface Props {
    sessionId: string;
}

function useSessionDetail(id: string) {
    return useQuery({
        queryKey: ["assessment-sessions", id],
        queryFn: () => getSession(id),
        enabled: !!id,
        staleTime: 15_000,
    });
}

export function AssessmentSessionDetailView({ sessionId }: Props) {
    const { data: session, isLoading, isError } = useSessionDetail(sessionId);
    const submitMutation = useSubmitSession();

    // Fetch scores or grades depending on method
    const { data: scoresData } = useQuery({
        queryKey: ["assessment-scores", sessionId],
        queryFn: () => getStudentScores(sessionId),
        enabled: !!sessionId && session?.evaluation_method === "QUANTITATIVE",
    });

    const { data: gradesData } = useQuery({
        queryKey: ["outcome-grades", sessionId],
        queryFn: () => getOutcomeGrades(sessionId),
        enabled: !!sessionId && session?.evaluation_method === "RUBRIC",
    });

    if (isLoading) {
        return (
            <div className="space-y-6">
                <Skeleton className="h-8 w-72" />
                <Skeleton className="h-48 w-full" />
            </div>
        );
    }

    if (isError || !session) {
        return (
            <p className="text-destructive py-8 text-center">
                {isError ? "Failed to load session." : "Session not found."}
            </p>
        );
    }

    const isQuantitative = session.evaluation_method === "QUANTITATIVE";
    const scores = scoresData?.items ?? [];
    const grades = gradesData?.items ?? [];
    const isTeacherEditable = session.status === "DRAFT";

    return (
        <article className="space-y-6">
            {/* Header */}
            <div className="flex flex-wrap items-start justify-between gap-4">
                <div className="space-y-1">
                    <div className="flex items-center gap-2">
                        <h1 className="text-lg font-semibold">{session.name}</h1>
                        <StatusBadge status={session.status} />
                    </div>
                    <div className="text-muted-foreground flex flex-wrap gap-x-4 gap-y-1 text-xs">
                        <span className="flex items-center gap-1">
                            {isQuantitative ? (
                                <FileSpreadsheet className="h-3.5 w-3.5" />
                            ) : (
                                <FileCheck className="h-3.5 w-3.5" />
                            )}
                            {EVALUATION_METHOD_LABELS[session.evaluation_method]}
                        </span>
                        {session.max_points && <span>Max: {session.max_points} pts</span>}
                        {session.scheduled_date && (
                            <span className="flex items-center gap-1">
                                <Calendar className="h-3.5 w-3.5" />
                                {session.scheduled_date}
                            </span>
                        )}
                    </div>
                </div>
            </div>

            {/* Rejection comment */}
            {session.rejection_comment && (
                <div className="flex items-start gap-2 rounded-md border border-amber-200 bg-amber-50 p-3 dark:bg-amber-950/20">
                    <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-amber-600" />
                    <div>
                        <p className="font-medium text-amber-800 dark:text-amber-300">
                            Rejected — needs revision
                        </p>
                        <p className="text-amber-700 dark:text-amber-400">
                            {session.rejection_comment}
                        </p>
                    </div>
                </div>
            )}

            {/* Scores / Grades table */}
            {isQuantitative ? (
                <div className="space-y-2">
                    <h2 className="font-medium">Student Scores</h2>
                    {scores.length === 0 ? (
                        <p className="text-muted-foreground py-4 text-center text-xs">
                            {isTeacherEditable
                                ? "No scores entered yet. Use the grading sheet to add scores."
                                : "No scores recorded for this session."}
                        </p>
                    ) : (
                        <StaticTable
                            columns={[
                                {
                                    id: "student_id",
                                    header: "Student ID",
                                    cell: (r) => r.student_id,
                                },
                                {
                                    id: "raw_score",
                                    header: "Score",
                                    align: "right",
                                    cell: (r) =>
                                        r.raw_score != null ? (
                                            <span className="tabular-nums">
                                                {r.raw_score}
                                                {session.max_points
                                                    ? ` / ${session.max_points}`
                                                    : ""}
                                            </span>
                                        ) : (
                                            <span className="text-muted-foreground">\u2014</span>
                                        ),
                                },
                                {
                                    id: "percentage",
                                    header: "%",
                                    align: "right",
                                    cell: (r) =>
                                        r.calculated_percentage != null ? (
                                            <span className="tabular-nums">
                                                {r.calculated_percentage.toFixed(1)}%
                                            </span>
                                        ) : (
                                            <span className="text-muted-foreground">\u2014</span>
                                        ),
                                },
                                {
                                    id: "level",
                                    header: "Level",
                                    cell: (r) => (
                                        <PerformanceLevelBadge level={r.final_performance_level} />
                                    ),
                                },
                            ]}
                            data={scores}
                            getRowId={(r) => r.id}
                            height={Math.min(scores.length * 44 + 44, 480)}
                        />
                    )}
                </div>
            ) : (
                <div className="space-y-2">
                    <h2 className="font-medium">Rubric Grades</h2>
                    {grades.length === 0 ? (
                        <p className="text-muted-foreground py-4 text-center text-xs">
                            {isTeacherEditable
                                ? "No grades entered yet."
                                : "No rubric grades recorded for this session."}
                        </p>
                    ) : (
                        <StaticTable
                            columns={[
                                {
                                    id: "student_id",
                                    header: "Student ID",
                                    cell: (r) => r.student_id,
                                },
                                {
                                    id: "indicator",
                                    header: "Indicator",
                                    cell: (r) => r.performance_indicator_id,
                                },
                                {
                                    id: "level",
                                    header: "Awarded",
                                    cell: (r) => <PerformanceLevelBadge level={r.awarded_level} />,
                                },
                            ]}
                            data={grades}
                            getRowId={(r) => r.id}
                            height={Math.min(grades.length * 44 + 44, 480)}
                        />
                    )}
                </div>
            )}

            {/* Actions */}
            <div className="flex items-center justify-between border-t pt-4">
                {/* Teacher: Submit for approval */}
                {session.status === "DRAFT" && (
                    <Button
                        size="sm"
                        onClick={() =>
                            submitMutation.mutate(sessionId, {
                                onError: (err) => toast.error(getErrorMessage(err)),
                            })
                        }
                        disabled={submitMutation.isPending}
                    >
                        {submitMutation.isPending ? "Submitting..." : "Submit for Approval"}
                    </Button>
                )}

                {/* Admin: Approve / Reject */}
                <ApprovalActions sessionId={sessionId} status={session.status} />
            </div>
        </article>
    );
}
