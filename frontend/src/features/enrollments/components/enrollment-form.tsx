"use client";

import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Button } from "@/components/ui/button";
import {
    Form,
    FormControl,
    FormField,
    FormItem,
    FormLabel,
    FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import type { CreateEnrollmentRequest } from "../types/enrollment";

const schema = z.object({
    student_id: z.string().min(1, "Student ID is required"),
    enrollment_date: z.string().min(1, "Enrollment date is required"),
});

type FormValues = z.infer<typeof schema>;

export function EnrollmentForm({
    onSubmit,
}: {
    onSubmit: (values: CreateEnrollmentRequest) => Promise<void>;
}) {
    const form = useForm<FormValues>({
        resolver: zodResolver(schema),
        defaultValues: {
            student_id: "",
            enrollment_date: new Date().toISOString().slice(0, 10),
        },
    });

    return (
        <Form {...form}>
            <form
                onSubmit={form.handleSubmit(async (values) => {
                    await onSubmit(values as CreateEnrollmentRequest);
                    form.reset();
                })}
                className="space-y-4"
            >
                <FormField
                    control={form.control}
                    name="student_id"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>Student ID</FormLabel>
                            <FormControl>
                                <Input placeholder="Student ID" {...field} />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />
                <FormField
                    control={form.control}
                    name="enrollment_date"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>Enrollment Date</FormLabel>
                            <FormControl>
                                <Input type="date" {...field} />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />
                <Button type="submit" className="w-full">
                    Enroll
                </Button>
            </form>
        </Form>
    );
}
