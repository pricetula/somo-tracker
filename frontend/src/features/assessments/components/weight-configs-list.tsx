/**
 * WeightConfigsList — Admin view listing all KNEC weight configurations.
 *
 * Uses the shared DataTable component with filters. Click "Add Config"
 * to open the create form (intercepted as a modal).
 */

"use client";

import { useState } from "react";
import { Plus, Loader2 } from "lucide-react";

import { DataTable } from "@/components/shared/data-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";
import type { FilterGroup } from "@/components/shared/data-table/types";
import { Badge } from "@/components/ui/badge";
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
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogHeader,
    DialogTitle,
    DialogTrigger,
} from "@/components/ui/dialog";
import { Alert, AlertDescription } from "@/components/ui/alert";

import type { AssessmentWeightConfig } from "@/lib/api/assessments";
import { listWeightConfigs } from "@/lib/api/assessments";
import { useCreateWeightConfig } from "../hooks/use-assessments";
import { getErrorMessage } from "@/lib/errors";

// ─── KNEX Target Exams ─────────────────────────────────────────────────

const TARGET_EXAMS = ["KPSEA", "KJSEA", "KSSEA"] as const;

const ASSESSMENT_TYPES = [
    "Formative_Classroom",
    "KNEC_Written_Assessment",
    "KNEC_SBA_Project",
    "National_KPSEA",
    "National_KJSEA",
    "National_KSSEA",
] as const;

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

// ─── Create Form ───────────────────────────────────────────────────────

function CreateWeightConfigForm({ onSuccess }: { onSuccess?: () => void }) {
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

// ─── Columns ──────────────────────────────────────────────────────────────

const columns: DataTableColumn<AssessmentWeightConfig>[] = [
    {
        id: "grade_level",
        header: "Grade",
        width: "80px",
        cell: (row) => <Badge variant="secondary">{row.grade_level}</Badge>,
    },
    {
        id: "assessment_type_code",
        header: "Assessment Type",
        cell: (row) => (
            <span className="text-xs font-medium">
                {row.assessment_type_code.replace(/_/g, " ")}
            </span>
        ),
    },
    {
        id: "target_exam",
        header: "Target Exam",
        width: "100px",
        cell: (row) => (
            <Badge variant="outline" className="text-xs">
                {row.target_exam}
            </Badge>
        ),
    },
    {
        id: "weight_percent",
        header: "Weight",
        width: "80px",
        align: "right",
        cell: (row) => <span className="font-semibold tabular-nums">{row.weight_percent}%</span>,
    },
    {
        id: "effective_from",
        header: "From",
        width: "80px",
        align: "right",
        cell: (row) => (
            <span className="text-muted-foreground tabular-nums">{row.effective_from}</span>
        ),
    },
    {
        id: "notes",
        header: "Notes",
        cell: (row) =>
            row.notes ? (
                <span className="text-muted-foreground line-clamp-1 text-xs">{row.notes}</span>
            ) : (
                <span className="text-muted-foreground">\u2014</span>
            ),
    },
];

// ─── Filter Groups ────────────────────────────────────────────────────────

const filterGroups: FilterGroup[] = [
    {
        id: "weight_filters",
        label: "Filter by",
        items: [
            {
                id: "grade_level",
                label: "Grade",
                type: "sub_menu_multi",
                submenu: GRADE_LEVELS.map((gl) => ({
                    id: gl,
                    label: gl,
                    value: gl,
                })),
            },
            {
                id: "target_exam",
                label: "Target Exam",
                type: "sub_menu_multi",
                submenu: TARGET_EXAMS.map((te) => ({
                    id: te,
                    label: te,
                    value: te,
                })),
            },
        ],
    },
];

// ─── Component ────────────────────────────────────────────────────────────

export function WeightConfigsList() {
    const [createOpen, setCreateOpen] = useState(false);

    return (
        <div className="space-y-4">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-foreground text-2xl font-semibold">
                        Weight Configurations
                    </h1>
                    <p className="text-muted-foreground mt-1">
                        KNEC national weighting formulas — defines how assessment types contribute
                        to target exam placement scores.
                    </p>
                </div>
                <Dialog open={createOpen} onOpenChange={setCreateOpen}>
                    <DialogTrigger asChild>
                        <Button size="sm">
                            <Plus className="mr-1.5 h-4 w-4" />
                            Add Config
                        </Button>
                    </DialogTrigger>
                    <DialogContent className="sm:max-w-md">
                        <DialogHeader>
                            <DialogTitle>Create Weight Configuration</DialogTitle>
                            <DialogDescription>
                                Define a new KNEC weighting rule for a grade level and assessment
                                type.
                            </DialogDescription>
                        </DialogHeader>
                        <CreateWeightConfigForm onSuccess={() => setCreateOpen(false)} />
                    </DialogContent>
                </Dialog>
            </div>

            <DataTable
                queryKey={["weight-configs"]}
                queryFn={() => listWeightConfigs()}
                columns={columns}
                getRowId={(row) => row.id}
                filterGroups={filterGroups}
                emptyState="No weight configurations yet. Add one to define KNEC weighting formulas."
                noResultsState="No configurations match your filters."
            />
        </div>
    );
}
