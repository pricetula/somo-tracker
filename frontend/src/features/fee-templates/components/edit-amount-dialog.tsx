"use client";

import { useState, useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog";
import { getErrorMessage } from "@/lib/errors";
import { updateFeeTemplate, type FeeTemplate } from "@/lib/api/billing";

export function EditAmountDialog({
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
