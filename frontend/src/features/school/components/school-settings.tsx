"use client";

import * as React from "react";
import { Pencil, Trash2, Check, X, Loader2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
    DialogTrigger,
} from "@/components/ui/dialog";
import { useMe } from "@/hooks/use-auth";
import { useSchools, useUpdateSchool, useDeleteSchool } from "@/hooks/use-schools";
import type { School } from "@/lib/api/schools";

/** Confirmation dialog for deleting a school. */
function DeleteDialog({
    schoolName,
    schoolId,
    onDelete,
    isPending,
}: {
    schoolName: string;
    schoolId: string;
    onDelete: (id: string) => void;
    isPending: boolean;
}) {
    const [open, setOpen] = React.useState(false);

    return (
        <Dialog open={open} onOpenChange={setOpen}>
            <DialogTrigger asChild>
                <Button size="icon" variant="ghost">
                    <Trash2 className="text-destructive h-4 w-4" />
                    <span className="sr-only">Delete {schoolName}</span>
                </Button>
            </DialogTrigger>
            <DialogContent>
                <DialogHeader>
                    <DialogTitle>Delete school</DialogTitle>
                    <DialogDescription>
                        Are you sure you want to delete <strong>{schoolName}</strong>? This action
                        cannot be undone. All associated data will be deactivated.
                    </DialogDescription>
                </DialogHeader>
                <DialogFooter>
                    <Button variant="outline" onClick={() => setOpen(false)}>
                        Cancel
                    </Button>
                    <Button
                        variant="destructive"
                        onClick={() => {
                            onDelete(schoolId);
                            setOpen(false);
                        }}
                        disabled={isPending}
                    >
                        {isPending ? "Deleting..." : "Delete"}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}

export function SchoolSettings() {
    const { data: me, isLoading: meLoading } = useMe();
    const tenantId = me?.tenant_id;
    const { data: schools, isLoading: schoolsLoading } = useSchools(tenantId);
    const updateSchool = useUpdateSchool();
    const deleteSchool = useDeleteSchool();

    // Inline editing state
    const [editingId, setEditingId] = React.useState<string | null>(null);
    const [editName, setEditName] = React.useState("");

    const isLoading = meLoading || schoolsLoading;

    const handleStartEdit = (school: School) => {
        if (!school.id || !school.name) return;
        setEditingId(school.id);
        setEditName(school.name);
    };

    const handleCancelEdit = () => {
        setEditingId(null);
        setEditName("");
    };

    const handleSaveEdit = (schoolId: string) => {
        if (!editName.trim()) return;
        updateSchool.mutate(
            { schoolId, payload: { name: editName.trim() } },
            { onSuccess: () => handleCancelEdit() }
        );
    };

    const handleDelete = (schoolId: string) => {
        deleteSchool.mutate(schoolId);
    };

    if (isLoading) {
        return (
            <div className="space-y-4">
                <Skeleton className="h-8 w-48" />
                <Skeleton className="h-24 w-full" />
                <Skeleton className="h-24 w-full" />
            </div>
        );
    }

    if (!me || !tenantId) return null;

    const activeSchools = (schools ?? []).filter((s): s is School & { id: string; name: string } =>
        Boolean(s.id && s.name)
    );

    return (
        <div className="mx-auto flex w-full max-w-2xl flex-col gap-8 p-8">
            <div>
                <h1 className="text-2xl font-semibold">Schools</h1>
                <p className="text-muted-foreground mt-1 text-sm">
                    Manage all schools in your organisation. You can update names or remove schools.
                </p>
            </div>

            {activeSchools.length === 0 ? (
                <Card>
                    <CardHeader>
                        <CardTitle>No schools found</CardTitle>
                        <CardDescription>
                            There are no schools in your organisation yet.
                        </CardDescription>
                    </CardHeader>
                </Card>
            ) : (
                <div className="space-y-3">
                    {activeSchools.map((school) => {
                        const isEditing = editingId === school.id;
                        const isPending =
                            updateSchool.isPending &&
                            updateSchool.variables?.schoolId === school.id;

                        return (
                            <Card key={school.id}>
                                <CardContent className="flex items-center gap-3 pt-4">
                                    {isEditing ? (
                                        <>
                                            <Input
                                                value={editName}
                                                onChange={(e) => setEditName(e.target.value)}
                                                onKeyDown={(e) => {
                                                    if (e.key === "Enter")
                                                        handleSaveEdit(school.id);
                                                    if (e.key === "Escape") handleCancelEdit();
                                                }}
                                                disabled={isPending}
                                                className="flex-1"
                                                autoFocus
                                            />
                                            <Button
                                                size="icon"
                                                variant="ghost"
                                                onClick={() => handleSaveEdit(school.id)}
                                                disabled={isPending || !editName.trim()}
                                            >
                                                {isPending ? (
                                                    <Loader2 className="h-4 w-4 animate-spin" />
                                                ) : (
                                                    <Check className="h-4 w-4" />
                                                )}
                                                <span className="sr-only">Save</span>
                                            </Button>
                                            <Button
                                                size="icon"
                                                variant="ghost"
                                                onClick={handleCancelEdit}
                                                disabled={isPending}
                                            >
                                                <X className="h-4 w-4" />
                                                <span className="sr-only">Cancel</span>
                                            </Button>
                                        </>
                                    ) : (
                                        <>
                                            <span className="flex-1 text-sm font-medium">
                                                {school.name}
                                            </span>
                                            <Button
                                                size="icon"
                                                variant="ghost"
                                                onClick={() => handleStartEdit(school)}
                                            >
                                                <Pencil className="h-4 w-4" />
                                                <span className="sr-only">Edit {school.name}</span>
                                            </Button>
                                            <DeleteDialog
                                                schoolName={school.name}
                                                schoolId={school.id}
                                                onDelete={handleDelete}
                                                isPending={deleteSchool.isPending}
                                            />
                                        </>
                                    )}
                                </CardContent>
                            </Card>
                        );
                    })}
                </div>
            )}
        </div>
    );
}
