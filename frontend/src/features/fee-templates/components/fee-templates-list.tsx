/**
 * FeeTemplatesList — manage fee templates with inline CRUD.
 *
 * A fee template links a fee category to a grade level + term with a fixed amount.
 */

"use client";

import { useState } from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table";
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
    AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { getErrorMessage } from "@/lib/errors";
import { useFeeCategories } from "@/features/fee-categories";
import { useAcademicTerms } from "@/features/academic-terms";
import {
    useFeeTemplates,
    useCreateFeeTemplate,
    useUpdateFeeTemplate,
    useDeleteFeeTemplate,
} from "../hooks/use-fee-templates";
import type { FeeTemplate } from "../types";

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

// ─── Component ────────────────────────────────────────────────────────────

export function FeeTemplatesList() {
    const { data: catsData } = useFeeCategories();
    const { data: termsData } = useAcademicTerms();
    const { data, isLoading, isError, error } = useFeeTemplates();
    const createMutation = useCreateFeeTemplate();
    const updateMutation = useUpdateFeeTemplate();
    const deleteMutation = useDeleteFeeTemplate();

    const categories = catsData?.items ?? [];
    const terms = termsData?.items ?? [];

    // New template form
    const [showForm, setShowForm] = useState(false);
    const [newCategoryId, setNewCategoryId] = useState("");
    const [newGrade, setNewGrade] = useState("");
    const [newTermId, setNewTermId] = useState("");
    const [newAmount, setNewAmount] = useState("");

    // Edit state
    const [editingId, setEditingId] = useState<string | null>(null);
    const [editAmount, setEditAmount] = useState("");

    // ── Loading ──────────────────────────────────────────────────────────
    if (isLoading) {
        return (
            <div className="space-y-3">
                <Skeleton className="h-8 w-48" />
                <Skeleton className="h-10 w-full" />
            </div>
        );
    }

    // ── Error ────────────────────────────────────────────────────────────
    if (isError) {
        return (
            <Alert variant="destructive">
                <AlertDescription>{getErrorMessage(error)}</AlertDescription>
            </Alert>
        );
    }

    const templates = data?.items ?? [];

    async function handleCreate() {
        if (!newCategoryId || !newGrade || !newTermId || !newAmount) return;
        createMutation.mutate(
            {
                fee_category_id: newCategoryId,
                grade_level: newGrade,
                academic_term_id: newTermId,
                amount: newAmount,
            },
            {
                onSuccess: () => {
                    setShowForm(false);
                    setNewCategoryId("");
                    setNewGrade("");
                    setNewTermId("");
                    setNewAmount("");
                },
            }
        );
    }

    function startEdit(t: FeeTemplate) {
        setEditingId(t.id);
        setEditAmount(t.amount);
    }

    function cancelEdit() {
        setEditingId(null);
    }

    function handleUpdate(id: string) {
        if (!editAmount) return;
        updateMutation.mutate(
            { id, payload: { amount: editAmount } },
            { onSuccess: () => setEditingId(null) }
        );
    }

    // Helper to resolve names
    function categoryName(id: string) {
        return categories.find((c) => c.id === id)?.name ?? id;
    }
    function termName(id: string) {
        return terms.find((t) => t.id === id)?.name ?? id;
    }

    return (
        <div className="space-y-6">
            {/* ── Create Button / Form ────────────────────────────────────── */}
            {showForm ? (
                <div className="space-y-4 rounded-md border p-4">
                    <h3 className="text-foreground font-medium">New Fee Template</h3>
                    <div className="grid grid-cols-2 gap-4">
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
                    <div className="flex gap-2">
                        <Button
                            onClick={handleCreate}
                            disabled={
                                !newCategoryId ||
                                !newGrade ||
                                !newTermId ||
                                !newAmount ||
                                createMutation.isPending
                            }
                        >
                            Create Template
                        </Button>
                        <Button variant="outline" onClick={() => setShowForm(false)}>
                            Cancel
                        </Button>
                    </div>
                </div>
            ) : (
                <Button size="sm" onClick={() => setShowForm(true)}>
                    Add Fee Template
                </Button>
            )}

            {/* ── List ────────────────────────────────────────────────────── */}
            {templates.length === 0 && !showForm ? (
                <p className="text-muted-foreground">
                    No fee templates yet. Create one to define fees for a grade and term.
                </p>
            ) : (
                <Table>
                    <TableHeader>
                        <TableRow>
                            <TableHead>Category</TableHead>
                            <TableHead>Grade</TableHead>
                            <TableHead>Term</TableHead>
                            <TableHead>Amount</TableHead>
                            <TableHead className="text-right">Actions</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {templates.map((t) => (
                            <TableRow key={t.id}>
                                <TableCell className="font-medium">
                                    {categoryName(t.fee_category_id)}
                                </TableCell>
                                <TableCell>{t.grade_level}</TableCell>
                                <TableCell>{termName(t.academic_term_id)}</TableCell>
                                <TableCell>
                                    {editingId === t.id ? (
                                        <Input
                                            type="number"
                                            step="0.01"
                                            min="0"
                                            value={editAmount}
                                            onChange={(e) => setEditAmount(e.target.value)}
                                            className="w-32"
                                            onKeyDown={(e) =>
                                                e.key === "Enter" && handleUpdate(t.id)
                                            }
                                        />
                                    ) : (
                                        <span>{t.amount}</span>
                                    )}
                                </TableCell>
                                <TableCell className="text-right">
                                    <div className="flex items-center justify-end gap-2">
                                        {editingId === t.id ? (
                                            <>
                                                <Button
                                                    variant="outline"
                                                    size="sm"
                                                    onClick={() => handleUpdate(t.id)}
                                                    disabled={updateMutation.isPending}
                                                >
                                                    Save
                                                </Button>
                                                <Button
                                                    variant="outline"
                                                    size="sm"
                                                    onClick={cancelEdit}
                                                >
                                                    Cancel
                                                </Button>
                                            </>
                                        ) : (
                                            <>
                                                <Button
                                                    variant="outline"
                                                    size="sm"
                                                    onClick={() => startEdit(t)}
                                                >
                                                    Edit
                                                </Button>
                                                <AlertDialog>
                                                    <AlertDialogTrigger asChild>
                                                        <Button
                                                            variant="outline"
                                                            size="sm"
                                                            className="text-destructive"
                                                        >
                                                            Delete
                                                        </Button>
                                                    </AlertDialogTrigger>
                                                    <AlertDialogContent>
                                                        <AlertDialogHeader>
                                                            <AlertDialogTitle>
                                                                Delete Fee Template
                                                            </AlertDialogTitle>
                                                            <AlertDialogDescription>
                                                                Are you sure you want to delete this
                                                                fee template for{" "}
                                                                {categoryName(t.fee_category_id)}{" "}
                                                                &mdash; {t.grade_level}?
                                                            </AlertDialogDescription>
                                                        </AlertDialogHeader>
                                                        <AlertDialogFooter>
                                                            <AlertDialogCancel>
                                                                Cancel
                                                            </AlertDialogCancel>
                                                            <AlertDialogAction
                                                                onClick={() =>
                                                                    deleteMutation.mutate(t.id)
                                                                }
                                                            >
                                                                Delete
                                                            </AlertDialogAction>
                                                        </AlertDialogFooter>
                                                    </AlertDialogContent>
                                                </AlertDialog>
                                            </>
                                        )}
                                    </div>
                                </TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
            )}
        </div>
    );
}
