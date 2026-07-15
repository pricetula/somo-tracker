/**
 * TermForm — create or edit an academic term.
 *
 * Renders inside a dialog on the year detail page.
 * In create mode requires academic_year_id; in edit mode pre-fills from the term.
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
import { useCreateTerm, useUpdateTerm } from "../hooks/use-academic-years";
import type { AcademicTerm } from "../types";

// ─── Schema ───────────────────────────────────────────────────────────────

const termSchema = z
    .object({
        name: z.string().min(1, "Name is required"),
        term_number: z.coerce
            .number()
            .int()
            .min(1, "Term number must be at least 1")
            .max(3, "Term number cannot exceed 3"),
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

type TermFormValues = z.infer<typeof termSchema>;

// ─── Props ────────────────────────────────────────────────────────────────

interface TermFormProps {
    academicYearId: string;
    /** If provided, the form operates in edit mode. */
    term?: AcademicTerm;
    /** Called after a successful mutation to close the dialog. */
    onSuccess?: () => void;
}

// ─── Component ────────────────────────────────────────────────────────────

export function TermForm({ academicYearId, term, onSuccess }: TermFormProps) {
    const createMutation = useCreateTerm();
    const updateMutation = useUpdateTerm();
    const isEdit = !!term;

    const form = useForm<TermFormValues>({
        resolver: zodResolver(termSchema),
        defaultValues: {
            name: term?.name ?? "",
            term_number: term?.term_number ?? 1,
            start_date: term?.start_date ?? "",
            end_date: term?.end_date ?? "",
        },
    });

    async function onSubmit(values: TermFormValues) {
        if (isEdit && term) {
            updateMutation.mutate(
                {
                    id: term.id,
                    payload: {
                        name: values.name !== term.name ? values.name : undefined,
                        start_date:
                            values.start_date !== term.start_date ? values.start_date : undefined,
                        end_date: values.end_date !== term.end_date ? values.end_date : undefined,
                        version: term.version,
                    },
                },
                { onSuccess: () => onSuccess?.() }
            );
        } else {
            createMutation.mutate(
                { ...values, academic_year_id: academicYearId },
                { onSuccess: () => onSuccess?.() }
            );
        }
    }

    const isPending = createMutation.isPending || updateMutation.isPending;
    const error = createMutation.error ?? updateMutation.error;

    return (
        <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
                {error && <p className="text-destructive text-sm">{getErrorMessage(error)}</p>}

                <FormField
                    control={form.control}
                    name="name"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>Term Name</FormLabel>
                            <FormControl>
                                <Input placeholder="e.g. Term 1" {...field} />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />

                <FormField
                    control={form.control}
                    name="term_number"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>Term Number</FormLabel>
                            <FormControl>
                                <Input type="number" min={1} max={3} placeholder="1" {...field} />
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

                <div className="flex justify-end gap-3 pt-2">
                    <Button type="submit" disabled={isPending}>
                        {isPending ? "Saving…" : isEdit ? "Update Term" : "Create Term"}
                    </Button>
                </div>
            </form>
        </Form>
    );
}
