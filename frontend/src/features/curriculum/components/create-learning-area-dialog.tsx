/**
 * Create Learning Area Dialog — modal form for creating a new learning area.
 */

"use client";

import * as React from "react";
import { useForm, useWatch } from "react-hook-form";

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
import { useCreateLearningArea } from "../hooks/use-curriculum";
import { isApiError } from "@/lib/errors";
import { EducationLevelCombobox } from "@/features/education-level";
import { GradeLevelCombobox } from "@/features/grade-level";

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
        control,
        setError,
        reset,
        formState: { errors, isSubmitting },
    } = useForm<{
        code: string;
        name: string;
        education_level: string;
        grade_level: string;
    }>({
        defaultValues: {
            code: "",
            name: "",
            education_level: "",
            grade_level: "",
        },
    });

    const educationLevel = useWatch({ control, name: "education_level" });
    const gradeLevel = useWatch({ control, name: "grade_level" });

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
                grade_level: data.grade_level,
            });
            onOpenChange(false);
        } catch (err) {
            if (isApiError(err) && err.status === 400 && err.errors) {
                for (const [field, messages] of Object.entries(err.errors)) {
                    if (
                        field === "code" ||
                        field === "name" ||
                        field === "education_level" ||
                        field === "grade_level"
                    ) {
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
                        {errors.code && <p className="text-destructive">{errors.code.message}</p>}
                    </div>

                    <div className="space-y-2">
                        <Label htmlFor="name">Name</Label>
                        <Input
                            id="name"
                            placeholder="e.g. Mathematics"
                            {...register("name", { required: "Name is required" })}
                        />
                        {errors.name && <p className="text-destructive">{errors.name.message}</p>}
                    </div>

                    <div className="space-y-2">
                        <Label htmlFor="education_level">Education Level</Label>
                        <EducationLevelCombobox
                            value={educationLevel}
                            onChange={(v) => setValue("education_level", v)}
                            placeholder="Select education level"
                        />
                        {errors.education_level && (
                            <p className="text-destructive">{errors.education_level.message}</p>
                        )}
                    </div>

                    <div className="space-y-2">
                        <Label htmlFor="grade_level">Grade Level</Label>
                        <GradeLevelCombobox
                            value={gradeLevel}
                            onChange={(v) => setValue("grade_level", v)}
                            placeholder="Select grade"
                        />
                        {errors.grade_level && (
                            <p className="text-destructive">{errors.grade_level.message}</p>
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
