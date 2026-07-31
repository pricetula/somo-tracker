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
import { createFeeTemplate } from "@/lib/api/billing";

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

export function CreateTemplateDialog({
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
