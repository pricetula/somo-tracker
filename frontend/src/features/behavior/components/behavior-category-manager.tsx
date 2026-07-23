/**
 * BehaviorCategoryManager — CRUD table for behavior categories.
 *
 * Admin-only. Shows name, default severity, active toggle.
 * Deactivates (soft-delete) rather than hard-deleting.
 */

"use client";

import { useState } from "react";
import { Loader2, Plus } from "lucide-react";
import { Skeleton } from "@/components/ui/skeleton";
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";

import {
    useBehaviorCategories,
    useCreateBehaviorCategory,
    useUpdateBehaviorCategory,
} from "../hooks/use-behavior";

export function BehaviorCategoryManager() {
    const { data, isLoading, isError } = useBehaviorCategories();
    const createCategory = useCreateBehaviorCategory();
    const updateCategory = useUpdateBehaviorCategory();

    const [newName, setNewName] = useState("");
    const [newSeverity, setNewSeverity] = useState<string>("");

    const categories = data?.items ?? [];

    const handleCreate = () => {
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
                },
            }
        );
    };

    const handleToggleActive = (id: string, currentActive: boolean) => {
        updateCategory.mutate({ id, payload: { is_active: !currentActive } });
    };

    const handleSeverityChange = (id: string, severity: string) => {
        updateCategory.mutate({
            id,
            payload: {
                default_severity:
                    severity === "__none__" ? null : (severity as "MINOR" | "NEEDS_FOLLOW_UP"),
            },
        });
    };

    if (isLoading) {
        return (
            <div className="space-y-3">
                <Skeleton className="h-8 w-48" />
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-10 w-full" />
            </div>
        );
    }

    if (isError) {
        return (
            <div className="border-destructive/50 text-destructive rounded-md border p-4">
                Failed to load behavior categories.
            </div>
        );
    }

    return (
        <div className="space-y-4">
            <h2 className="text-xl font-semibold">Behavior Categories</h2>

            {/* Add new category row */}
            <div className="flex items-end gap-3">
                <div className="flex-1 space-y-1">
                    <label className="font-medium">Category Name</label>
                    <Input
                        placeholder="e.g. Noise Making"
                        value={newName}
                        onChange={(e) => setNewName(e.target.value)}
                    />
                </div>
                <div className="w-40 space-y-1">
                    <label className="font-medium">Default Severity</label>
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
                <Button
                    onClick={handleCreate}
                    disabled={!newName.trim() || createCategory.isPending}
                >
                    {createCategory.isPending ? (
                        <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    ) : (
                        <Plus className="mr-2 h-4 w-4" />
                    )}
                    Add
                </Button>
            </div>

            {/* Categories table */}
            <Table>
                <TableHeader>
                    <TableRow>
                        <TableHead>Name</TableHead>
                        <TableHead className="w-40">Default Severity</TableHead>
                        <TableHead className="w-20">Active</TableHead>
                    </TableRow>
                </TableHeader>
                <TableBody>
                    {categories.length === 0 ? (
                        <TableRow>
                            <TableCell colSpan={3} className="text-muted-foreground text-center">
                                No categories defined yet.
                            </TableCell>
                        </TableRow>
                    ) : (
                        categories.map((cat) => (
                            <TableRow key={cat.id}>
                                <TableCell className="font-medium">{cat.name}</TableCell>
                                <TableCell>
                                    <Select
                                        value={cat.default_severity ?? "__none__"}
                                        onValueChange={(val) => handleSeverityChange(cat.id, val)}
                                    >
                                        <SelectTrigger className="h-8">
                                            <SelectValue placeholder="None" />
                                        </SelectTrigger>
                                        <SelectContent>
                                            <SelectItem value="__none__">None</SelectItem>
                                            <SelectItem value="MINOR">Minor</SelectItem>
                                            <SelectItem value="NEEDS_FOLLOW_UP">
                                                Needs Follow-up
                                            </SelectItem>
                                        </SelectContent>
                                    </Select>
                                </TableCell>
                                <TableCell>
                                    <Switch
                                        checked={cat.is_active}
                                        onCheckedChange={() =>
                                            handleToggleActive(cat.id, cat.is_active)
                                        }
                                    />
                                </TableCell>
                            </TableRow>
                        ))
                    )}
                </TableBody>
            </Table>
        </div>
    );
}
