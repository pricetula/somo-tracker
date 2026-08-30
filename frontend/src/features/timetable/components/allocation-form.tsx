"use client";

import * as React from "react";
import { useSearchParams } from "next/navigation";
import { z } from "zod";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { toast } from "sonner";

import { useCreateAllocations } from "../hooks";
import { LearningAreaCombobox } from "@/features/curriculum";
import { TeacherCombobox } from "@/features/teachers";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
    Form,
    FormControl,
    FormField,
    FormItem,
    FormLabel,
    FormMessage,
} from "@/components/ui/form";
import { getErrorMessage } from "@/lib/errors";

// ─── Schema ────────────────────────────────────────────────────────────────

const allocationSchema = z.object({
    block_id: z.string().min(1, "Block ID is required"),
    class_id: z.string().min(1, "Class ID is required"),
    learning_area_id: z.string().min(1, "Learning area is required"),
    teacher_id: z.string().min(1, "Teacher is required"),
    room_identifier: z.string().optional(),
});

type AllocationFormData = z.infer<typeof allocationSchema>;

// ─── Props ───────────────────────────────────────────────────────────────

interface AllocationFormProps {
    onSuccess?: () => void;
}

// ─── Component ────────────────────────────────────────────────────────────

export function AllocationForm({ onSuccess }: AllocationFormProps) {
    const searchParams = useSearchParams();

    const blockId = searchParams.get("block") ?? "";
    const classId = searchParams.get("class") ?? "";
    const dayRaw = searchParams.get("day") ?? "";

    const dayOfWeek = React.useMemo(() => {
        const n = parseInt(dayRaw, 10);
        return !isNaN(n) && n >= 1 && n <= 7 ? n : null;
    }, [dayRaw]);

    const mutation = useCreateAllocations();

    const form = useForm<AllocationFormData>({
        resolver: zodResolver(allocationSchema),
        defaultValues: {
            block_id: blockId,
            class_id: classId,
            learning_area_id: "",
            teacher_id: "",
            room_identifier: "",
        },
    });

    // Sync URL params into form defaults when they change
    React.useEffect(() => {
        form.reset({
            block_id: blockId,
            class_id: classId,
            learning_area_id: form.getValues("learning_area_id"),
            teacher_id: form.getValues("teacher_id"),
            room_identifier: form.getValues("room_identifier"),
        });
    }, [blockId, classId, form]);

    const onSubmit = (data: AllocationFormData) => {
        mutation.mutate(
            [
                {
                    block_id: data.block_id,
                    class_id: data.class_id,
                    learning_area_id: data.learning_area_id,
                    teacher_id: data.teacher_id,
                    room_identifier: data.room_identifier || null,
                },
            ],
            {
                onSuccess: () => {
                    toast.success("Allocation created");
                    if (onSuccess) onSuccess();
                },
                onError: (err: unknown) => {
                    // Canonical 400 -> field-level errors via form.setError
                    if (
                        err &&
                        typeof err === "object" &&
                        "status" in err &&
                        (err as { status: number }).status === 400 &&
                        "errors" in err &&
                        (err as { errors?: Record<string, string[]> }).errors
                    ) {
                        const errorMap = (err as { errors: Record<string, string[]> }).errors;
                        Object.entries(errorMap).forEach(([field, messages]) => {
                            form.setError(field as keyof AllocationFormData, {
                                message: messages?.[0] ?? "Invalid",
                            });
                        });
                    } else {
                        toast.error(getErrorMessage(err));
                    }
                },
            }
        );
    };

    // Guard against missing / invalid URL state
    if (dayOfWeek === null) {
        return (
            <div className="bg-destructive/10 text-destructive rounded-md px-4 py-3">
                Invalid day parameter (<code>?day=</code> must be a number 1–7).
            </div>
        );
    }
    if (!blockId) {
        return (
            <div className="bg-destructive/10 text-destructive rounded-md px-4 py-3">
                Missing <code>?block=</code> parameter.
            </div>
        );
    }
    if (!classId) {
        return (
            <div className="bg-destructive/10 text-destructive rounded-md px-4 py-3">
                Missing <code>?class=</code> parameter.
            </div>
        );
    }

    return (
        <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-5">
                {/* Hidden form fields for URL-derived IDs */}
                <FormField
                    control={form.control}
                    name="block_id"
                    render={({ field }) => <input type="hidden" {...field} value={field.value} />}
                />
                <FormField
                    control={form.control}
                    name="class_id"
                    render={({ field }) => <input type="hidden" {...field} value={field.value} />}
                />

                {/* Learning Area */}
                <FormField
                    control={form.control}
                    name="learning_area_id"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>Learning Area</FormLabel>
                            <FormControl>
                                <LearningAreaCombobox
                                    value={field.value}
                                    onChange={(value: string | string[]) =>
                                        field.onChange(
                                            typeof value === "string" ? value : (value?.[0] ?? "")
                                        )
                                    }
                                    placeholder="Select a learning area..."
                                />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />

                {/* Teacher */}
                <FormField
                    control={form.control}
                    name="teacher_id"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>Teacher</FormLabel>
                            <FormControl>
                                <TeacherCombobox
                                    value={field.value}
                                    onChange={(value: string | string[]) =>
                                        field.onChange(
                                            typeof value === "string" ? value : (value?.[0] ?? "")
                                        )
                                    }
                                    placeholder="Select a teacher..."
                                />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />

                {/* Room */}
                <FormField
                    control={form.control}
                    name="room_identifier"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>Room (optional)</FormLabel>
                            <FormControl>
                                <Input
                                    {...field}
                                    value={field.value ?? ""}
                                    placeholder="Room number / identifier"
                                    onChange={field.onChange}
                                />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />

                <div className="flex gap-2 pt-2">
                    <Button type="submit" disabled={mutation.isPending}>
                        {mutation.isPending ? "Creating..." : "Create Allocation"}
                    </Button>
                </div>
            </form>
        </Form>
    );
}
