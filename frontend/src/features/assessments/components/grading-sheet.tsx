/**
 * GradingSheet — Editable score entry matrix for QUANTITATIVE sessions.
 *
 * Shows a table of enrolled students with raw score inputs. Teachers enter
 * scores, see a live preview of the calculated percentage, and save in bulk.
 * Only editable when the session is in DRAFT status.
 *
 * Data is fetched from GET /api/v1/assessments/sessions/:id/grading-data,
 * which returns the session, roster, and existing scores merged into one
 * response — no separate class roster call needed.
 */

"use client";

import { useState, useCallback, useEffect } from "react";
import { Loader2, Save } from "lucide-react";

import { getGradingData, type GradingDataStudent } from "@/lib/api/assessments";
import { useBulkUpsertScores } from "../hooks/use-assessments";
import { PerformanceLevelBadge } from "./performance-level-badge";
import { getErrorMessage, isApiError } from "@/lib/errors";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { toast } from "sonner";

interface Props {
    sessionId: string;
    maxPoints: number;
    status: string;
    classId: string;
    academicTermId: string;
}

/** Map of student_id → raw_score string for local editing state. */
type ScoreDraft = Record<string, string>;

export function GradingSheet({ sessionId, maxPoints, status, classId, academicTermId }: Props) {
    const isEditable = status === "DRAFT";
    const saveMutation = useBulkUpsertScores();

    const [students, setStudents] = useState<GradingDataStudent[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    const [draft, setDraft] = useState<ScoreDraft>({});
    const [fieldErrors, setFieldErrors] = useState<Record<string, string[]>>({});
    const [saveError, setSaveError] = useState<string | null>(null);

    // Fetch grading data (session + roster + scores in one call)
    useEffect(() => {
        let cancelled = false;

        async function load() {
            try {
                const data = await getGradingData(sessionId);
                if (cancelled) return;

                setStudents(data.students);
                setLoading(false);
                setError(null);

                // Initialise draft from existing scores
                const draftFromScores: ScoreDraft = {};
                for (const s of data.students) {
                    if (s.score?.raw_score != null) {
                        draftFromScores[s.student_id] = String(s.score.raw_score);
                    }
                }
                setDraft(draftFromScores);
            } catch (err) {
                if (cancelled) return;
                setError(getErrorMessage(err));
                setLoading(false);
            }
        }

        load();
        return () => {
            cancelled = true;
        };
    }, [sessionId]);

    // ── Update a single student's score ──────────────────────────────
    const updateScore = useCallback((studentId: string, value: string) => {
        setDraft((prev) => ({ ...prev, [studentId]: value }));
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

        for (const student of students) {
            const val = draft[student.student_id];
            if (val !== undefined && val.trim() !== "") {
                hasValue = true;
                const num = parseFloat(val);
                if (isNaN(num) || num < 0) {
                    errors[student.student_id] = ["Must be ≥ 0"];
                } else if (num > maxPoints) {
                    errors[student.student_id] = [`Max ${maxPoints}`];
                }
            }
        }

        if (!hasValue) {
            setSaveError("Enter at least one score before saving.");
            return false;
        }

        setFieldErrors(errors);
        return Object.keys(errors).length === 0;
    }, [draft, maxPoints, students]);

    // ── Save ────────────────────────────────────────────────────────
    const handleSave = useCallback(() => {
        if (!validate()) return;

        const scores: { student_id: string; raw_score: number }[] = [];
        for (const student of students) {
            const val = draft[student.student_id];
            if (val === undefined || val.trim() === "") continue;
            const num = parseFloat(val);
            if (isNaN(num)) continue;
            scores.push({ student_id: student.student_id, raw_score: num });
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
    }, [validate, draft, students, sessionId, saveMutation]);

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

    // ── Render states ────────────────────────────────────────────────
    if (loading) {
        return (
            <div className="space-y-3">
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-10 w-1/2" />
            </div>
        );
    }

    if (error) {
        return (
            <Alert variant="destructive">
                <AlertDescription>Failed to load grading data: {error}</AlertDescription>
            </Alert>
        );
    }

    if (students.length === 0) {
        const enrollUrl = `/classes/${classId}/enroll?academictermid=${academicTermId}`;
        return (
            <Alert>
                <AlertDescription>
                    No students are enrolled in this class.{" "}
                    <Link
                        href={enrollUrl}
                        className="underline underline-offset-2 hover:text-blue-600"
                    >
                        Enroll students
                    </Link>{" "}
                    before entering scores.
                </AlertDescription>
            </Alert>
        );
    }

    // ── Read-only view (published / pending approval) ────────────────
    if (!isEditable) {
        const scored = students.filter((s) => s.score?.raw_score != null);
        if (scored.length === 0) {
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
                        {scored.map((s) => (
                            <tr key={s.student_id} className="hover:bg-muted/30">
                                <td className="px-3 py-2 font-medium">{s.student_name}</td>
                                <td className="px-3 py-2 text-right tabular-nums">
                                    {s.score?.raw_score != null
                                        ? `${s.score.raw_score} / ${maxPoints}`
                                        : "-"}
                                </td>
                                <td className="px-3 py-2 text-right tabular-nums">
                                    {s.score?.calculated_percentage != null
                                        ? `${s.score.calculated_percentage.toFixed(1)}%`
                                        : "-"}
                                </td>
                                <td className="px-3 py-2">
                                    <PerformanceLevelBadge
                                        level={s.score?.final_performance_level ?? null}
                                    />
                                </td>
                            </tr>
                        ))}
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
                        {students.map((student) => {
                            const sid = student.student_id;
                            const val = draft[sid] ?? "";
                            const pct = previewPercentage(val);
                            const existingScore = student.score;
                            const hasError = !!fieldErrors[sid];
                            const errorMsg = fieldErrors[sid]?.[0];

                            return (
                                <tr
                                    key={sid}
                                    className={`hover:bg-muted/30 ${hasError ? "bg-destructive/5" : ""}`}
                                >
                                    <td className="px-3 py-2 font-medium">
                                        {student.student_name}
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
                                                        updateScore(sid, e.target.value)
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
                                            <span className="text-foreground">
                                                {pct.toFixed(1)}%
                                            </span>
                                        ) : (
                                            <span className="text-muted-foreground">-</span>
                                        )}
                                    </td>
                                    <td className="px-3 py-2">
                                        <PerformanceLevelBadge
                                            level={existingScore?.final_performance_level ?? null}
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
                    {students.length} students scored
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
