"use client";

import { useState } from "react";
import { Plus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { DataTable } from "@/components/shared/data-table";
import { type DataTableColumn } from "@/components/shared/data-table/types";
import { useFeeCategories } from "@/features/fee-categories";
import { useAcademicTerms } from "@/features/academic-terms";
import { listFeeTemplates, deleteFeeTemplate, type FeeTemplate } from "@/lib/api/billing";

function createColumns(
    data: { categories: { id: string; name: string }[]; terms: { id: string; name: string }[] },
    onEdit: (t: FeeTemplate) => void
): DataTableColumn<FeeTemplate>[] {
    const catMap = new Map(data.categories.map((c) => [c.id, c.name]));
    const termMap = new Map(data.terms.map((t) => [t.id, t.name]));

    return [
        {
            id: "category",
            header: "Category",
            cell: (row) => (
                <span className="font-medium">
                    {catMap.get(row.fee_category_id) ?? row.fee_category_id}
                </span>
            ),
        },
        {
            id: "grade_level",
            header: "Grade",
            width: "80px",
            cell: (row) => <span className="text-muted-foreground">{row.grade_level}</span>,
        },
        {
            id: "term",
            header: "Term",
            cell: (row) => (
                <span className="text-muted-foreground">
                    {termMap.get(row.academic_term_id) ?? row.academic_term_id}
                </span>
            ),
        },
        {
            id: "amount",
            header: "Amount",
            width: "120px",
            align: "right",
            cell: (row) => <span className="font-medium tabular-nums">{row.amount}</span>,
        },
        {
            id: "actions",
            header: "",
            width: "140px",
            align: "right",
            cell: (row) => (
                <div className="flex items-center justify-end gap-2">
                    <Button variant="outline" size="sm" onClick={() => onEdit(row)}>
                        Edit
                    </Button>
                    <DeleteCell template={row} />
                </div>
            ),
        },
    ];
}

import { CreateTemplateDialog } from "./create-template-dialog";
import { EditAmountDialog } from "./edit-amount-dialog";
import { DeleteCell } from "./delete-cell";

export function FeeTemplatesList() {
    const [createOpen, setCreateOpen] = useState(false);
    const [editTemplate, setEditTemplate] = useState<FeeTemplate | null>(null);

    const { data: catsData } = useFeeCategories();
    const { data: termsData } = useAcademicTerms();

    const categories = catsData?.items ?? [];
    const terms = termsData?.items ?? [];

    const columns = createColumns({ categories, terms }, (t) => setEditTemplate(t));

    return (
        <div className="space-y-4">
            <DataTable
                isCheckable
                queryKey={["fee-templates"]}
                queryFn={() => listFeeTemplates()}
                columns={columns}
                getRowId={(row) => row.id}
                deleteFn={(id) => deleteFeeTemplate(String(id))}
                emptyState="No fee templates yet. Create one to define fees for a grade and term."
                noResultsState="No fee templates match your search."
                renderToolBarComponents={() => (
                    <Button
                        key="add-template"
                        variant="outline"
                        size="sm"
                        onClick={() => setCreateOpen(true)}
                    >
                        <Plus className="mr-1 size-4" />
                        Add Fee Template
                    </Button>
                )}
            />

            <CreateTemplateDialog open={createOpen} onOpenChange={setCreateOpen} />

            {editTemplate && (
                <EditAmountDialog
                    key={editTemplate.id}
                    template={editTemplate}
                    open={!!editTemplate}
                    onOpenChange={(open) => {
                        if (!open) setEditTemplate(null);
                    }}
                />
            )}
        </div>
    );
}
