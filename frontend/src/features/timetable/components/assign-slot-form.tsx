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
import { TeachersCombobox } from "@/features/teachers/components/teachers-combobox";
import { SubjectsCombobox } from "@/features/curriculum/components/subjects-combobox";
import { RoomsCombobox } from "@/features/rooms/components/rooms-combobox";
import { useQuery } from "@tanstack/react-query";
import { getClass } from "@/features/classes/services/api";

type ConflictType = "teacher_double_book" | "room_double_book" | "class_overlap";

interface TimetableConflict {
    type: ConflictType;
    resource_name?: string;
    resource_id?: string;
    existing_slot_id?: string;
}

function extractConflicts(err: ApiError): TimetableConflict[] {
    const raw = (err.errors as Record<string, unknown> | undefined)?.conflicts;
    if (!Array.isArray(raw)) return [];
    return raw
        .map((item): TimetableConflict | null => {
            if (typeof item === "string") {
                return { type: item as ConflictType };
            }
            if (typeof item === "object" && item !== null && "type" in item) {
                const obj = item as {
                    type: string;
                    resource_name?: string;
                    resource_id?: string;
                    existing_slot_id?: string;
                };
                return {
                    type: obj.type as ConflictType,
                    resource_name: obj.resource_name,
                    resource_id: obj.resource_id,
                    existing_slot_id: obj.existing_slot_id,
                };
            }
            return null;
        })
        .filter((c): c is TimetableConflict => c !== null);
}

const schema = z.object({
    subject_id: z.string().min(1, "Subject is required"),
    teacher_membership_id: z.string().min(1, "Teacher is required"),
    room_id: z.string().optional(),
});

type FormValues = z.infer<typeof schema>;

interface AssignSlotFormProps {
    dayOfWeek: number;
    timeSlotId: string;
    classId: string;
    onSuccess?: () => void;
    initialSubjectId?: string;
    initialTeacherId?: string;
    initialRoomId?: string;
    assignmentId?: string;
}

export function AssignSlotForm({
    dayOfWeek,
    timeSlotId,
    classId,
    onSuccess,
    initialSubjectId,
    initialTeacherId,
    initialRoomId,
    assignmentId,
}: AssignSlotFormProps) {
    const { data: classData, isLoading: isClassLoading } = useQuery({
        queryKey: ["class", classId],
        queryFn: () => getClass(classId),
        enabled: !!classId,
    });

    const gradeId = classData?.gradeId ?? "";
    const { mutate, isPending } = useAssignSlot();

    const form = useForm<FormValues>({
        resolver: zodResolver(schema),
        defaultValues: {
            subject_id: initialSubjectId ?? "",
            teacher_membership_id: initialTeacherId ?? "",
            room_id: initialRoomId ?? "",
        },
    });

    const onSubmit = React.useCallback(
        (data: FormValues) => {
            if (assignmentId) {
                // Edit mode – use PATCH if available, fallback to delete+create via mutate for now
                mutate(
                    {
                        class_room_id: classId,
                        day_of_week: dayOfWeek,
                        time_slot_id: timeSlotId,
                        subject_id: data.subject_id,
                        teacher_membership_id: data.teacher_membership_id,
                        room_id: data.room_id || undefined,
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
                            } else if (
                                err instanceof ApiError &&
                                err.status === 409 &&
                                err.code === "timetable_conflict"
                            ) {
                                const conflicts = extractConflicts(err);
                                conflicts.forEach((c) => {
                                    if (c.type === "teacher_double_book") {
                                        form.setError("teacher_membership_id", {
                                            type: "server",
                                            message: `Teacher already booked: ${c.resource_name ?? "selected teacher"}`,
                                        });
                                    }
                                    if (c.type === "room_double_book") {
                                        form.setError("room_id", {
                                            type: "server",
                                            message: `Room already booked: ${c.resource_name ?? "selected room"}`,
                                        });
                                    }
                                    if (c.type === "class_overlap") {
                                        form.setError("root", {
                                            type: "server",
                                            message: `Class already has a slot at this time`,
                                        });
                                    }
                                });
                                toast.error(
                                    err.message ||
                                        "Timetable conflict detected. Please adjust the assignment."
                                );
                            } else {
                                toast.error(getErrorMessage(err));
                            }
                        },
                    }
                );
                return;
            }
            mutate(
                {
                    class_room_id: classId,
                    day_of_week: dayOfWeek,
                    time_slot_id: timeSlotId,
                    subject_id: data.subject_id,
                    teacher_membership_id: data.teacher_membership_id,
                    room_id: data.room_id || undefined,
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
                        } else if (
                            err instanceof ApiError &&
                            err.status === 409 &&
                            err.code === "timetable_conflict"
                        ) {
                            const conflicts = extractConflicts(err);
                            conflicts.forEach((c) => {
                                if (c.type === "teacher_double_book") {
                                    form.setError("teacher_membership_id", {
                                        type: "server",
                                        message: `Teacher already booked: ${c.resource_name ?? "selected teacher"}`,
                                    });
                                }
                                if (c.type === "room_double_book") {
                                    form.setError("room_id", {
                                        type: "server",
                                        message: `Room already booked: ${c.resource_name ?? "selected room"}`,
                                    });
                                }
                                if (c.type === "class_overlap") {
                                    form.setError("root", {
                                        type: "server",
                                        message: `Class already has a slot at this time`,
                                    });
                                }
                            });
                            toast.error(
                                err.message ||
                                    "Timetable conflict detected. Please adjust the assignment."
                            );
                        } else {
                            toast.error(getErrorMessage(err));
                        }
                    },
                }
            );
        },
        [mutate, dayOfWeek, timeSlotId, onSuccess, form, classId, assignmentId]
    );

    const isDisabled = isPending || !classId || isClassLoading;

    return (
        <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-5">
                {form.formState.errors.root && (
                    <div className="text-destructive text-sm">
                        {form.formState.errors.root.message}
                    </div>
                )}
                <FormField
                    control={form.control}
                    name="subject_id"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>Subject</FormLabel>
                            <FormControl>
                                <SubjectsCombobox
                                    gradeId={gradeId}
                                    value={field.value}
                                    onChange={field.onChange}
                                    disabled={isDisabled}
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
                                    disabled={isDisabled}
                                    placeholder="Select teacher"
                                />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />

                <FormField
                    control={form.control}
                    name="room_id"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>
                                Room <span className="text-muted-foreground text-xs">optional</span>
                            </FormLabel>
                            <FormControl>
                                <RoomsCombobox
                                    value={field.value}
                                    onChange={field.onChange}
                                    disabled={isDisabled}
                                    placeholder="Select room"
                                />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />

                <Button type="submit" className="w-full" disabled={isDisabled}>
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
