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
import { createFeeCategory } from "@/lib/api/billing";

export function CreateCategoryDialog({
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
