/**
 * Create/Edit Sub-Strand Dialog — modal form for creating or editing a sub-strand.
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
import { useCreateSubStrand } from "../hooks/use-curriculum";

// ─── Props ─────────────────────────────────────────────────────────────────

interface CreateSubStrandDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    strandId: string;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function CreateSubStrandDialog({
    open,
    onOpenChange,
    strandId,
}: CreateSubStrandDialogProps) {
    const createMutation = useCreateSubStrand();
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
            strand_id: strandId,
            name: data.name,
        });
        onOpenChange(false);
    });

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="sm:max-w-[425px]">
                <DialogHeader>
                    <DialogTitle>Add Sub-Strand</DialogTitle>
                    <DialogDescription>
                        Create a new sub-strand within this strand.
                    </DialogDescription>
                </DialogHeader>

                <form onSubmit={onSubmit} className="space-y-4">
                    <div className="space-y-2">
                        <Label htmlFor="sub-strand-name">Name</Label>
                        <Input
                            id="sub-strand-name"
                            placeholder="e.g. Addition and Subtraction"
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
