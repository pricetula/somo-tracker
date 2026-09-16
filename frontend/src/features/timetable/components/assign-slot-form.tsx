"use client";

import React from "react";
import { z } from "zod";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
    Form,
    FormControl,
    FormField,
    FormItem,
    FormLabel,
    FormMessage,
} from "@/components/ui/form";
import { useAssignSlot } from "../hooks/use-assign-slot";
import { ApiError } from "@/lib/api/client";
import { getErrorMessage } from "@/lib/errors";
import { ClassesCombobox } from "@/features/classes/components/classes-combobox";
import { TeachersCombobox } from "@/features/teachers/components/teachers-combobox";
import { SubjectsCombobox } from "@/features/curriculum/components/subjects-combobox";

const schema = z.object({
    class_room_id: z.string().min(1, "Class is required"),
    subject_id: z.string().min(1, "Subject is required"),
    teacher_membership_id: z.string().min(1, "Teacher is required"),
});

type FormValues = z.infer<typeof schema>;

interface AssignSlotFormProps {
    dayOfWeek: number;
    timeSlotId: string;
    onSuccess?: () => void;
}

export function AssignSlotForm({ dayOfWeek, timeSlotId, onSuccess }: AssignSlotFormProps) {
    const { mutate, isPending } = useAssignSlot();

    const form = useForm<FormValues>({
        resolver: zodResolver(schema),
        defaultValues: {
            class_room_id: "",
            subject_id: "",
            teacher_membership_id: "",
        },
    });

    const onSubmit = React.useCallback(
        (data: FormValues) => {
            mutate(
                {
                    class_room_id: data.class_room_id,
                    day_of_week: dayOfWeek,
                    time_slot_id: timeSlotId,
                    subject_id: data.subject_id,
                    teacher_membership_id: data.teacher_membership_id,
                },
                {
                    onSuccess: () => {
                        onSuccess?.();
                    },
                    onError: (err) => {
                        if (err instanceof ApiError && err.status === 400 && err.errors) {
                            Object.entries(err.errors).forEach(([field, messages]) => {
                                form.setError(field as keyof FormValues, {
                                    type: "server",
                                    message: messages?.[0],
                                });
                            });
                        } else {
                            toast.error(getErrorMessage(err));
                        }
                    },
                }
            );
        },
        [mutate, dayOfWeek, timeSlotId, onSuccess, form]
    );

    return (
        <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-5">
                <FormField
                    control={form.control}
                    name="class_room_id"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>Class</FormLabel>
                            <FormControl>
                                <ClassesCombobox
                                    value={field.value}
                                    onChange={field.onChange}
                                    disabled={isPending}
                                    placeholder="Select class"
                                />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />

                <FormField
                    control={form.control}
                    name="subject_id"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>Subject</FormLabel>
                            <FormControl>
                                <SubjectsCombobox
                                    value={field.value}
                                    onChange={field.onChange}
                                    disabled={isPending}
                                    placeholder="Select subject"
                                />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />

                <FormField
                    control={form.control}
                    name="teacher_membership_id"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>Teacher</FormLabel>
                            <FormControl>
                                <TeachersCombobox
                                    value={field.value}
                                    onChange={field.onChange}
                                    disabled={isPending}
                                    placeholder="Select teacher"
                                />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />

                <Button type="submit" className="w-full" disabled={isPending}>
                    {isPending ? (
                        <>
                            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                            Assigning…
                        </>
                    ) : (
                        "Assign Slot"
                    )}
                </Button>
            </form>
        </Form>
    );
}
