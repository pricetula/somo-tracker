"use client";

import { useState, useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog";
import { getErrorMessage } from "@/lib/errors";
import { updateFeeCategory, type FeeCategory } from "@/lib/api/billing";

export function EditCategoryDialog({
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
