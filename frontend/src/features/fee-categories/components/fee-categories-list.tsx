/**
 * FeeCategoriesList — manage fee categories with inline CRUD.
 */

"use client";

import { useState } from "react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
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
import { Checkbox } from "@/components/ui/checkbox";
import { getErrorMessage } from "@/lib/errors";
import {
    useFeeCategories,
    useCreateFeeCategory,
    useUpdateFeeCategory,
    useDeleteFeeCategory,
} from "../hooks/use-fee-categories";

export function FeeCategoriesList() {
    const { data, isLoading, isError, error } = useFeeCategories();
    const createMutation = useCreateFeeCategory();
    const updateMutation = useUpdateFeeCategory();
    const deleteMutation = useDeleteFeeCategory();

    // New category form
    const [newName, setNewName] = useState("");
    const [newMandatory, setNewMandatory] = useState(false);

    // Edit state
    const [editingId, setEditingId] = useState<string | null>(null);
    const [editName, setEditName] = useState("");
    const [editMandatory, setEditMandatory] = useState(false);

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

    const categories = data?.items ?? [];

    async function handleCreate() {
        if (!newName.trim()) return;
        createMutation.mutate(
            { name: newName.trim(), is_mandatory: newMandatory },
            {
                onSuccess: () => {
                    setNewName("");
                    setNewMandatory(false);
                },
            }
        );
    }

    function startEdit(cat: (typeof categories)[number]) {
        setEditingId(cat.id);
        setEditName(cat.name);
        setEditMandatory(cat.is_mandatory);
    }

    function cancelEdit() {
        setEditingId(null);
    }

    async function handleUpdate(id: string) {
        if (!editName.trim()) return;
        updateMutation.mutate(
            { id, payload: { name: editName.trim(), is_mandatory: editMandatory } },
            { onSuccess: () => setEditingId(null) }
        );
    }

    return (
        <div className="space-y-6">
            {/* ── Create form ─────────────────────────────────────────────── */}
            <div className="flex items-end gap-3">
                <div className="space-y-1.5">
                    <Label htmlFor="new-name">New Category Name</Label>
                    <Input
                        id="new-name"
                        value={newName}
                        onChange={(e) => setNewName(e.target.value)}
                        placeholder="e.g. Tuition"
                        className="w-64"
                        onKeyDown={(e) => e.key === "Enter" && handleCreate()}
                    />
                </div>
                <div className="flex items-center gap-2 pb-1.5">
                    <Checkbox
                        id="new-mandatory"
                        checked={newMandatory}
                        onCheckedChange={(v) => setNewMandatory(v === true)}
                    />
                    <Label htmlFor="new-mandatory" className="text-sm">
                        Mandatory
                    </Label>
                </div>
                <Button
                    size="sm"
                    onClick={handleCreate}
                    disabled={!newName.trim() || createMutation.isPending}
                >
                    Add Category
                </Button>
            </div>

            {/* ── List ────────────────────────────────────────────────────── */}
            {categories.length === 0 ? (
                <p className="text-muted-foreground text-sm">
                    No fee categories yet. Create one above.
                </p>
            ) : (
                <Table>
                    <TableHeader>
                        <TableRow>
                            <TableHead>Name</TableHead>
                            <TableHead>Mandatory</TableHead>
                            <TableHead className="text-right">Actions</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {categories.map((cat) => (
                            <TableRow key={cat.id}>
                                <TableCell>
                                    {editingId === cat.id ? (
                                        <Input
                                            value={editName}
                                            onChange={(e) => setEditName(e.target.value)}
                                            className="w-64"
                                            onKeyDown={(e) =>
                                                e.key === "Enter" && handleUpdate(cat.id)
                                            }
                                        />
                                    ) : (
                                        <span className="font-medium">{cat.name}</span>
                                    )}
                                </TableCell>
                                <TableCell>
                                    {editingId === cat.id ? (
                                        <Checkbox
                                            checked={editMandatory}
                                            onCheckedChange={(v) => setEditMandatory(v === true)}
                                        />
                                    ) : (
                                        <Badge variant={cat.is_mandatory ? "default" : "secondary"}>
                                            {cat.is_mandatory ? "Mandatory" : "Optional"}
                                        </Badge>
                                    )}
                                </TableCell>
                                <TableCell className="text-right">
                                    <div className="flex items-center justify-end gap-2">
                                        {editingId === cat.id ? (
                                            <>
                                                <Button
                                                    variant="outline"
                                                    size="sm"
                                                    onClick={() => handleUpdate(cat.id)}
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
                                                    onClick={() => startEdit(cat)}
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
                                                                Delete Fee Category
                                                            </AlertDialogTitle>
                                                            <AlertDialogDescription>
                                                                Are you sure you want to delete
                                                                &ldquo;{cat.name}&rdquo;? This
                                                                cannot be undone.
                                                            </AlertDialogDescription>
                                                        </AlertDialogHeader>
                                                        <AlertDialogFooter>
                                                            <AlertDialogCancel>
                                                                Cancel
                                                            </AlertDialogCancel>
                                                            <AlertDialogAction
                                                                onClick={() =>
                                                                    deleteMutation.mutate(cat.id)
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
