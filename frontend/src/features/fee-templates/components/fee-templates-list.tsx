/**
 * FeeTemplatesList — manage fee templates with DataTable + create/edit dialogs.
 *
 * A fee template links a fee category to a grade level + term with a fixed amount.
 * Uses the shared DataTable component for listing with per-row edit/delete actions.
 */

"use client";

import { useState, useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Plus } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
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
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { getErrorMessage } from "@/lib/errors";
import { useFeeCategories } from "@/features/fee-categories";
import { useAcademicTerms } from "@/features/academic-terms";
import {
    listFeeTemplates,
    createFeeTemplate,
    updateFeeTemplate,
    deleteFeeTemplate,
    type FeeTemplate,
} from "@/lib/api/billing";

// ─── Grade level options ──────────────────────────────────────────────────

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
];

// ─── Create Dialog ────────────────────────────────────────────────────────

function CreateTemplateDialog({
    open,
    onOpenChange,
}: {
    open: boolean;
    onOpenChange: (open: boolean) => void;
}) {
    const queryClient = useQueryClient();
    const { data: catsData } = useFeeCategories();
    const { data: termsData } = useAcademicTerms();

    const categories = catsData?.items ?? [];
    const terms = termsData?.items ?? [];

    const [newCategoryId, setNewCategoryId] = useState("");
    const [newGrade, setNewGrade] = useState("");
    const [newTermId, setNewTermId] = useState("");
    const [newAmount, setNewAmount] = useState("");
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const handleCreate = useCallback(async () => {
        if (!newCategoryId || !newGrade || !newTermId || !newAmount) return;
        setSaving(true);
        setError(null);
        try {
            await createFeeTemplate({
                fee_category_id: newCategoryId,
                grade_level: newGrade,
                academic_term_id: newTermId,
                amount: newAmount,
            });
            await queryClient.invalidateQueries({ queryKey: ["fee-templates"] });
            toast.success("Fee template created.");
            setNewCategoryId("");
            setNewGrade("");
            setNewTermId("");
            setNewAmount("");
            onOpenChange(false);
        } catch (err) {
            setError(getErrorMessage(err));
        } finally {
            setSaving(false);
        }
    }, [newCategoryId, newGrade, newTermId, newAmount, queryClient, onOpenChange]);

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="sm:max-w-md">
                <DialogHeader>
                    <DialogTitle>Create Fee Template</DialogTitle>
                    <DialogDescription>
                        Link a fee category to a grade level and term with a fixed amount.
                    </DialogDescription>
                </DialogHeader>
                <div className="space-y-4">
                    {error && (
                        <div className="text-destructive bg-destructive/10 rounded-md px-3 py-2">
                            {error}
                        </div>
                    )}
                    <div className="space-y-1.5">
                        <Label>Fee Category</Label>
                        <Select value={newCategoryId} onValueChange={setNewCategoryId}>
                            <SelectTrigger>
                                <SelectValue placeholder="Select category" />
                            </SelectTrigger>
                            <SelectContent>
                                {categories.map((c) => (
                                    <SelectItem key={c.id} value={c.id}>
                                        {c.name}
                                    </SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                    </div>
                    <div className="space-y-1.5">
                        <Label>Grade Level</Label>
                        <Select value={newGrade} onValueChange={setNewGrade}>
                            <SelectTrigger>
                                <SelectValue placeholder="Select grade" />
                            </SelectTrigger>
                            <SelectContent>
                                {GRADE_LEVELS.map((g) => (
                                    <SelectItem key={g} value={g}>
                                        {g}
                                    </SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                    </div>
                    <div className="space-y-1.5">
                        <Label>Academic Term</Label>
                        <Select value={newTermId} onValueChange={setNewTermId}>
                            <SelectTrigger>
                                <SelectValue placeholder="Select term" />
                            </SelectTrigger>
                            <SelectContent>
                                {terms.map((t) => (
                                    <SelectItem key={t.id} value={t.id}>
                                        {t.name}
                                    </SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                    </div>
                    <div className="space-y-1.5">
                        <Label>Amount</Label>
                        <Input
                            type="number"
                            step="0.01"
                            min="0"
                            value={newAmount}
                            onChange={(e) => setNewAmount(e.target.value)}
                            placeholder="0.00"
                        />
                    </div>
                </div>
                <DialogFooter>
                    <Button variant="outline" onClick={() => onOpenChange(false)} disabled={saving}>
                        Cancel
                    </Button>
                    <Button
                        onClick={handleCreate}
                        disabled={!newCategoryId || !newGrade || !newTermId || !newAmount || saving}
                    >
                        {saving ? "Creating..." : "Create"}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}

// ─── Edit Amount Dialog ───────────────────────────────────────────────────

function EditAmountDialog({
    template,
    open,
    onOpenChange,
}: {
    template: FeeTemplate;
    open: boolean;
    onOpenChange: (open: boolean) => void;
}) {
    const queryClient = useQueryClient();
    const [editAmount, setEditAmount] = useState(template.amount);
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const handleSave = useCallback(async () => {
        if (!editAmount) return;
        setSaving(true);
        setError(null);
        try {
            await updateFeeTemplate(template.id, { amount: editAmount });
            await queryClient.invalidateQueries({ queryKey: ["fee-templates"] });
            toast.success("Fee template updated.");
            onOpenChange(false);
        } catch (err) {
            setError(getErrorMessage(err));
        } finally {
            setSaving(false);
        }
    }, [template.id, editAmount, queryClient, onOpenChange]);

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="sm:max-w-sm">
                <DialogHeader>
                    <DialogTitle>Edit Amount</DialogTitle>
                    <DialogDescription>Update the fee amount for this template.</DialogDescription>
                </DialogHeader>
                <div className="space-y-4">
                    {error && (
                        <div className="text-destructive bg-destructive/10 rounded-md px-3 py-2">
                            {error}
                        </div>
                    )}
                    <div className="space-y-1.5">
                        <Label>Amount</Label>
                        <Input
                            type="number"
                            step="0.01"
                            min="0"
                            value={editAmount}
                            onChange={(e) => setEditAmount(e.target.value)}
                        />
                    </div>
                </div>
                <DialogFooter>
                    <Button variant="outline" onClick={() => onOpenChange(false)} disabled={saving}>
                        Cancel
                    </Button>
                    <Button onClick={handleSave} disabled={!editAmount || saving}>
                        {saving ? "Saving..." : "Save"}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}

// ─── Delete Cell ──────────────────────────────────────────────────────────

function DeleteCell({ template }: { template: FeeTemplate }) {
    const queryClient = useQueryClient();

    const handleDelete = useCallback(async () => {
        try {
            await deleteFeeTemplate(template.id);
            await queryClient.invalidateQueries({ queryKey: ["fee-templates"] });
            toast.success("Fee template deleted.");
        } catch (err) {
            toast.error(getErrorMessage(err));
        }
    }, [template.id, queryClient]);

    return (
        <RowActions
            rowId={template.id}
            label={`${template.grade_level} template`}
            onDelete={handleDelete}
        />
    );
}

// ─── Columns ──────────────────────────────────────────────────────────────

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

// ─── Component ────────────────────────────────────────────────────────────

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
