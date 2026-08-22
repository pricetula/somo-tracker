/**
 * Create/Edit Strand Dialog — modal form for creating or editing a strand.
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
import { Label } from "@/components/ui/label";
import { useCreateStrand } from "../hooks/use-curriculum";

// ─── Props ─────────────────────────────────────────────────────────────────

interface CreateStrandDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    learningAreaId: string;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function CreateStrandDialog({
    open,
    onOpenChange,
    learningAreaId,
}: CreateStrandDialogProps) {
    const createMutation = useCreateStrand();
    const {
        register,
        handleSubmit,
        reset,
        formState: { errors, isSubmitting },
    } = useForm<{ name: string }>({
        defaultValues: { name: "" },
    });

    React.useEffect(() => {
        if (open) reset();
    }, [open, reset]);

    const onSubmit = handleSubmit(async (data) => {
        await createMutation.mutateAsync({
            learning_area_id: learningAreaId,
            name: data.name,
        });
        onOpenChange(false);
    });

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="sm:max-w-[425px]">
                <DialogHeader>
                    <DialogTitle>Add Strand</DialogTitle>
                    <DialogDescription>
                        Create a new strand within this learning area.
                    </DialogDescription>
                </DialogHeader>

                <form onSubmit={onSubmit} className="space-y-4">
                    <div className="space-y-2">
                        <Label htmlFor="strand-name">Name</Label>
                        <Input
                            id="strand-name"
                            placeholder="e.g. Numbers and Operations"
                            {...register("name", { required: "Name is required" })}
                        />
                        {errors.name && (
                            <p className="text-destructive text-xs">{errors.name.message}</p>
                        )}
                    </div>

                    <DialogFooter>
                        <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                            Cancel
                        </Button>
                        <Button type="submit" disabled={isSubmitting || createMutation.isPending}>
                            {createMutation.isPending ? "Creating..." : "Create"}
                        </Button>
                    </DialogFooter>
                </form>
            </DialogContent>
        </Dialog>
    );
}
