/**
 * AddClassForm — Creates a new class.
 *
 * Uses reusable comboboxes from their respective feature modules:
 *  - GradeLevelCombobox from grade-level feature
 *  - StreamCombobox from streams feature
 *
 * The academic_year_id and academic_term_id are resolved server-side
 * from the current active academic year/term.
 *
 * Form validation is handled by zod + react-hook-form.
 */

"use client";

import React from "react";
import { z } from "zod";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { GradeLevelCombobox } from "@/features/grade-level";
import { StreamCombobox } from "@/features/streams";
import { Button } from "@/components/ui/button";
import {
    Form,
    FormControl,
    FormField,
    FormItem,
    FormLabel,
    FormMessage,
} from "@/components/ui/form";
import { useCreateClass } from "../hooks/use-classes";

// ─── Zod Schema ────────────────────────────────────────────────────────────

const createClassSchema = z.object({
    grade_level: z.string().min(1, "Grade level is required"),
    stream_id: z.string().min(1, "Stream is required"),
    student_ids: z.array(z.string()),
});

type CreateClassSchema = z.infer<typeof createClassSchema>;

// ─── Props ─────────────────────────────────────────────────────────────────

interface AddClassFormProps {
    /** Called when the class is successfully created. */
    onSuccess?: () => void;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function AddClassForm({ onSuccess }: AddClassFormProps) {
    const router = useRouter();

    const form = useForm<CreateClassSchema>({
        resolver: zodResolver(createClassSchema),
        defaultValues: {
            grade_level: "",
            stream_id: "",
            student_ids: [] as string[],
        },
    });

    const createClassMutation = useCreateClass();

    const onSubmit = (data: CreateClassSchema) => {
        createClassMutation.mutate(data, {
            onSuccess,
        });
    };

    const handleCancel = () => {
        router.back();
    };

    return (
        <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
                <FormField
                    control={form.control}
                    name="grade_level"
                    render={({ field }) => {
                        return (
                            <FormItem>
                                <FormLabel>Grade Level</FormLabel>
                                <FormControl>
                                    <GradeLevelCombobox
                                        value={field.value}
                                        onChange={(value: string) => {
                                            field.onChange({
                                                target: { value },
                                            } as React.ChangeEvent<HTMLSelectElement>);
                                        }}
                                    />
                                </FormControl>
                                <FormMessage />
                            </FormItem>
                        );
                    }}
                />

                <FormField
                    control={form.control}
                    name="stream_id"
                    render={({ field }) => {
                        return (
                            <FormItem>
                                <FormLabel>Stream</FormLabel>
                                <FormControl>
                                    <StreamCombobox
                                        value={field.value}
                                        onChange={(value: string) => {
                                            field.onChange({
                                                target: { value },
                                            } as React.ChangeEvent<HTMLSelectElement>);
                                        }}
                                    />
                                </FormControl>
                                <FormMessage />
                            </FormItem>
                        );
                    }}
                />

                <div className="flex gap-2 pt-4">
                    <Button type="submit" disabled={createClassMutation.isPending}>
                        {createClassMutation.isPending ? "Creating..." : "Create Class"}
                    </Button>
                    <Button type="button" variant="outline" onClick={handleCancel}>
                        Cancel
                    </Button>
                </div>
            </form>
        </Form>
    );
}
