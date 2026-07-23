/**
 * FeeCategoriesList — manage fee categories with DataTable + create/edit dialogs.
 *
 * Uses the shared DataTable component for listing with per-row edit/delete actions.
 * Create and edit are handled via dialogs.
 */

"use client";

import { useState, useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Plus } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import { DataTable } from "@/components/shared/data-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";
import { RowActions } from "@/components/shared/data-table/row-actions";
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog";
import { getErrorMessage } from "@/lib/errors";
import {
    listFeeCategories,
    createFeeCategory,
    updateFeeCategory,
    deleteFeeCategory,
    type FeeCategory,
} from "@/lib/api/billing";

// ─── Create Dialog ────────────────────────────────────────────────────────

function CreateCategoryDialog({
    open,
    onOpenChange,
}: {
    open: boolean;
    onOpenChange: (open: boolean) => void;
}) {
    const queryClient = useQueryClient();
    const [newName, setNewName] = useState("");
    const [newMandatory, setNewMandatory] = useState(false);
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const handleCreate = useCallback(async () => {
        if (!newName.trim()) return;
        setSaving(true);
        setError(null);
        try {
            await createFeeCategory({ name: newName.trim(), is_mandatory: newMandatory });
            await queryClient.invalidateQueries({ queryKey: ["fee-categories"] });
            toast.success("Fee category created.");
            setNewName("");
            setNewMandatory(false);
            onOpenChange(false);
        } catch (err) {
            setError(getErrorMessage(err));
        } finally {
            setSaving(false);
        }
    }, [newName, newMandatory, queryClient, onOpenChange]);

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="sm:max-w-md">
                <DialogHeader>
                    <DialogTitle>Create Fee Category</DialogTitle>
                    <DialogDescription>
                        Add a new fee category (e.g. Tuition, Transport).
                    </DialogDescription>
                </DialogHeader>
                <div className="space-y-4">
                    {error && (
                        <div className="text-destructive bg-destructive/10 rounded-md px-3 py-2">
                            {error}
                        </div>
                    )}
                    <div className="space-y-1.5">
                        <Label htmlFor="cat-name">Category Name</Label>
                        <Input
                            id="cat-name"
                            value={newName}
                            onChange={(e) => setNewName(e.target.value)}
                            placeholder="e.g. Tuition"
                            onKeyDown={(e) => e.key === "Enter" && handleCreate()}
                        />
                    </div>
                    <div className="flex items-center gap-2">
                        <Checkbox
                            id="cat-mandatory"
                            checked={newMandatory}
                            onCheckedChange={(v) => setNewMandatory(v === true)}
                        />
                        <Label htmlFor="cat-mandatory">Mandatory</Label>
                    </div>
                </div>
                <DialogFooter>
                    <Button variant="outline" onClick={() => onOpenChange(false)} disabled={saving}>
                        Cancel
                    </Button>
                    <Button onClick={handleCreate} disabled={!newName.trim() || saving}>
                        {saving ? "Creating..." : "Create"}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}

// ─── Edit Dialog ──────────────────────────────────────────────────────────

function EditCategoryDialog({
    category,
    open,
    onOpenChange,
}: {
    category: FeeCategory;
    open: boolean;
    onOpenChange: (open: boolean) => void;
}) {
    const queryClient = useQueryClient();
    const [editName, setEditName] = useState(category.name);
    const [editMandatory, setEditMandatory] = useState(category.is_mandatory);
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const handleSave = useCallback(async () => {
        if (!editName.trim()) return;
        setSaving(true);
        setError(null);
        try {
            await updateFeeCategory(category.id, {
                name: editName.trim(),
                is_mandatory: editMandatory,
            });
            await queryClient.invalidateQueries({ queryKey: ["fee-categories"] });
            toast.success("Fee category updated.");
            onOpenChange(false);
        } catch (err) {
            setError(getErrorMessage(err));
        } finally {
            setSaving(false);
        }
    }, [category.id, editName, editMandatory, queryClient, onOpenChange]);

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="sm:max-w-md">
                <DialogHeader>
                    <DialogTitle>Edit Fee Category</DialogTitle>
                    <DialogDescription>Update the fee category details.</DialogDescription>
                </DialogHeader>
                <div className="space-y-4">
                    {error && (
                        <div className="text-destructive bg-destructive/10 rounded-md px-3 py-2">
                            {error}
                        </div>
                    )}
                    <div className="space-y-1.5">
                        <Label htmlFor="edit-cat-name">Category Name</Label>
                        <Input
                            id="edit-cat-name"
                            value={editName}
                            onChange={(e) => setEditName(e.target.value)}
                            onKeyDown={(e) => e.key === "Enter" && handleSave()}
                        />
                    </div>
                    <div className="flex items-center gap-2">
                        <Checkbox
                            id="edit-cat-mandatory"
                            checked={editMandatory}
                            onCheckedChange={(v) => setEditMandatory(v === true)}
                        />
                        <Label htmlFor="edit-cat-mandatory">Mandatory</Label>
                    </div>
                </div>
                <DialogFooter>
                    <Button variant="outline" onClick={() => onOpenChange(false)} disabled={saving}>
                        Cancel
                    </Button>
                    <Button onClick={handleSave} disabled={!editName.trim() || saving}>
                        {saving ? "Saving..." : "Save"}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}

// ─── Delete cell ──────────────────────────────────────────────────────────

function DeleteCell({ category }: { category: FeeCategory }) {
    const queryClient = useQueryClient();

    const handleDelete = useCallback(async () => {
        try {
            await deleteFeeCategory(category.id);
            await queryClient.invalidateQueries({ queryKey: ["fee-categories"] });
            toast.success("Fee category deleted.");
        } catch (err) {
            toast.error(getErrorMessage(err));
        }
    }, [category.id, queryClient]);

    return <RowActions rowId={category.id} label={category.name} onDelete={handleDelete} />;
}

// ─── Columns ──────────────────────────────────────────────────────────────

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

// ─── Component ────────────────────────────────────────────────────────────

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
