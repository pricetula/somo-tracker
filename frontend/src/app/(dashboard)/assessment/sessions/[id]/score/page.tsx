/**
 * Scoring Grid page — full-page matrix for recording learner rubric results.
 *
 * Loads session detail (for blueprint indicators), then fetches students
 * from the class and allows batch rubric level selection.
 *
 * Maps to:
 *   GET  /api/v1/assessment/sessions/:id — session detail (blueprint_id, class_id)
 *   GET  /api/v1/students/list?class_id=X — students in the class
 *   GET  /api/v1/assessment/sessions/:id/results — existing saved results
 *   POST /api/v1/assessment/sessions/:id/results/batch — save results
 */

"use client";

import * as React from "react";
import { useParams, useRouter } from "next/navigation";

import { Button } from "@/components/ui/button";
import { ArrowLeft } from "lucide-react";

import {
    ScoringGrid,
    useSessionDetail,
    useSessionResults,
    useBatchUpsertResults,
    useBlueprintDetail,
} from "@/features/assessment";
import { useStudents } from "@/features/students";
import type { LearnerRubricResult } from "@/features/assessment";

export default function ScorePage() {
    const params = useParams();
    const router = useRouter();
    const sessionId = params.id as string;

    // Fetch session detail (gives us blueprint_id and class_id)
    const {
        data: sessionData,
        isLoading: sessionLoading,
        isError: sessionError,
    } = useSessionDetail(sessionId);

    const session = sessionData?.data;
    const blueprintId = session?.blueprint_id ?? "";
    const classId = session?.class_id ?? "";

    // Fetch blueprint detail for linked indicators
    const { data: blueprintData, isLoading: blueprintLoading } = useBlueprintDetail(blueprintId, {
        enabled: !!blueprintId,
    });

    const indicators = blueprintData?.data?.indicators ?? [];

    // Fetch saved results
    const { data: resultsData, isLoading: resultsLoading } = useSessionResults(sessionId, {
        enabled: !!sessionId,
    });

    const savedResults = React.useMemo(() => resultsData?.data ?? [], [resultsData]);

    // Fetch students in the class via React Query
    const {
        data: studentsData,
        isLoading: studentsLoading,
        isError: studentsError,
    } = useStudents({ class_id: classId, limit: 200 }, { enabled: !!classId });

    const students = studentsData?.students ?? [];

    const batchUpsert = useBatchUpsertResults();

    // Build a lookup map: "student_id:indicator_id" → LearnerRubricResult
    const resultsMap = React.useMemo(() => {
        const map = new Map<string, LearnerRubricResult>();
        for (const r of savedResults) {
            map.set(`${r.student_id}:${r.indicator_id}`, r);
        }
        return map;
    }, [savedResults]);

    const handleSave = async (
        results: Array<{
            student_id: string;
            indicator_id: string;
            rubric_level: string;
            score_type: string;
            raw_score?: string | null;
        }>
    ) => {
        await batchUpsert.mutateAsync({
            sessionId,
            data: { results },
        });
    };

    const isLoading = sessionLoading || blueprintLoading || resultsLoading || studentsLoading;

    if (sessionError) {
        return (
            <div className="flex flex-col items-center justify-center py-16">
                <p className="text-destructive text-sm font-medium">
                    Failed to load assessment session.
                </p>
                <Button
                    variant="outline"
                    size="sm"
                    className="mt-4"
                    onClick={() => router.push("/assessment/sessions")}
                >
                    Back to Sessions
                </Button>
            </div>
        );
    }

    return (
        <div className="flex flex-1 flex-col px-6 pt-6 pb-8">
            {/* Back link */}
            <Button
                variant="ghost"
                size="sm"
                className="mb-4 w-fit"
                onClick={() => router.push(`/assessment/sessions/${sessionId}`)}
            >
                <ArrowLeft className="mr-1.5 size-4" />
                Back to Session
            </Button>

            {/* Page header */}
            <div className="mb-6">
                <h1 className="text-2xl font-semibold tracking-tight">Scoring Grid</h1>
                <p className="text-muted-foreground mt-1 text-sm">
                    {isLoading
                        ? "Loading session data…"
                        : `Recording scores for ${students.length} student${students.length !== 1 ? "s" : ""} across ${indicators.length} indicator${indicators.length !== 1 ? "s" : ""}.`}
                </p>
            </div>

            {studentsError && (
                <div className="bg-destructive/10 text-destructive mb-4 rounded-md px-3 py-2 text-sm">
                    Failed to load students. Please try again.
                </div>
            )}

            <ScoringGrid
                students={students.map((s) => ({ id: s.id, full_name: s.full_name }))}
                indicators={indicators}
                savedResults={resultsMap}
                isLoading={isLoading}
                onSave={handleSave}
                isSaving={batchUpsert.isPending}
            />
        </div>
    );
}
