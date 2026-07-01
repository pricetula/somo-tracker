/**
 * Create/Edit Performance Indicator Dialog — modal form for creating or editing
 * a performance indicator.
 */

"use client";

import * as React from "react";
import { useForm } from "react-hook-form";

import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import {
    useCreatePerformanceIndicator,
    useUpdatePerformanceIndicator,
} from "../hooks/use-curriculum";
import type { PerformanceIndicator } from "@/lib/api/curriculum";

// ─── Props ─────────────────────────────────────────────────────────────────

interface CreateIndicatorDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    subStrandId: string;
    /** When provided, the dialog opens in edit mode. */
    indicator?: PerformanceIndicator | null;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function CreateIndicatorDialog({
    open,
    onOpenChange,
    subStrandId,
    indicator,
}: CreateIndicatorDialogProps) {
    const createMutation = useCreatePerformanceIndicator();
    const updateMutation = useUpdatePerformanceIndicator();
    const isEdit = !!indicator;

    const {
        register,
        handleSubmit,
        reset,
        formState: { errors, isSubmitting },
    } = useForm<{ description: string; sequence_order: number }>({
        defaultValues: {
            description: "",
            sequence_order: 0,
        },
    });

    React.useEffect(() => {
        if (open) {
            if (indicator) {
                reset({
                    description: indicator.description,
                    sequence_order: indicator.sequence_order,
                });
            } else {
                reset({ description: "", sequence_order: 0 });
            }
        }
    }, [open, indicator, reset]);

    const isPending = createMutation.isPending || updateMutation.isPending;

    const onSubmit = handleSubmit(async (data) => {
        if (isEdit && indicator) {
            const payload: { description?: string; sequence_order?: number } = {};
            if (data.description !== indicator.description) payload.description = data.description;
            if (data.sequence_order !== indicator.sequence_order)
                payload.sequence_order = data.sequence_order;

            if (Object.keys(payload).length > 0) {
                await updateMutation.mutateAsync({
                    id: indicator.id,
                    data: payload,
                    learningAreaId: "",
                });
            }
        } else {
            await createMutation.mutateAsync({
                sub_strand_id: subStrandId,
                description: data.description,
                sequence_order: data.sequence_order || undefined,
            });
        }
        onOpenChange(false);
    });

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="sm:max-w-[525px]">
                <DialogHeader>
                    <DialogTitle>
                        {isEdit ? "Edit Performance Indicator" : "Add Performance Indicator"}
                    </DialogTitle>
                    <DialogDescription>
                        {isEdit
                            ? "Update the performance indicator details."
                            : "Define a new performance indicator for this sub-strand."}
                    </DialogDescription>
                </DialogHeader>

                <form onSubmit={onSubmit} className="space-y-4">
                    <div className="space-y-2">
                        <Label htmlFor="indicator-description">Description</Label>
                        <Textarea
                            id="indicator-description"
                            placeholder="Describe the learning outcome..."
                            rows={3}
                            {...register("description", {
                                required: "Description is required",
                            })}
                        />
                        {errors.description && (
                            <p className="text-destructive text-xs">{errors.description.message}</p>
                        )}
                    </div>

                    <div className="space-y-2">
                        <Label htmlFor="indicator-sequence">Sequence Order</Label>
                        <Input
                            id="indicator-sequence"
                            type="number"
                            min={1}
                            placeholder="Auto if empty"
                            {...register("sequence_order", { valueAsNumber: true })}
                        />
                        <p className="text-muted-foreground text-xs">
                            Leave empty to auto-increment.
                        </p>
                    </div>

                    <DialogFooter>
                        <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                            Cancel
                        </Button>
                        <Button type="submit" disabled={isSubmitting || isPending}>
                            {isPending ? "Saving..." : isEdit ? "Save Changes" : "Create"}
                        </Button>
                    </DialogFooter>
                </form>
            </DialogContent>
        </Dialog>
    );
}
