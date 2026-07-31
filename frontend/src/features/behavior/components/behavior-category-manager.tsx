"use client";

import { useState, useCallback } from "react";
import { Loader2, Plus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
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
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog";
import { DataTable } from "@/components/shared/data-table";
import { type DataTableColumn } from "@/components/shared/data-table/types";
import { useBehaviorCategories, useCreateBehaviorCategory } from "../hooks/use-behavior";
import { type BehaviorCategory } from "@/lib/api/behavior";

function createColumns(): DataTableColumn<BehaviorCategory>[] {
    return [
        {
            id: "name",
            header: "Name",
            cell: (row) => <span className="font-medium">{row.name}</span>,
        },
        {
            id: "default_severity",
            header: "Default Severity",
            width: "200px",
            cell: (row) => <SeverityCell category={row} />,
        },
        {
            id: "is_active",
            header: "Active",
            width: "100px",
            cell: (row) => <ActiveToggleCell category={row} />,
        },
        {
            id: "actions",
            header: "",
            width: "48px",
            align: "right",
            cell: (row) => <ActionsCell category={row} />,
        },
    ];
}

import { SeverityCell } from "./severity-cell";
import { ActiveToggleCell } from "./active-toggle-cell";
import { ActionsCell } from "./category-actions-cell";

export function BehaviorCategoryManager() {
    const { data, isLoading } = useBehaviorCategories();
    const createCategory = useCreateBehaviorCategory();

    const [newName, setNewName] = useState("");
    const [newSeverity, setNewSeverity] = useState<string>("");
    const [dialogOpen, setDialogOpen] = useState(false);

    const columns = createColumns();
    const categories = data?.items ?? [];

    const handleCreate = useCallback(() => {
        if (!newName.trim()) return;
        createCategory.mutate(
            {
                name: newName.trim(),
                default_severity: (newSeverity as "MINOR" | "NEEDS_FOLLOW_UP") || undefined,
            },
            {
                onSuccess: () => {
                    setNewName("");
                    setNewSeverity("");
                    setDialogOpen(false);
                },
            }
        );
    }, [newName, newSeverity, createCategory]);

    if (isLoading) {
        return (
            <div className="space-y-3">
                <div className="flex items-center gap-3">
                    <Input placeholder="Category Name" className="max-w-xs" disabled />
                    <Button disabled>Add</Button>
                </div>
                <DataTable
                    queryKey={["behavior", "categories", "loading"]}
                    queryFn={() => Promise.resolve({ items: [] })}
                    columns={columns}
                    getRowId={(row) => row.id}
                    emptyState="Loading..."
                    height={200}
                />
            </div>
        );
    }

    return (
        <div className="space-y-4">
            <DataTable
                queryKey={["behavior", "categories"]}
                queryFn={() => Promise.resolve({ items: categories, total: categories.length })}
                columns={columns}
                getRowId={(row) => row.id}
                emptyState="No categories defined yet."
                noResultsState="No categories match your search."
                renderToolBarComponents={() => (
                    <Button
                        key="add-category"
                        variant="outline"
                        size="sm"
                        onClick={() => setDialogOpen(true)}
                    >
                        <Plus className="mr-1 size-4" />
                        Add Category
                    </Button>
                )}
            />

            <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
                <DialogContent className="sm:max-w-sm">
                    <DialogHeader>
                        <DialogTitle>Add Behavior Category</DialogTitle>
                        <DialogDescription>
                            Create a new behavior category for incident/behavior reporting.
                        </DialogDescription>
                    </DialogHeader>
                    <div className="space-y-4">
                        <div className="space-y-1">
                            <label className="text-sm font-medium">Category Name</label>
                            <Input
                                placeholder="e.g. Noise Making"
                                value={newName}
                                onChange={(e) => setNewName(e.target.value)}
                                onKeyDown={(e) => e.key === "Enter" && handleCreate()}
                            />
                        </div>
                        <div className="space-y-1">
                            <label className="text-sm font-medium">Default Severity</label>
                            <Select value={newSeverity} onValueChange={setNewSeverity}>
                                <SelectTrigger>
                                    <SelectValue placeholder="Optional" />
                                </SelectTrigger>
                                <SelectContent>
                                    <SelectItem value="MINOR">Minor</SelectItem>
                                    <SelectItem value="NEEDS_FOLLOW_UP">Needs Follow-up</SelectItem>
                                </SelectContent>
                            </Select>
                        </div>
                    </div>
                    <DialogFooter>
                        <Button
                            variant="outline"
                            onClick={() => setDialogOpen(false)}
                            disabled={createCategory.isPending}
                        >
                            Cancel
                        </Button>
                        <Button
                            onClick={handleCreate}
                            disabled={!newName.trim() || createCategory.isPending}
                        >
                            {createCategory.isPending ? (
                                <>
                                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                                    Adding...
                                </>
                            ) : (
                                "Add"
                            )}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </div>
    );
}
