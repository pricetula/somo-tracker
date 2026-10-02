"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import {
    Form,
    FormField,
    FormItem,
    FormLabel,
    FormControl,
    FormMessage,
} from "@/components/ui/form";
import { useCreateSubstitution, useUpdateSubstitution } from "../hooks/use-substitutions";
import type { CreateSubstitutionRequest, UpdateSubstitutionRequest } from "../types/substitution";

const schema = z.object({
    classTimetableSlotId: z.string().min(1, "Class timetable slot is required"),
    substitutionDate: z.string().min(1, "Date is required"),
    originalTeacherMembershipId: z.string().min(1, "Original teacher is required"),
    substituteTeacherMembershipId: z.string().optional(),
    status: z
        .enum(["PENDING", "ASSIGNED", "COMPLETED", "CANCELLED"])
        .default("PENDING")
        .nonoptional(),
    reason: z.string().optional(),
});

type FormValues = z.infer<typeof schema>;

interface SubstitutionFormProps {
    onSuccess: () => void;
    defaultValues?: Partial<FormValues>;
    isEditing?: boolean;
    substitutionId?: string;
}

export function SubstitutionForm({
    onSuccess,
    defaultValues,
    isEditing = false,
    substitutionId,
}: SubstitutionFormProps) {
    const createSub = useCreateSubstitution();
    const updateSub = useUpdateSubstitution();
    const isPending = isEditing ? updateSub.isPending : createSub.isPending;

    const form = useForm<FormValues>({
        resolver: zodResolver(schema),
        defaultValues: {
            classTimetableSlotId: "",
            substitutionDate: "",
            originalTeacherMembershipId: "",
            substituteTeacherMembershipId: "",
            status: "PENDING",
            reason: "",
            ...defaultValues,
        },
    });

    const onSubmit = form.handleSubmit((values) => {
        if (isEditing && substitutionId) {
            updateSub.mutate(
                { id: substitutionId, data: values as Omit<UpdateSubstitutionRequest, "id"> },
                { onSuccess }
            );
        } else {
            createSub.mutate(values as CreateSubstitutionRequest, { onSuccess });
        }
    });

    return (
        <Form {...form}>
            <form onSubmit={onSubmit} className="space-y-4">
                <FormField
                    control={form.control}
                    name="classTimetableSlotId"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>Class Timetable Slot</FormLabel>
                            <FormControl>
                                <Input {...field} placeholder="Enter class timetable slot ID" />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />
                <FormField
                    control={form.control}
                    name="substitutionDate"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>Substitution Date</FormLabel>
                            <FormControl>
                                <Input {...field} type="date" />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />
                <FormField
                    control={form.control}
                    name="originalTeacherMembershipId"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>Original Teacher</FormLabel>
                            <FormControl>
                                <Input
                                    {...field}
                                    placeholder="Enter original teacher membership ID"
                                />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />
                <FormField
                    control={form.control}
                    name="substituteTeacherMembershipId"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>Substitute Teacher</FormLabel>
                            <FormControl>
                                <Input
                                    {...field}
                                    placeholder="Enter substitute teacher membership ID (optional)"
                                />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />
                <FormField
                    control={form.control}
                    name="status"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>Status</FormLabel>
                            <FormControl>
                                <Select onValueChange={field.onChange} defaultValue={field.value}>
                                    <SelectTrigger>
                                        <SelectValue placeholder="Select status" />
                                    </SelectTrigger>
                                    <SelectContent>
                                        <SelectItem value="PENDING">Pending</SelectItem>
                                        <SelectItem value="ASSIGNED">Assigned</SelectItem>
                                        <SelectItem value="COMPLETED">Completed</SelectItem>
                                        <SelectItem value="CANCELLED">Cancelled</SelectItem>
                                    </SelectContent>
                                </Select>
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />
                <FormField
                    control={form.control}
                    name="reason"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>Reason</FormLabel>
                            <FormControl>
                                <Textarea
                                    {...field}
                                    placeholder="Reason for substitution"
                                    rows={3}
                                />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />
                <Button type="submit" disabled={isPending} className="w-full">
                    {isPending
                        ? isEditing
                            ? "Updating..."
                            : "Creating..."
                        : isEditing
                          ? "Update"
                          : "Create"}
                </Button>
            </form>
        </Form>
    );
}
