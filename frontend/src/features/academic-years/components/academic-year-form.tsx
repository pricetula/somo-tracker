/**
 * AcademicYearForm — create or edit an academic year.
 *
 * Uses react-hook-form with zod validation.
 * In edit mode, pre-fills the form with the existing year data.
 */

"use client";

import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";

import {
    Form,
    FormControl,
    FormField,
    FormItem,
    FormLabel,
    FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { getErrorMessage } from "@/lib/errors";
import { useCreateAcademicYear, useUpdateAcademicYear } from "../hooks/use-academic-years";
import type { AcademicYear } from "../types";

// ─── Schema ───────────────────────────────────────────────────────────────

const academicYearSchema = z
    .object({
        name: z.string().min(1, "Name is required").max(150, "Name must be 150 characters or less"),
        start_date: z.string().regex(/^\d{4}-\d{2}-\d{2}$/, "Must be YYYY-MM-DD format"),
        end_date: z.string().regex(/^\d{4}-\d{2}-\d{2}$/, "Must be YYYY-MM-DD format"),
    })
    .refine(
        (data) => {
            if (!data.start_date || !data.end_date) return true;
            return data.end_date >= data.start_date;
        },
        {
            message: "End date must be on or after start date",
            path: ["end_date"],
        }
    );

type AcademicYearFormValues = z.infer<typeof academicYearSchema>;

// ─── Props ────────────────────────────────────────────────────────────────

interface AcademicYearFormProps {
    /** If provided, the form operates in edit mode. */
    year?: AcademicYear;
}

// ─── Component ────────────────────────────────────────────────────────────

export function AcademicYearForm({ year }: AcademicYearFormProps) {
    const createMutation = useCreateAcademicYear();
    const updateMutation = useUpdateAcademicYear();
    const isEdit = !!year;

    const form = useForm<AcademicYearFormValues>({
        resolver: zodResolver(academicYearSchema),
        defaultValues: {
            name: year?.name ?? "",
            start_date: year?.start_date ?? "",
            end_date: year?.end_date ?? "",
        },
    });

    async function onSubmit(values: AcademicYearFormValues) {
        if (isEdit && year) {
            updateMutation.mutate({
                id: year.id,
                payload: {
                    name: values.name !== year.name ? values.name : undefined,
                    start_date:
                        values.start_date !== year.start_date ? values.start_date : undefined,
                    end_date: values.end_date !== year.end_date ? values.end_date : undefined,
                    version: year.version,
                },
            });
        } else {
            createMutation.mutate(values);
        }
    }

    const isPending = createMutation.isPending || updateMutation.isPending;
    const error = createMutation.error ?? updateMutation.error;

    return (
        <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
                {error && <p className="text-destructive">{getErrorMessage(error)}</p>}

                <FormField
                    control={form.control}
                    name="name"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>Year Name</FormLabel>
                            <FormControl>
                                <Input placeholder="e.g. 2025" {...field} />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />

                <div className="flex gap-4">
                    <FormField
                        control={form.control}
                        name="start_date"
                        render={({ field }) => (
                            <FormItem className="flex-1">
                                <FormLabel>Start Date</FormLabel>
                                <FormControl>
                                    <Input type="date" {...field} />
                                </FormControl>
                                <FormMessage />
                            </FormItem>
                        )}
                    />

                    <FormField
                        control={form.control}
                        name="end_date"
                        render={({ field }) => (
                            <FormItem className="flex-1">
                                <FormLabel>End Date</FormLabel>
                                <FormControl>
                                    <Input type="date" {...field} />
                                </FormControl>
                                <FormMessage />
                            </FormItem>
                        )}
                    />
                </div>

                <div className="flex gap-3">
                    <Button type="submit" disabled={isPending}>
                        {isPending ? "Saving…" : isEdit ? "Update Year" : "Create Year"}
                    </Button>
                </div>
            </form>
        </Form>
    );
}
