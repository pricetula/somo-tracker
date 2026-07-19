/**
 * RubricGradingMatrix — Editable grid for RUBRIC sessions.
 *
 * Rows = enrolled students, Columns = performance indicators (from the
 * session's learning area tree), Cells = EE/ME/AE/BE dropdown.
 * Teachers assign a level to each (student × indicator) combination.
 * Only editable when the session is in DRAFT status.
 *
 * Roster and existing grades are fetched from GET /assessments/sessions/:id/grading-data
 * — no separate class roster call needed.
 */

"use client";

import { useState, useCallback, useEffect } from "react";
import { Loader2, Save } from "lucide-react";

import { getGradingData, type GradingDataStudent } from "@/lib/api/assessments";
import { getLearningAreaTree } from "@/lib/api/curriculum";
import type { PerformanceIndicator } from "@/lib/api/curriculum";
import { useBulkUpsertOutcomeGrades } from "../hooks/use-assessments";
import { PERFORMANCE_LEVELS, PERFORMANCE_LEVEL_LABELS } from "../types";
import { getErrorMessage, isApiError } from "@/lib/errors";
import { Button } from "@/components/ui/button";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { toast } from "sonner";

interface Props {
    sessionId: string;
    learningAreaId: string;
    status: string;
}

/** Valid rubric levels */
type RubricLevel = "EE" | "ME" | "AE" | "BE";

/** Type guard: is the string a valid CBC rubric level? */
function isRubricLevel(value: string): value is RubricLevel {
    return (PERFORMANCE_LEVELS as readonly string[]).includes(value);
}

/** Flat map: (student_id, indicator_id) → awarded_level */
type GradeDraft = Record<string, string>;

/** Key for the draft map */
function draftKey(studentId: string, indicatorId: string): string {
    return `${studentId}::${indicatorId}`;
}

/** Flatten learning area tree to all indicators */
function flattenIndicators(tree: {
    strands?: { sub_strands?: { performance_indicators?: PerformanceIndicator[] }[] }[];
}): PerformanceIndicator[] {
    const all: PerformanceIndicator[] = [];
    for (const strand of tree.strands ?? []) {
        for (const sub of strand.sub_strands ?? []) {
            for (const pi of sub.performance_indicators ?? []) {
                all.push(pi);
            }
        }
    }
    return all;
}

export function RubricGradingMatrix({ sessionId, learningAreaId, status }: Props) {
    const isEditable = status === "DRAFT";
    const saveMutation = useBulkUpsertOutcomeGrades();

    const [students, setStudents] = useState<GradingDataStudent[]>([]);
    const [studentsLoading, setStudentsLoading] = useState(true);
    const [studentsError, setStudentsError] = useState<string | null>(null);

    const [indicatorsState, setIndicatorsState] = useState<{
        loading: boolean;
        error: string | null;
        indicators: PerformanceIndicator[];
    }>({ loading: true, error: null, indicators: [] });

    const [draft, setDraft] = useState<GradeDraft>({});
    const [saveError, setSaveError] = useState<string | null>(null);
    const [success, setSuccess] = useState(false);

    useEffect(() => {
        let cancelled = false;

        async function load() {
            try {
                // Fetch grading data (roster + grades) and learning area tree in parallel
                const [data, tree] = await Promise.all([
                    getGradingData(sessionId),
                    getLearningAreaTree(learningAreaId),
                ]);

                if (cancelled) return;

                setStudents(data.students);
                setStudentsLoading(false);
                setStudentsError(null);

                const indicators = flattenIndicators(tree);
                setIndicatorsState({ loading: false, error: null, indicators });

                // Initialise draft from existing grades (embedded in each student)
                const draftFromGrades: GradeDraft = {};
                for (const s of data.students) {
                    for (const g of s.grades ?? []) {
                        draftFromGrades[draftKey(s.student_id, g.performance_indicator_id)] =
                            g.awarded_level;
                    }
                }
                setDraft(draftFromGrades);
            } catch (err) {
                if (cancelled) return;
                const msg = getErrorMessage(err);
                setStudentsError(msg);
                setStudentsLoading(false);
                setIndicatorsState((prev) => ({ ...prev, loading: false, error: msg }));
            }
        }

        load();
        return () => {
            cancelled = true;
        };
    }, [sessionId, learningAreaId]);

    // ── Update a single cell ──────────────────────────────────────────
    const updateCell = useCallback((studentId: string, indicatorId: string, level: string) => {
        setDraft((prev) => ({ ...prev, [draftKey(studentId, indicatorId)]: level }));
        setSaveError(null);
        setSuccess(false);
    }, []);

    // ── Save ──────────────────────────────────────────────────────────
    const handleSave = useCallback(() => {
        const grades: {
            student_id: string;
            performance_indicator_id: string;
            awarded_level: RubricLevel;
        }[] = [];

        for (const student of students) {
            for (const indicator of indicatorsState.indicators) {
                const key = draftKey(student.student_id, indicator.id);
                const level = draft[key];
                if (level && isRubricLevel(level)) {
                    grades.push({
                        student_id: student.student_id,
                        performance_indicator_id: indicator.id,
                        awarded_level: level,
                    });
                }
            }
        }

        if (grades.length === 0) {
            setSaveError("Assign at least one grade before saving.");
            return;
        }

        saveMutation.mutate(
            { sessionId, payload: { grades } },
            {
                onSuccess: () => {
                    toast.success("Grades saved successfully.");
                    setSaveError(null);
                    setSuccess(true);
                },
                onError: (err) => {
                    if (isApiError(err) && err.status === 400 && err.errors) {
                        setSaveError(Object.values(err.errors).flat().join(", "));
                    } else {
                        setSaveError(getErrorMessage(err));
                    }
                },
            }
        );
    }, [draft, students, indicatorsState.indicators, sessionId, saveMutation]);

    // ── Level colour map (mirrors PerformanceLevelBadge) ──────────────
    const levelColors: Record<string, string> = {
        EE: "border-emerald-400 text-emerald-700 bg-emerald-50 dark:bg-emerald-950/20 dark:text-emerald-400",
        ME: "border-blue-400 text-blue-700 bg-blue-50 dark:bg-blue-950/20 dark:text-blue-400",
        AE: "border-amber-400 text-amber-700 bg-amber-50 dark:bg-amber-950/20 dark:text-amber-400",
        BE: "border-rose-400 text-rose-700 bg-rose-50 dark:bg-rose-950/20 dark:text-rose-400",
    };

    // ── Loading state ────────────────────────────────────────────────
    if (studentsLoading || indicatorsState.loading) {
        return (
            <div className="space-y-3">
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-10 w-full" />
            </div>
        );
    }

    // ── Error state ──────────────────────────────────────────────────
    const error = studentsError || indicatorsState.error;
    if (error) {
        return (
            <Alert variant="destructive">
                <AlertDescription>Failed to load data: {error}</AlertDescription>
            </Alert>
        );
    }

    // ── Empty states ─────────────────────────────────────────────────
    if (students.length === 0) {
        return (
            <Alert>
                <AlertDescription>
                    No students are enrolled in this class. Enroll students before assigning grades.
                </AlertDescription>
            </Alert>
        );
    }

    if (indicatorsState.indicators.length === 0) {
        return (
            <Alert>
                <AlertDescription>
                    No performance indicators found for this learning area. Add indicators to the
                    curriculum before using rubric grading.
                </AlertDescription>
            </Alert>
        );
    }

    // ── Read-only view ───────────────────────────────────────────────
    if (!isEditable) {
        const hasGrades = students.some((s) => (s.grades?.length ?? 0) > 0);
        if (!hasGrades) {
            return (
                <p className="text-muted-foreground py-4 text-center text-xs">
                    No rubric grades recorded for this session.
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
                            {indicatorsState.indicators.map((indicator) => (
                                <th
                                    key={indicator.id}
                                    className="text-muted-foreground min-w-[120px] px-3 py-2 text-left text-xs font-medium"
                                >
                                    <span title={indicator.description}>
                                        {indicator.description.length > 30
                                            ? `${indicator.description.slice(0, 30)}...`
                                            : indicator.description}
                                    </span>
                                </th>
                            ))}
                        </tr>
                    </thead>
                    <tbody className="divide-y">
                        {students.map((student) => (
                            <tr key={student.student_id} className="hover:bg-muted/30">
                                <td className="px-3 py-2 font-medium">{student.student_name}</td>
                                {indicatorsState.indicators.map((indicator) => {
                                    const grade = student.grades?.find(
                                        (g) => g.performance_indicator_id === indicator.id
                                    );
                                    return (
                                        <td key={indicator.id} className="px-3 py-2">
                                            {grade ? (
                                                <Badge
                                                    variant="secondary"
                                                    className={
                                                        levelColors[grade.awarded_level] ??
                                                        "bg-muted text-muted-foreground"
                                                    }
                                                >
                                                    {grade.awarded_level}
                                                </Badge>
                                            ) : (
                                                <span className="text-muted-foreground">
                                                    \u2014
                                                </span>
                                            )}
                                        </td>
                                    );
                                })}
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

            {success && (
                <Alert>
                    <AlertDescription>Grades saved successfully.</AlertDescription>
                </Alert>
            )}

            <ScrollArea className="rounded-md border">
                <div className="min-w-max">
                    <table className="w-full text-sm">
                        <thead className="bg-muted/50">
                            <tr>
                                <th className="text-muted-foreground bg-muted/50 sticky left-0 z-10 px-3 py-2 text-left text-xs font-medium">
                                    Student
                                </th>
                                {indicatorsState.indicators.map((indicator) => (
                                    <th
                                        key={indicator.id}
                                        className="text-muted-foreground min-w-[160px] px-3 py-2 text-left text-xs font-medium"
                                    >
                                        <div className="flex flex-col">
                                            <span>
                                                {indicator.description.length > 35
                                                    ? `${indicator.description.slice(0, 35)}...`
                                                    : indicator.description}
                                            </span>
                                            <span className="text-muted-foreground/60 mt-0.5 text-[10px]">
                                                #{indicator.sequence_order}
                                            </span>
                                        </div>
                                    </th>
                                ))}
                            </tr>
                        </thead>
                        <tbody className="divide-y">
                            {students.map((student) => (
                                <tr key={student.student_id} className="hover:bg-muted/30">
                                    <td className="text-muted-foreground bg-background sticky left-0 z-10 px-3 py-2 text-xs font-medium">
                                        {student.student_name}
                                    </td>
                                    {indicatorsState.indicators.map((indicator) => {
                                        const key = draftKey(student.student_id, indicator.id);
                                        const value = draft[key] ?? "";
                                        return (
                                            <td key={indicator.id} className="px-2 py-1.5">
                                                <Select
                                                    value={value}
                                                    onValueChange={(v) =>
                                                        updateCell(
                                                            student.student_id,
                                                            indicator.id,
                                                            v
                                                        )
                                                    }
                                                    disabled={saveMutation.isPending}
                                                >
                                                    <SelectTrigger
                                                        className={`h-8 text-xs ${
                                                            value
                                                                ? (levelColors[value] ??
                                                                  "border-border")
                                                                : "text-muted-foreground border-dashed"
                                                        }`}
                                                    >
                                                        <SelectValue placeholder="\u2014" />
                                                    </SelectTrigger>
                                                    <SelectContent>
                                                        {PERFORMANCE_LEVELS.map((level) => (
                                                            <SelectItem
                                                                key={level}
                                                                value={level}
                                                                className="text-xs"
                                                            >
                                                                <span className="flex items-center gap-2">
                                                                    <span className="font-semibold">
                                                                        {level}
                                                                    </span>
                                                                    <span className="text-muted-foreground">
                                                                        {
                                                                            PERFORMANCE_LEVEL_LABELS[
                                                                                level
                                                                            ]
                                                                        }
                                                                    </span>
                                                                </span>
                                                            </SelectItem>
                                                        ))}
                                                    </SelectContent>
                                                </Select>
                                            </td>
                                        );
                                    })}
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>
            </ScrollArea>

            <div className="flex items-center justify-between">
                <p className="text-muted-foreground text-xs">
                    {Object.values(draft).filter((v) => v !== "").length} grades assigned across{" "}
                    {students.length} students × {indicatorsState.indicators.length} indicators
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
                            Save Grades
                        </>
                    )}
                </Button>
            </div>
        </div>
    );
}
