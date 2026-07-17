/**
 * GradingSheet — Editable score entry matrix for QUANTITATIVE sessions.
 *
 * Shows a table of enrolled students with raw score inputs. Teachers enter
 * scores, see a live preview of the calculated percentage, and save in bulk.
 * Only editable when the session is in DRAFT status.
 */

"use client";

import { useState, useCallback, useEffect } from "react";
import { Loader2, Save } from "lucide-react";

import { getClassRoster, type RosterEntry } from "@/lib/api/classes";
import { getStudentScores, type StudentScore } from "@/lib/api/assessments";
import { useBulkUpsertScores } from "../hooks/use-assessments";
import { PerformanceLevelBadge } from "./performance-level-badge";
import { getErrorMessage, isApiError } from "@/lib/errors";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { toast } from "sonner";

interface Props {
    sessionId: string;
    classId: string;
    maxPoints: number;
    status: string;
    academicTermId: string;
}

/** Map of student_id → raw_score string for local editing state. */
type ScoreDraft = Record<string, string>;

export function GradingSheet({ sessionId, classId, maxPoints, status, academicTermId }: Props) {
    const isEditable = status === "DRAFT";
    const saveMutation = useBulkUpsertScores();

    const [rosterState, setRosterState] = useState<{
        loading: boolean;
        error: string | null;
        students: RosterEntry[];
    }>({ loading: true, error: null, students: [] });

    const [scoresState, setScoresState] = useState<{
        loading: boolean;
        error: string | null;
        existing: StudentScore[];
    }>({ loading: true, error: null, existing: [] });

    const [draft, setDraft] = useState<ScoreDraft>({});
    const [fieldErrors, setFieldErrors] = useState<Record<string, string[]>>({});
    const [saveError, setSaveError] = useState<string | null>(null);

    // Fetch roster and scores, initialise draft from existing scores
    useEffect(() => {
        let cancelled = false;

        async function load() {
            try {
                const [roster, scores] = await Promise.all([
                    getClassRoster(classId, {
                        academic_term_id: academicTermId,
                        limit: 200,
                    }),
                    getStudentScores(sessionId),
                ]);

                if (cancelled) return;

                const students = roster.items ?? [];
                const existing = scores.items ?? [];

                setRosterState({ loading: false, error: null, students });
                setScoresState({ loading: false, error: null, existing });

                // Initialise draft from existing scores
                const draftFromScores: ScoreDraft = {};
                for (const s of existing) {
                    if (s.raw_score != null) {
                        draftFromScores[s.student_id] = String(s.raw_score);
                    }
                }
                setDraft(draftFromScores);
            } catch (err) {
                if (cancelled) return;
                const msg = getErrorMessage(err);
                setRosterState((prev) => ({ ...prev, loading: false, error: msg }));
                setScoresState((prev) => ({ ...prev, loading: false, error: msg }));
            }
        }

        load();
        return () => {
            cancelled = true;
        };
    }, [classId, sessionId, academicTermId]);

    // ── Update a single student's score ──────────────────────────────
    const updateScore = useCallback((studentId: string, value: string) => {
        setDraft((prev) => ({ ...prev, [studentId]: value }));
        // Clear field error on edit
        setFieldErrors((prev) => {
            if (prev[studentId]) {
                const next = { ...prev };
                delete next[studentId];
                return next;
            }
            return prev;
        });
        setSaveError(null);
    }, []);

    // ── Validate ────────────────────────────────────────────────────
    const validate = useCallback((): boolean => {
        const errors: Record<string, string[]> = {};
        let hasValue = false;

        for (const student of rosterState.students) {
            const val = draft[student.id];
            if (val !== undefined && val.trim() !== "") {
                hasValue = true;
                const num = parseFloat(val);
                if (isNaN(num) || num < 0) {
                    errors[student.id] = ["Must be ≥ 0"];
                } else if (num > maxPoints) {
                    errors[student.id] = [`Max ${maxPoints}`];
                }
            }
        }

        if (!hasValue) {
            setSaveError("Enter at least one score before saving.");
            return false;
        }

        setFieldErrors(errors);
        return Object.keys(errors).length === 0;
    }, [draft, maxPoints, rosterState.students]);

    // ── Save ────────────────────────────────────────────────────────
    const handleSave = useCallback(() => {
        if (!validate()) return;

        const scores: { student_id: string; raw_score: number }[] = [];
        for (const student of rosterState.students) {
            const val = draft[student.id];
            if (val === undefined || val.trim() === "") continue;
            const num = parseFloat(val);
            if (isNaN(num)) continue;
            scores.push({ student_id: student.id, raw_score: num });
        }

        saveMutation.mutate(
            { sessionId, payload: { scores } },
            {
                onSuccess: () => {
                    toast.success("Scores saved successfully.");
                    setSaveError(null);
                },
                onError: (err) => {
                    if (isApiError(err) && err.status === 400 && err.errors) {
                        setFieldErrors(err.errors);
                    } else {
                        setSaveError(getErrorMessage(err));
                    }
                },
            }
        );
    }, [validate, draft, rosterState.students, sessionId, saveMutation]);

    // ── Preview percentage helper ────────────────────────────────────
    const previewPercentage = useCallback(
        (rawScoreStr: string): number | null => {
            if (!rawScoreStr || rawScoreStr.trim() === "") return null;
            const num = parseFloat(rawScoreStr);
            if (isNaN(num) || maxPoints <= 0) return null;
            return (num / maxPoints) * 100;
        },
        [maxPoints]
    );

    // ── Find existing score for a student ────────────────────────────
    const existingScore = useCallback(
        (studentId: string): StudentScore | undefined => {
            return scoresState.existing.find((s) => s.student_id === studentId);
        },
        [scoresState.existing]
    );

    // ── Render states ────────────────────────────────────────────────
    if (rosterState.loading || scoresState.loading) {
        return (
            <div className="space-y-3">
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-10 w-1/2" />
            </div>
        );
    }

    if (rosterState.error) {
        return (
            <Alert variant="destructive">
                <AlertDescription>
                    Failed to load class roster: {rosterState.error}
                </AlertDescription>
            </Alert>
        );
    }

    if (rosterState.students.length === 0) {
        return (
            <Alert>
                <AlertDescription>
                    No students are enrolled in this class. Enroll students before entering scores.
                </AlertDescription>
            </Alert>
        );
    }

    // ── Read-only view (published / pending approval) ────────────────
    if (!isEditable) {
        if (scoresState.existing.length === 0) {
            return (
                <p className="text-muted-foreground py-4 text-center text-xs">
                    No scores recorded for this session.
                </p>
            );
        }

        return (
            <div className="overflow-x-auto rounded-md border">
                <table className="w-full text-sm">
                    <thead className="bg-muted/50">
                        <tr>
                            <th className="text-muted-foreground px-3 py-2 text-left text-xs font-medium">
                                Student
                            </th>
                            <th className="text-muted-foreground px-3 py-2 text-right text-xs font-medium">
                                Score
                            </th>
                            <th className="text-muted-foreground px-3 py-2 text-right text-xs font-medium">
                                %
                            </th>
                            <th className="text-muted-foreground px-3 py-2 text-left text-xs font-medium">
                                Level
                            </th>
                        </tr>
                    </thead>
                    <tbody className="divide-y">
                        {scoresState.existing.map((score) => {
                            const student = rosterState.students.find(
                                (s) => s.id === score.student_id
                            );
                            return (
                                <tr key={score.id} className="hover:bg-muted/30">
                                    <td className="px-3 py-2 font-medium">
                                        {student?.full_name ?? score.student_id}
                                    </td>
                                    <td className="px-3 py-2 text-right tabular-nums">
                                        {score.raw_score != null
                                            ? `${score.raw_score} / ${maxPoints}`
                                            : "\u2014"}
                                    </td>
                                    <td className="px-3 py-2 text-right tabular-nums">
                                        {score.calculated_percentage != null
                                            ? `${score.calculated_percentage.toFixed(1)}%`
                                            : "\u2014"}
                                    </td>
                                    <td className="px-3 py-2">
                                        <PerformanceLevelBadge
                                            level={score.final_performance_level}
                                        />
                                    </td>
                                </tr>
                            );
                        })}
                    </tbody>
                </table>
            </div>
        );
    }

    // ── Editable view ────────────────────────────────────────────────
    return (
        <div className="space-y-4">
            {saveError && (
                <Alert variant="destructive">
                    <AlertDescription>{saveError}</AlertDescription>
                </Alert>
            )}

            <div className="overflow-x-auto rounded-md border">
                <table className="w-full text-sm">
                    <thead className="bg-muted/50">
                        <tr>
                            <th className="text-muted-foreground w-1/2 px-3 py-2 text-left text-xs font-medium">
                                Student
                            </th>
                            <th className="text-muted-foreground px-3 py-2 text-right text-xs font-medium">
                                Raw Score
                                <span className="font-normal"> (max {maxPoints})</span>
                            </th>
                            <th className="text-muted-foreground px-3 py-2 text-right text-xs font-medium">
                                %
                            </th>
                            <th className="text-muted-foreground px-3 py-2 text-left text-xs font-medium">
                                Current Level
                            </th>
                        </tr>
                    </thead>
                    <tbody className="divide-y">
                        {rosterState.students.map((student) => {
                            const val = draft[student.id] ?? "";
                            const pct = previewPercentage(val);
                            const existing = existingScore(student.id);
                            const hasError = !!fieldErrors[student.id];
                            const errorMsg = fieldErrors[student.id]?.[0];

                            return (
                                <tr
                                    key={student.id}
                                    className={`hover:bg-muted/30 ${hasError ? "bg-destructive/5" : ""}`}
                                >
                                    <td className="px-3 py-2 font-medium">
                                        {student.full_name}
                                        {student.admission_number && (
                                            <span className="text-muted-foreground ml-2 text-xs">
                                                #{student.admission_number}
                                            </span>
                                        )}
                                    </td>
                                    <td className="px-3 py-2">
                                        <div className="flex items-center justify-end gap-2">
                                            <div className="w-24">
                                                <Input
                                                    type="number"
                                                    min={0}
                                                    max={maxPoints}
                                                    step={0.5}
                                                    value={val}
                                                    onChange={(e) =>
                                                        updateScore(student.id, e.target.value)
                                                    }
                                                    className={`h-8 text-right ${hasError ? "border-destructive" : ""}`}
                                                    placeholder="0"
                                                    disabled={saveMutation.isPending}
                                                />
                                            </div>
                                            {hasError && (
                                                <span className="text-destructive text-xs">
                                                    {errorMsg}
                                                </span>
                                            )}
                                        </div>
                                    </td>
                                    <td className="px-3 py-2 text-right tabular-nums">
                                        {pct !== null ? (
                                            <span
                                                className={
                                                    pct >= 0
                                                        ? "text-foreground"
                                                        : "text-muted-foreground"
                                                }
                                            >
                                                {pct.toFixed(1)}%
                                            </span>
                                        ) : (
                                            <span className="text-muted-foreground">\u2014</span>
                                        )}
                                    </td>
                                    <td className="px-3 py-2">
                                        <PerformanceLevelBadge
                                            level={existing?.final_performance_level}
                                        />
                                    </td>
                                </tr>
                            );
                        })}
                    </tbody>
                </table>
            </div>

            <div className="flex items-center justify-between">
                <p className="text-muted-foreground text-xs">
                    {Object.values(draft).filter((v) => v.trim() !== "").length} of{" "}
                    {rosterState.students.length} students scored
                </p>
                <Button size="sm" onClick={handleSave} disabled={saveMutation.isPending}>
                    {saveMutation.isPending ? (
                        <>
                            <Loader2 className="mr-1.5 h-4 w-4 animate-spin" />
                            Saving...
                        </>
                    ) : (
                        <>
                            <Save className="mr-1.5 h-4 w-4" />
                            Save Scores
                        </>
                    )}
                </Button>
            </div>
        </div>
    );
}
