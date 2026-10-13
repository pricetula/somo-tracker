"use client";

import { useMemo } from "react";
// import { endOfYear, startOfYear } from "date-fns";
import { Trash2 } from "lucide-react";
import { useForm, useFieldArray, useWatch } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
    Form,
    FormControl,
    FormField,
    FormItem,
    FormLabel,
    FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { DatePicker } from "@/components/ui/date-picker";
import { useCreateAcademicPeriod } from "@/features/school/hooks/use-academic-period";

const currentDate = new Date();
const maxYear = currentDate.getFullYear();
const minYear = maxYear - 10;

const termSchema = z.object({
    name: z.string().min(1, "Term name is required"),
    start_date: z.string().min(1, "Start date is required"),
    end_date: z.string().min(1, "End date is required"),
});

const schema = z.object({
    year: z.coerce
        .number()
        .int()
        .min(minYear, "Year must be at least " + minYear)
        .max(maxYear, "Year must be at most " + maxYear),
    terms: z.array(termSchema).min(1, "At least one term is required"),
});

type FormValues = z.infer<typeof schema>;

export function AcademicPeriodAddForm() {
    const router = useRouter();
    const mutation = useCreateAcademicPeriod();

    const form = useForm<FormValues>({
        resolver: zodResolver(schema),
        defaultValues: {
            year: maxYear,
            terms: [{ name: "", start_date: "", end_date: "" }],
        },
    });

    const { fields, append, remove } = useFieldArray({
        control: form.control,
        name: "terms",
    });

    const selectedYear = useWatch({
        control: form.control,
        name: "year",
    });

    // TODO: Add ability to disable min and max dates
    // const { minDate, maxDate } = useMemo(() => {
    //     if (selectedYear && selectedYear <= maxYear && selectedYear >= minYear) {
    //         const selectedDate = new Date(selectedYear, 0, 1);
    //         return { minDate: startOfYear(selectedDate), maxDate: endOfYear(selectedDate) };
    //     }
    //     return { minDate: startOfYear(currentDate), maxDate: endOfYear(currentDate) };
    // }, [selectedYear]);

    const enableDatePickers = useMemo(() => {
        return selectedYear && selectedYear <= maxYear && selectedYear >= minYear;
    }, [selectedYear]);

    const onSubmit = (values: FormValues) => {
        mutation.mutate(
            values as {
                year: number;
                terms: { name: string; start_date: string; end_date: string }[];
            },
            {
                onSuccess: () => {
                    router.back();
                },
            }
        );
    };

    return (
        <Card>
            <CardHeader>
                <CardTitle>Period Details</CardTitle>
            </CardHeader>
            <CardContent>
                <Form {...form}>
                    <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
                        <FormField
                            control={form.control}
                            name="year"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>Academic Year</FormLabel>
                                    <FormControl>
                                        <Input
                                            type="number"
                                            placeholder="e.g. 2026"
                                            {...field}
                                            onChange={(e) =>
                                                field.onChange(e.target.valueAsNumber || "")
                                            }
                                        />
                                    </FormControl>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />

                        <div className="space-y-3">
                            <div className="flex items-center justify-between">
                                <Label>Terms</Label>
                                <Button
                                    type="button"
                                    variant="outline"
                                    size="sm"
                                    onClick={() =>
                                        append({ name: "", start_date: "", end_date: "" })
                                    }
                                >
                                    Add term
                                </Button>
                            </div>

                            {fields.map((field, index) => (
                                <div
                                    key={field.id}
                                    className="flex flex-col gap-3 rounded-lg border p-3 md:flex-row md:items-end md:justify-between"
                                >
                                    <FormField
                                        control={form.control}
                                        name={`terms.${index}.name`}
                                        render={({ field }) => (
                                            <FormItem>
                                                <FormLabel className="text-xs">Term name</FormLabel>
                                                <FormControl>
                                                    <Input placeholder="Term 1" {...field} />
                                                </FormControl>
                                                <FormMessage />
                                            </FormItem>
                                        )}
                                    />
                                    <FormField
                                        control={form.control}
                                        name={`terms.${index}.start_date`}
                                        render={({ field }) => (
                                            <FormItem>
                                                <FormLabel className="text-xs">Start</FormLabel>
                                                <FormControl>
                                                    <DatePicker
                                                        value={field.value}
                                                        onChange={field.onChange}
                                                        placeholder="Pick start"
                                                        className="w-40"
                                                        disabled={!enableDatePickers}
                                                    />
                                                </FormControl>
                                                <FormMessage />
                                            </FormItem>
                                        )}
                                    />

                                    <FormField
                                        control={form.control}
                                        name={`terms.${index}.end_date`}
                                        render={({ field }) => (
                                            <FormItem>
                                                <FormLabel className="text-xs">End</FormLabel>
                                                <FormControl>
                                                    <DatePicker
                                                        value={field.value}
                                                        onChange={field.onChange}
                                                        placeholder="Pick end"
                                                        className="w-40"
                                                        disabled={!enableDatePickers}
                                                    />
                                                </FormControl>
                                                <FormMessage />
                                            </FormItem>
                                        )}
                                    />

                                    <Button
                                        type="button"
                                        variant="ghost"
                                        size="icon"
                                        onClick={() => remove(index)}
                                        aria-label="Remove term"
                                        className="col-span-1 ml-auto"
                                        disabled={fields.length === 1}
                                    >
                                        <Trash2 className="text-muted-foreground h-4 w-4" />
                                    </Button>
                                </div>
                            ))}
                        </div>

                        <div className="flex justify-end gap-2 pt-2">
                            <Button type="button" variant="outline" onClick={() => router.back()}>
                                Cancel
                            </Button>
                            <Button type="submit" disabled={mutation.isPending}>
                                {mutation.isPending ? "Creating..." : "Create Period"}
                            </Button>
                        </div>
                    </form>
                </Form>
            </CardContent>
        </Card>
    );
}
