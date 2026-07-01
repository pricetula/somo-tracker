/**
 * Create Learning Area Dialog — modal form for creating a new learning area.
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
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { useCreateLearningArea } from "../hooks/use-curriculum";
import { isApiError } from "@/lib/errors";

// ─── Props ─────────────────────────────────────────────────────────────────

interface CreateLearningAreaDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function CreateLearningAreaDialog({ open, onOpenChange }: CreateLearningAreaDialogProps) {
    const createMutation = useCreateLearningArea();
    const {
        register,
        handleSubmit,
        setValue,
        setError,
        reset,
        formState: { errors, isSubmitting },
    } = useForm<{
        code: string;
        name: string;
        education_level: string;
    }>({
        defaultValues: {
            code: "",
            name: "",
            education_level: "",
        },
    });

    React.useEffect(() => {
        if (open) {
            reset();
        }
    }, [open, reset]);

    const onSubmit = handleSubmit(async (data) => {
        try {
            await createMutation.mutateAsync({
                code: data.code.toUpperCase(),
                name: data.name,
                education_level: data.education_level,
            });
            onOpenChange(false);
        } catch (err) {
            if (isApiError(err) && err.status === 400 && err.errors) {
                for (const [field, messages] of Object.entries(err.errors)) {
                    if (field === "code" || field === "name" || field === "education_level") {
                        setError(field, { message: messages[0] });
                    }
                }
            }
        }
    });

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="sm:max-w-[425px]">
                <DialogHeader>
                    <DialogTitle>Add Learning Area</DialogTitle>
                    <DialogDescription>
                        Create a new CBC learning area (subject) for your school.
                    </DialogDescription>
                </DialogHeader>

                <form onSubmit={onSubmit} className="space-y-4">
                    <div className="space-y-2">
                        <Label htmlFor="code">Code</Label>
                        <Input
                            id="code"
                            placeholder="e.g. MATH, INT_SCI"
                            {...register("code", { required: "Code is required" })}
                        />
                        {errors.code && (
                            <p className="text-destructive text-xs">{errors.code.message}</p>
                        )}
                    </div>

                    <div className="space-y-2">
                        <Label htmlFor="name">Name</Label>
                        <Input
                            id="name"
                            placeholder="e.g. Mathematics"
                            {...register("name", { required: "Name is required" })}
                        />
                        {errors.name && (
                            <p className="text-destructive text-xs">{errors.name.message}</p>
                        )}
                    </div>

                    <div className="space-y-2">
                        <Label htmlFor="education_level">Education Level</Label>
                        <Select onValueChange={(v) => setValue("education_level", v)}>
                            <SelectTrigger id="education_level">
                                <SelectValue placeholder="Select education level" />
                            </SelectTrigger>
                            <SelectContent>
                                <SelectItem value="Early_Years">Early Years</SelectItem>
                                <SelectItem value="Upper_Primary">Upper Primary</SelectItem>
                                <SelectItem value="Junior_Secondary">Junior Secondary</SelectItem>
                                <SelectItem value="Senior_School">Senior School</SelectItem>
                            </SelectContent>
                        </Select>
                        {errors.education_level && (
                            <p className="text-destructive text-xs">
                                {errors.education_level.message}
                            </p>
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
