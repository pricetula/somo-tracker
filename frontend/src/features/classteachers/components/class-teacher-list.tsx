/**
 * ClassTeacherList — displays teacher assignments for a class or teacher.
 *
 * SCHOOL_ADMIN: can view by class or by teacher, and delete assignments.
 */
"use client";

import { useState } from "react";
import {
    useClassTeachersByClass,
    useClassTeachersByTeacher,
    useDeleteClassTeacher,
} from "../hooks/use-classteachers";
import { AssignTeacherDialog } from "./assign-teacher-dialog";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Checkbox } from "@/components/ui/checkbox";
import { Plus, Trash2 } from "lucide-react";
import { RowActions } from "@/components/shared/data-table/row-actions";
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
import { toast } from "sonner";

type ViewMode = "by-class" | "by-teacher";

export function ClassTeacherList() {
    const [mode, setMode] = useState<ViewMode>("by-class");
    const [entityId, setEntityId] = useState("");
    const [showAssign, setShowAssign] = useState(false);
    const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
    const [pendingDeleteId, setPendingDeleteId] = useState<string | null>(null);
    const [bulkDeleteOpen, setBulkDeleteOpen] = useState(false);

    const byClass = useClassTeachersByClass(mode === "by-class" ? entityId : "");
    const byTeacher = useClassTeachersByTeacher(mode === "by-teacher" ? entityId : "");
    const deleteMutation = useDeleteClassTeacher();

    const data = mode === "by-class" ? byClass.data : byTeacher.data;
    const isLoading = mode === "by-class" ? byClass.isLoading : byTeacher.isLoading;
    const items = data?.items ?? [];

    // ── Selection handlers ────────────────────────────────────────
    const toggleSelect = (id: string) => {
        setSelectedIds((prev) => {
            const next = new Set(prev);
            if (next.has(id)) next.delete(id);
            else next.add(id);
            return next;
        });
    };

    const selectAll = (checked: boolean) => {
        if (checked) {
            setSelectedIds(new Set(items.map((ct) => ct.id)));
        } else {
            setSelectedIds(new Set());
        }
    };

    const allSelected = items.length > 0 && items.every((ct) => selectedIds.has(ct.id));
    const someSelected = items.some((ct) => selectedIds.has(ct.id)) && !allSelected;

    // ── Delete handlers ───────────────────────────────────────────
    const handleBulkDelete = async () => {
        const ids = [...selectedIds];
        try {
            for (const id of ids) {
                await deleteMutation.mutateAsync(id);
            }
            toast.success(`${ids.length} assignment${ids.length !== 1 ? "s" : ""} deleted.`);
            setSelectedIds(new Set());
        } catch (err) {
            toast.error(getErrorMessage(err));
        }
    };

    const handleSingleDelete = async () => {
        if (!pendingDeleteId) return;
        try {
            await deleteMutation.mutateAsync(pendingDeleteId);
            toast.success("Assignment deleted.");
            setPendingDeleteId(null);
        } catch (err) {
            toast.error(getErrorMessage(err));
        }
    };

    return (
        <div className="space-y-4">
            <div className="flex items-center justify-between gap-4">
                <div className="flex items-center gap-2">
                    <select
                        className="border-input bg-background rounded-md border px-3 py-1.5"
                        value={mode}
                        onChange={(e) => setMode(e.target.value as ViewMode)}
                    >
                        <option value="by-class">By Class</option>
                        <option value="by-teacher">By Teacher</option>
                    </select>
                    <input
                        className="border-input bg-background w-64 rounded-md border px-3 py-1.5"
                        placeholder={
                            mode === "by-class" ? "Enter Class ID…" : "Enter Teacher User ID…"
                        }
                        value={entityId}
                        onChange={(e) => setEntityId(e.target.value)}
                    />
                </div>
                <div className="flex items-center gap-2">
                    {selectedIds.size > 0 && (
                        <AlertDialog open={bulkDeleteOpen} onOpenChange={setBulkDeleteOpen}>
                            <AlertDialogTrigger asChild>
                                <Button variant="destructive" size="sm">
                                    <Trash2 className="mr-1 size-3" />
                                    Delete {selectedIds.size}
                                </Button>
                            </AlertDialogTrigger>
                            <AlertDialogContent>
                                <AlertDialogHeader>
                                    <AlertDialogTitle>
                                        Delete {selectedIds.size} assignment
                                        {selectedIds.size !== 1 ? "s" : ""}
                                    </AlertDialogTitle>
                                    <AlertDialogDescription>
                                        This action cannot be undone. The selected assignments will
                                        be permanently removed.
                                    </AlertDialogDescription>
                                </AlertDialogHeader>
                                <AlertDialogFooter>
                                    <AlertDialogCancel>Cancel</AlertDialogCancel>
                                    <AlertDialogAction onClick={handleBulkDelete}>
                                        Delete
                                    </AlertDialogAction>
                                </AlertDialogFooter>
                            </AlertDialogContent>
                        </AlertDialog>
                    )}
                    {entityId && (
                        <Button variant="outline" size="sm" onClick={() => setShowAssign(true)}>
                            <Plus className="mr-1 size-4" />
                            Assign Teacher
                        </Button>
                    )}
                </div>
            </div>

            {isLoading ? (
                <p className="text-muted-foreground">Loading assignments…</p>
            ) : items.length === 0 ? (
                <p className="text-muted-foreground">
                    {entityId
                        ? "No assignments found."
                        : "Enter a class or teacher ID to view assignments."}
                </p>
            ) : (
                <>
                    <div className="flex items-center gap-2 border-b pb-2">
                        <Checkbox
                            checked={allSelected ? true : someSelected ? "indeterminate" : false}
                            onCheckedChange={(checked) => selectAll(checked === true)}
                        />
                        <span className="text-muted-foreground text-xs">
                            {selectedIds.size > 0
                                ? `${selectedIds.size} selected`
                                : `${items.length} assignment${items.length !== 1 ? "s" : ""}`}
                        </span>
                    </div>
                    <div className="space-y-2">
                        {items.map((ct) => (
                            <div
                                key={ct.id}
                                className={`bg-muted/30 flex items-center justify-between rounded-md p-3 ${selectedIds.has(ct.id) ? "bg-accent/20" : ""}`}
                            >
                                <div className="flex items-center gap-3">
                                    <Checkbox
                                        checked={selectedIds.has(ct.id)}
                                        onCheckedChange={() => toggleSelect(ct.id)}
                                    />
                                    <div className="space-y-1">
                                        <p className="text-foreground font-medium">
                                            {ct.teacher_name ?? ct.user_id}
                                        </p>
                                        <div className="text-muted-foreground flex items-center gap-2 text-xs">
                                            <Badge variant="secondary" className="text-xs">
                                                {ct.teacher_role.replace(/_/g, " ")}
                                            </Badge>
                                            {ct.learning_area && <span>{ct.learning_area}</span>}
                                        </div>
                                    </div>
                                </div>
                                <RowActions
                                    rowId={ct.id}
                                    label="assignment"
                                    onDelete={() => setPendingDeleteId(ct.id)}
                                    disabled={deleteMutation.isPending}
                                />
                            </div>
                        ))}
                    </div>

                    {/* Single delete confirmation dialog */}
                    <AlertDialog
                        open={!!pendingDeleteId}
                        onOpenChange={(open) => {
                            if (!open) setPendingDeleteId(null);
                        }}
                    >
                        <AlertDialogContent>
                            <AlertDialogHeader>
                                <AlertDialogTitle>Delete Assignment</AlertDialogTitle>
                                <AlertDialogDescription>
                                    Are you sure you want to delete this assignment? This cannot be
                                    undone.
                                </AlertDialogDescription>
                            </AlertDialogHeader>
                            <AlertDialogFooter>
                                <AlertDialogCancel>Cancel</AlertDialogCancel>
                                <AlertDialogAction onClick={handleSingleDelete}>
                                    Delete
                                </AlertDialogAction>
                            </AlertDialogFooter>
                        </AlertDialogContent>
                    </AlertDialog>
                </>
            )}

            <AssignTeacherDialog
                open={showAssign}
                onOpenChange={setShowAssign}
                prefillClassId={mode === "by-class" ? entityId : undefined}
            />
        </div>
    );
}
