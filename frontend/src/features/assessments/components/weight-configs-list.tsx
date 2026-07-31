"use client";

import { useState } from "react";
import { Plus } from "lucide-react";
import { DataTable } from "@/components/shared/data-table";
import { type DataTableColumn } from "@/components/shared/data-table/types";
import { type FilterGroup } from "@/components/shared/data-table/types";
import { RowActions } from "@/components/shared/data-table/row-actions";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogHeader,
    DialogTitle,
    DialogTrigger,
} from "@/components/ui/dialog";
import { type AssessmentWeightConfig } from "@/lib/api/assessments";
import { listWeightConfigs } from "@/lib/api/assessments";
import { useDeleteWeightConfig } from "../hooks/use-assessments";

const TARGET_EXAMS = ["KPSEA", "KJSEA", "KSSEA"] as const;
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
function createColumns(
    deleteMutation: ReturnType<typeof useDeleteWeightConfig>
): DataTableColumn<AssessmentWeightConfig>[] {
    return [
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
            cell: (row) => (
                <span className="font-semibold tabular-nums">{row.weight_percent}%</span>
            ),
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
                    <span className="text-muted-foreground">-</span>
                ),
        },
        {
            id: "actions",
            header: "",
            width: "48px",
            align: "right",
            cell: (row) => (
                <RowActions
                    rowId={row.id}
                    label={`${row.grade_level} ${row.assessment_type_code}`}
                    onDelete={() => deleteMutation.mutate(row.id)}
                    disabled={deleteMutation.isPending}
                />
            ),
        },
    ];
}
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

import { CreateWeightConfigForm } from "./create-weight-config-form";

export function WeightConfigsList() {
    const [createOpen, setCreateOpen] = useState(false);
    const deleteMutation = useDeleteWeightConfig();
    const columns = createColumns(deleteMutation);

    // ─── Delete mutation wrapper for DataTable bulk delete ─────────────
    const bulkDeleteFn = (id: string | number) => deleteMutation.mutateAsync(String(id));

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
                isCheckable
                queryKey={["weight-configs"]}
                queryFn={() => listWeightConfigs()}
                columns={columns}
                getRowId={(row) => row.id}
                filterGroups={filterGroups}
                deleteFn={bulkDeleteFn}
                emptyState="No weight configurations yet. Add one to define KNEC weighting formulas."
                noResultsState="No configurations match your filters."
            />
        </div>
    );
}
