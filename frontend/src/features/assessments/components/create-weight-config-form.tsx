"use client";

import { useState } from "react";
import { Loader2 } from "lucide-react";
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
import { useCreateWeightConfig } from "../hooks/use-assessments";
import { getErrorMessage } from "@/lib/errors";

const GRADE_LEVELS = [
    "PP1",
    "PP2",
    "G1",
    "G2",
    "G3",
    "G4",
    "G5",
    "G6",
    "G7",
    "G8",
    "G9",
    "G10",
    "G11",
    "G12",
] as const;
const TARGET_EXAMS = ["KPSEA", "KJSEA", "KSSEA"] as const;
const ASSESSMENT_TYPES = [
    "Formative_Classroom",
    "KNEC_Written_Assessment",
    "KNEC_SBA_Project",
    "National_KPSEA",
    "National_KJSEA",
    "National_KSSEA",
] as const;

export function CreateWeightConfigForm({ onSuccess }: { onSuccess?: () => void }) {
    const [gradeLevel, setGradeLevel] = useState("");
    const [assessmentTypeCode, setAssessmentTypeCode] = useState("");
    const [targetExam, setTargetExam] = useState("");
    const [weightPercent, setWeightPercent] = useState("");
    const [effectiveFrom, setEffectiveFrom] = useState("");
    const [notes, setNotes] = useState("");
    const [error, setError] = useState<string | null>(null);

    const createMutation = useCreateWeightConfig();

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault();
        setError(null);

        if (!gradeLevel) {
            setError("Grade level is required.");
            return;
        }
        if (!assessmentTypeCode) {
            setError("Assessment type is required.");
            return;
        }
        if (!targetExam) {
            setError("Target exam is required.");
            return;
        }
        const wp = parseFloat(weightPercent);
        if (isNaN(wp) || wp <= 0 || wp > 100) {
            setError("Weight must be between 0 and 100.");
            return;
        }
        const ef = parseInt(effectiveFrom, 10);
        if (isNaN(ef) || ef < 2024 || ef > 2100) {
            setError("Year must be between 2024 and 2100.");
            return;
        }

        createMutation.mutate(
            {
                grade_level: gradeLevel,
                assessment_type_code: assessmentTypeCode,
                target_exam: targetExam,
                weight_percent: wp,
                effective_from: ef,
                notes: notes.trim() || null,
            },
            {
                onSuccess: () => onSuccess?.(),
                onError: (err) => setError(getErrorMessage(err)),
            }
        );
    };

    return (
        <form onSubmit={handleSubmit} className="space-y-4">
            {error && (
                <Alert variant="destructive">
                    <AlertDescription>{error}</AlertDescription>
                </Alert>
            )}

            <div className="space-y-1.5">
                <Label>Grade Level</Label>
                <Select value={gradeLevel} onValueChange={setGradeLevel}>
                    <SelectTrigger>
                        <SelectValue placeholder="Select grade level..." />
                    </SelectTrigger>
                    <SelectContent>
                        {GRADE_LEVELS.map((gl) => (
                            <SelectItem key={gl} value={gl}>
                                {gl}
                            </SelectItem>
                        ))}
                    </SelectContent>
                </Select>
            </div>

            <div className="space-y-1.5">
                <Label>Assessment Type</Label>
                <Select value={assessmentTypeCode} onValueChange={setAssessmentTypeCode}>
                    <SelectTrigger>
                        <SelectValue placeholder="Select assessment type..." />
                    </SelectTrigger>
                    <SelectContent>
                        {ASSESSMENT_TYPES.map((at) => (
                            <SelectItem key={at} value={at}>
                                {at.replace(/_/g, " ")}
                            </SelectItem>
                        ))}
                    </SelectContent>
                </Select>
            </div>

            <div className="space-y-1.5">
                <Label>Target Exam</Label>
                <Select value={targetExam} onValueChange={setTargetExam}>
                    <SelectTrigger>
                        <SelectValue placeholder="Select target exam..." />
                    </SelectTrigger>
                    <SelectContent>
                        {TARGET_EXAMS.map((te) => (
                            <SelectItem key={te} value={te}>
                                {te}
                            </SelectItem>
                        ))}
                    </SelectContent>
                </Select>
            </div>

            <div className="grid grid-cols-2 gap-4">
                <div className="space-y-1.5">
                    <Label>Weight (%)</Label>
                    <Input
                        type="number"
                        min={0.1}
                        max={100}
                        step={0.1}
                        value={weightPercent}
                        onChange={(e) => setWeightPercent(e.target.value)}
                        placeholder="e.g. 20"
                    />
                </div>
                <div className="space-y-1.5">
                    <Label>Effective From</Label>
                    <Input
                        type="number"
                        min={2024}
                        max={2100}
                        step={1}
                        value={effectiveFrom}
                        onChange={(e) => setEffectiveFrom(e.target.value)}
                        placeholder="e.g. 2026"
                    />
                </div>
            </div>

            <div className="space-y-1.5">
                <Label>Notes (optional)</Label>
                <Input
                    value={notes}
                    onChange={(e) => setNotes(e.target.value)}
                    placeholder="e.g. Grade 4 project component contributes 20% to KPSEA"
                />
            </div>

            <div className="flex items-center justify-end gap-2 pt-2">
                <Button type="submit" size="sm" disabled={createMutation.isPending}>
                    {createMutation.isPending ? (
                        <>
                            <Loader2 className="mr-1.5 h-4 w-4 animate-spin" />
                            Creating...
                        </>
                    ) : (
                        "Create Config"
                    )}
                </Button>
            </div>
        </form>
    );
}
