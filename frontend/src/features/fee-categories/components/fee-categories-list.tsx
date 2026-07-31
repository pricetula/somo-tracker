"use client";

import { useState } from "react";
import { Plus } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { DataTable } from "@/components/shared/data-table";
import { type DataTableColumn } from "@/components/shared/data-table/types";
import { listFeeCategories, deleteFeeCategory, type FeeCategory } from "@/lib/api/billing";

function createColumns(onEdit: (cat: FeeCategory) => void): DataTableColumn<FeeCategory>[] {
    return [
        {
            id: "name",
            header: "Name",
            cell: (row) => <span className="font-medium">{row.name}</span>,
        },
        {
            id: "is_mandatory",
            header: "Mandatory",
            width: "120px",
            cell: (row) => (
                <Badge variant={row.is_mandatory ? "default" : "secondary"}>
                    {row.is_mandatory ? "Mandatory" : "Optional"}
                </Badge>
            ),
        },
        {
            id: "actions",
            header: "",
            width: "48px",
            align: "right",
            cell: (row) => (
                <div className="flex items-center justify-end gap-2">
                    <Button variant="outline" size="sm" onClick={() => onEdit(row)}>
                        Edit
                    </Button>
                    <DeleteCell category={row} />
                </div>
            ),
        },
    ];
}

import { CreateCategoryDialog } from "./create-category-dialog";
import { EditCategoryDialog } from "./edit-category-dialog";
import { DeleteCell } from "./delete-cell";

export function FeeCategoriesList() {
    const [createOpen, setCreateOpen] = useState(false);
    const [editCategory, setEditCategory] = useState<FeeCategory | null>(null);

    const columns = createColumns((cat) => setEditCategory(cat));

    return (
        <div className="space-y-4">
            <DataTable
                isCheckable
                addHref={undefined}
                queryKey={["fee-categories"]}
                queryFn={() => listFeeCategories()}
                columns={columns}
                getRowId={(row) => row.id}
                deleteFn={(id) => deleteFeeCategory(String(id))}
                emptyState="No fee categories yet. Create one to get started."
                noResultsState="No fee categories match your search."
                renderToolBarComponents={() => (
                    <Button
                        key="add-category"
                        variant="outline"
                        size="sm"
                        onClick={() => setCreateOpen(true)}
                    >
                        <Plus className="mr-1 size-4" />
                        Add Category
                    </Button>
                )}
            />

            <CreateCategoryDialog open={createOpen} onOpenChange={setCreateOpen} />

            {editCategory && (
                <EditCategoryDialog
                    key={editCategory.id}
                    category={editCategory}
                    open={!!editCategory}
                    onOpenChange={(open) => {
                        if (!open) setEditCategory(null);
                    }}
                />
            )}
        </div>
    );
}
