/**
 * Student Sheet — shadcn Sheet (right-drawer) for manually adding a student.
 *
 * Preserves underlying view state. On save, invalidates query cache
 * and closes smoothly.
 */

"use client";

import * as React from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";

import {
    Sheet,
    SheetContent,
    SheetHeader,
    SheetTitle,
    SheetDescription,
    SheetClose,
} from "@/components/ui/sheet";
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
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";

import { useCreateStudent } from "@/features/students/hooks/use-students";

// ─── Schema ────────────────────────────────────────────────────────────────

const studentSchema = z.object({
    first_name: z.string().min(1, "First name is required"),
    middle_name: z.string().optional(),
    last_name: z.string().min(1, "Last name is required"),
    gender: z.enum(["MALE", "FEMALE", "OTHER", "PREFER_NOT_TO_SAY"], {
        required_error: "Gender is required",
    }),
    date_of_birth: z
        .string()
        .min(1, "Date of birth is required")
        .regex(/^\d{4}-\d{2}-\d{2}$/, "Use YYYY-MM-DD format"),
});

type StudentFormValues = z.infer<typeof studentSchema>;

// ─── Props ─────────────────────────────────────────────────────────────────

interface StudentSheetProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function StudentSheet({ open, onOpenChange }: StudentSheetProps) {
    const createStudent = useCreateStudent();

    const form = useForm<StudentFormValues>({
        resolver: zodResolver(studentSchema),
        defaultValues: {
            first_name: "",
            middle_name: "",
            last_name: "",
            gender: undefined,
            date_of_birth: "",
        },
    });

    async function onSubmit(values: StudentFormValues) {
        try {
            await createStudent.mutateAsync({
                ...values,
                middle_name: values.middle_name || undefined,
            });
            form.reset();
            onOpenChange(false);
        } catch {
            // Error toast is handled by the mutation
        }
    }

    return (
        <Sheet open={open} onOpenChange={onOpenChange}>
            <SheetContent side="right" className="w-full sm:max-w-md">
                <SheetHeader>
                    <SheetTitle>Add Student</SheetTitle>
                    <SheetDescription>Fill in the student&apos;s details below.</SheetDescription>
                </SheetHeader>

                <Form {...form}>
                    <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4 p-6">
                        <FormField
                            control={form.control}
                            name="first_name"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>First Name</FormLabel>
                                    <FormControl>
                                        <Input placeholder="e.g. Jane" {...field} />
                                    </FormControl>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />

                        <FormField
                            control={form.control}
                            name="middle_name"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>Middle Name</FormLabel>
                                    <FormControl>
                                        <Input placeholder="Optional" {...field} />
                                    </FormControl>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />

                        <FormField
                            control={form.control}
                            name="last_name"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>Last Name</FormLabel>
                                    <FormControl>
                                        <Input placeholder="e.g. Doe" {...field} />
                                    </FormControl>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />

                        <FormField
                            control={form.control}
                            name="gender"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>Gender</FormLabel>
                                    <Select
                                        onValueChange={field.onChange}
                                        value={field.value ?? ""}
                                    >
                                        <FormControl>
                                            <SelectTrigger>
                                                <SelectValue placeholder="Select gender" />
                                            </SelectTrigger>
                                        </FormControl>
                                        <SelectContent>
                                            <SelectItem value="MALE">Male</SelectItem>
                                            <SelectItem value="FEMALE">Female</SelectItem>
                                            <SelectItem value="OTHER">Other</SelectItem>
                                            <SelectItem value="PREFER_NOT_TO_SAY">
                                                Prefer not to say
                                            </SelectItem>
                                        </SelectContent>
                                    </Select>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />

                        <FormField
                            control={form.control}
                            name="date_of_birth"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>Date of Birth</FormLabel>
                                    <FormControl>
                                        <Input type="date" placeholder="YYYY-MM-DD" {...field} />
                                    </FormControl>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />

                        <div className="flex justify-end gap-3 pt-2">
                            <SheetClose asChild>
                                <Button variant="outline" type="button">
                                    Cancel
                                </Button>
                            </SheetClose>
                            <Button type="submit" disabled={createStudent.isPending}>
                                {createStudent.isPending ? "Saving..." : "Save"}
                            </Button>
                        </div>
                    </form>
                </Form>
            </SheetContent>
        </Sheet>
    );
}
