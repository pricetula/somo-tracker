/**
 * CreateAssessmentSessionForm — Creates a new assessment session in DRAFT.
 *
 * Supports both QUANTITATIVE (marks-based) and RUBRIC (indicator-level) methods.
 * QUANTITATIVE requires max_points and a grading_scale_profile_id.
 * RUBRIC requires neither — the teacher assigns levels directly.
 */

"use client";

import { useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import { useMutation, useQuery } from "@tanstack/react-query";
import { Loader2 } from "lucide-react";

import { createSession, listScaleProfiles } from "@/lib/api/assessments";
import type { AssessmentSession } from "@/lib/api/assessments";
import { getErrorMessage, isApiError } from "@/lib/errors";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { DatePicker } from "@/components/ui/date-picker";
import { ClassCombobox } from "@/features/classes";
import { LearningAreaCombobox } from "@/features/curriculum";
import { AcademicYearCombobox, AcademicTermCombobox } from "@/features/academic-terms";

interface Props {
    onSuccess?: (session: AssessmentSession) => void;
}

export function CreateAssessmentSessionForm({ onSuccess }: Props) {
    const router = useRouter();

    // ── Form state ────────────────────────────────────────────────────
    const [name, setName] = useState("");
    const [evaluationMethod, setEvaluationMethod] = useState("QUANTITATIVE");
    const [maxPoints, setMaxPoints] = useState("");
    const [gradingScaleProfileId, setGradingScaleProfileId] = useState("");
    const [scheduledDate, setScheduledDate] = useState("");
    const [classId, setClassId] = useState("");
    const [learningAreaId, setLearningAreaId] = useState("");
    const [academicYearId, setAcademicYearId] = useState("");
    const [academicTermId, setAcademicTermId] = useState("");
    const [error, setError] = useState<string | null>(null);
    const [fieldErrors, setFieldErrors] = useState<Record<string, string[]>>({});

    // ── Fetch grading scale profiles for dropdown ─────────────────────
    const { data: profilesData } = useQuery({
        queryKey: ["scale-profiles", "list", true],
        queryFn: () => listScaleProfiles(true),
        staleTime: 5 * 60 * 1000,
    });

    // ── Mutation ──────────────────────────────────────────────────────
    const createMutation = useMutation({
        mutationFn: () => {
            const errors: Record<string, string[]> = {};
            if (!name.trim()) errors.name = ["Assessment name is required."];
            if (!classId) errors.class_id = ["Please select a class."];
            if (!learningAreaId) errors.learning_area_id = ["Please select a learning area."];
            if (!academicYearId) errors.academic_year_id = ["Please select an academic year."];
            if (!academicTermId) errors.academic_term_id = ["Please select an academic term."];

            if (Object.keys(errors).length > 0) {
                setFieldErrors(errors);
                throw new Error("Please fill in all required fields.");
            }

            return createSession({
                class_id: classId,
                learning_area_id: learningAreaId,
                academic_term_id: academicTermId,
                academic_year_id: academicYearId,
                name: name.trim(),
                evaluation_method: evaluationMethod as "QUANTITATIVE" | "RUBRIC",
                max_points:
                    evaluationMethod === "QUANTITATIVE" ? parseFloat(maxPoints) || null : null,
                grading_scale_profile_id:
                    evaluationMethod === "QUANTITATIVE" ? gradingScaleProfileId || null : null,
                scheduled_date: scheduledDate || null,
            });
        },
        onSuccess: (result) => {
            router.back();
            onSuccess?.(result as unknown as AssessmentSession);
        },
        onError: (err) => {
            if (isApiError(err) && err.status === 400 && err.errors) {
                setFieldErrors(err.errors);
                setError(null);
            } else {
                setFieldErrors({});
                setError(getErrorMessage(err));
            }
        },
    });

    const handleSubmit = useCallback(
        (e: React.FormEvent) => {
            e.preventDefault();
            setError(null);
            setFieldErrors({});
            createMutation.mutate();
        },
        [createMutation]
    );

    const profiles = profilesData?.items ?? [];

    return (
        <form onSubmit={handleSubmit} className="space-y-4">
            {error && (
                <Alert variant="destructive">
                    <AlertDescription>{error}</AlertDescription>
                </Alert>
            )}

            {/* Name */}
            <div className="space-y-1.5">
                <Label htmlFor="name">Assessment Name</Label>
                <Input
                    id="name"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    placeholder='e.g. "Mathematics CAT 1"'
                    disabled={createMutation.isPending}
                />
                {fieldErrors.name && (
                    <p className="text-destructive text-xs">{fieldErrors.name[0]}</p>
                )}
            </div>

            {/* Class */}
            <div className="space-y-1.5">
                <Label>Class</Label>
                <ClassCombobox
                    value={classId}
                    onChange={(v) => {
                        setClassId(v as string);
                        setFieldErrors({});
                    }}
                    placeholder="Select a class..."
                    onCreateItem={() => router.push("/classes/add")}
                />
                {fieldErrors.class_id && (
                    <p className="text-destructive text-xs">{fieldErrors.class_id[0]}</p>
                )}
            </div>

            {/* Learning Area */}
            <div className="space-y-1.5">
                <Label>Learning Area / Subject</Label>
                <LearningAreaCombobox
                    value={learningAreaId}
                    onChange={(v) => {
                        setLearningAreaId(v as string);
                        setFieldErrors({});
                    }}
                    placeholder="Select a learning area..."
                    onCreateItem={() => router.push("/curriculum/new")}
                />
                {fieldErrors.learning_area_id && (
                    <p className="text-destructive text-xs">{fieldErrors.learning_area_id[0]}</p>
                )}
            </div>

            {/* Academic Year */}
            <div className="space-y-1.5">
                <Label>Academic Year</Label>
                <AcademicYearCombobox
                    value={academicYearId}
                    onChange={(v) => {
                        setAcademicYearId(v);
                        setFieldErrors({});
                    }}
                    placeholder="Select an academic year..."
                    onCreateItem={() => router.push("/academic-years/new")}
                />
                {fieldErrors.academic_year_id && (
                    <p className="text-destructive text-xs">{fieldErrors.academic_year_id[0]}</p>
                )}
            </div>

            {/* Academic Term */}
            <div className="space-y-1.5">
                <Label>Academic Term</Label>
                <AcademicTermCombobox
                    value={academicTermId}
                    onChange={(v) => {
                        setAcademicTermId(v);
                        setFieldErrors({});
                    }}
                    placeholder="Select an academic term..."
                    onCreateItem={(_search) =>
                        router.push(
                            `/academic-terms/new?academic_year_id=${encodeURIComponent(academicYearId)}`
                        )
                    }
                />
                {fieldErrors.academic_term_id && (
                    <p className="text-destructive text-xs">{fieldErrors.academic_term_id[0]}</p>
                )}
            </div>

            {/* Evaluation Method */}
            <div className="space-y-1.5">
                <Label>Evaluation Method</Label>
                <Select
                    value={evaluationMethod}
                    onValueChange={(v) => {
                        setEvaluationMethod(v);
                        setFieldErrors({});
                    }}
                >
                    <SelectTrigger>
                        <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                        <SelectItem value="QUANTITATIVE">
                            Marks-Based — total points converted to CBC levels
                        </SelectItem>
                        <SelectItem value="RUBRIC">
                            Rubric — direct indicator-level grading
                        </SelectItem>
                    </SelectContent>
                </Select>
                {fieldErrors.evaluation_method && (
                    <p className="text-destructive text-xs">{fieldErrors.evaluation_method[0]}</p>
                )}
            </div>

            {/* QUANTITATIVE-specific fields */}
            {evaluationMethod === "QUANTITATIVE" && (
                <>
                    {/* Max Points */}
                    <div className="space-y-1.5">
                        <Label htmlFor="maxPoints">Maximum Points</Label>
                        <Input
                            id="maxPoints"
                            type="number"
                            min={1}
                            step={0.5}
                            value={maxPoints}
                            onChange={(e) => setMaxPoints(e.target.value)}
                            placeholder="e.g. 50"
                            disabled={createMutation.isPending}
                        />
                        {fieldErrors.max_points && (
                            <p className="text-destructive text-xs">{fieldErrors.max_points[0]}</p>
                        )}
                    </div>

                    {/* Grading Scale Profile */}
                    <div className="space-y-1.5">
                        <Label>Grading Scale</Label>
                        <Select
                            value={gradingScaleProfileId}
                            onValueChange={(v) => {
                                setGradingScaleProfileId(v);
                                setFieldErrors({});
                            }}
                        >
                            <SelectTrigger>
                                <SelectValue placeholder="Select a scale profile..." />
                            </SelectTrigger>
                            <SelectContent>
                                {profiles.map((p) => (
                                    <SelectItem key={p.id} value={p.id}>
                                        {p.name}
                                    </SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                        {fieldErrors.grading_scale_profile_id && (
                            <p className="text-destructive text-xs">
                                {fieldErrors.grading_scale_profile_id[0]}
                            </p>
                        )}
                    </div>
                </>
            )}

            {/* For RUBRIC, show a note */}
            {evaluationMethod === "RUBRIC" && (
                <Alert>
                    <AlertDescription>
                        Rubric assessments let you grade each KICD performance indicator directly
                        using EE, ME, AE, or BE. No numeric conversion needed.
                    </AlertDescription>
                </Alert>
            )}

            {/* Scheduled Date */}
            <div className="space-y-1.5">
                <Label>Scheduled Date (optional)</Label>
                <DatePicker
                    value={scheduledDate}
                    onChange={setScheduledDate}
                    placeholder="Pick a date"
                    disabled={createMutation.isPending}
                />
            </div>

            {/* Submit */}
            <div className="flex items-center justify-end gap-2 pt-2">
                <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => router.back()}
                    disabled={createMutation.isPending}
                >
                    Cancel
                </Button>
                <Button type="submit" size="sm" disabled={createMutation.isPending || !name.trim()}>
                    {createMutation.isPending ? (
                        <>
                            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                            Creating...
                        </>
                    ) : (
                        "Create Session"
                    )}
                </Button>
            </div>
        </form>
    );
}
