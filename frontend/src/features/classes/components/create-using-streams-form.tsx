"use client";

import { useMemo, useEffect } from "react";
import { useForm, useFieldArray } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Form, FormControl, FormField, FormItem } from "@/components/ui/form";
import { X } from "lucide-react";
import { useStreams } from "@/features/streams";
import { useGrades } from "@/features/grades";
import { createClass } from "../services/api";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { toast } from "sonner";

const itemSchema = z.object({
    name: z.string().min(1, "Name is required"),
    gradeId: z.string().min(1, "Grade is required"),
    streamId: z.string().min(1, "Stream is required"),
    gradeLabel: z.string(),
    streamLabel: z.string(),
});

const formSchema = z.object({
    items: z.array(itemSchema).min(1, "At least one class is required"),
});

type FormValues = z.infer<typeof formSchema>;

export function CreateUsingStreamsForm() {
    const router = useRouter();
    const { data: streams, isLoading: streamsLoading } = useStreams();
    const { data: grades, isLoading: gradesLoading } = useGrades();

    // Derive default list from all grades × all streams
    const defaultItems = useMemo(() => {
        if (!grades || !streams || grades.length === 0 || streams.length === 0) return [];
        const list: FormValues["items"] = [];
        for (const grade of grades) {
            for (const stream of streams) {
                list.push({
                    name: `${grade.local_label} ${stream.name}`,
                    gradeId: grade.id,
                    streamId: stream.id,
                    gradeLabel: grade.local_label,
                    streamLabel: stream.name,
                });
            }
        }
        return list;
    }, [grades, streams]);

    const form = useForm<FormValues>({
        resolver: zodResolver(formSchema),
        defaultValues: { items: defaultItems },
    });

    // Reset form when data loads / changes so the derived list updates
    const { fields, remove } = useFieldArray({
        control: form.control,
        name: "items",
    });

    // Sync default items when streams/grades load or change
    useEffect(() => {
        if (defaultItems.length > 0) {
            form.reset({ items: defaultItems });
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [defaultItems]);

    const mutation = useApiMutation(
        async (items: FormValues["items"]) => {
            // Create sequentially to avoid overwhelming the API; could parallelize if preferred
            for (const item of items) {
                await createClass({
                    name: item.name,
                    gradeId: item.gradeId,
                    streamId: item.streamId,
                });
            }
        },
        {
            onSuccess: () => {
                toast.success("Classes created successfully");
                router.back();
            },
        }
    );

    const onSubmit = (values: FormValues) => {
        mutation.mutate(values.items);
    };

    const isLoading = streamsLoading || gradesLoading;

    if (isLoading) {
        return <div className="text-muted-foreground text-sm">Loading streams and grades…</div>;
    }

    if (!streams || streams.length === 0 || !grades || grades.length === 0) {
        return (
            <div className="space-y-2">
                <p className="text-muted-foreground text-sm">No streams or grades available.</p>
                <Button variant="outline" onClick={() => router.back()}>
                    Go back
                </Button>
            </div>
        );
    }

    return (
        <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
                <div className="space-y-2">
                    <h2 className="text-lg font-semibold">Available streams</h2>
                    <div className="flex flex-wrap gap-2">
                        {streams.map((s) => (
                            <span
                                key={s.id}
                                className="inline-flex items-center gap-2 rounded-full border px-3 py-1 text-xs"
                            >
                                <span
                                    className="h-2.5 w-2.5 rounded-full"
                                    style={{ backgroundColor: s.color || "#9ca3af" }}
                                />
                                {s.name}
                            </span>
                        ))}
                    </div>
                </div>

                <div className="space-y-2">
                    <h2 className="text-lg font-semibold">Classes to create</h2>
                    <p className="text-muted-foreground text-sm">
                        {fields.length} class(es) derived from {grades.length} grade level(s) ×{" "}
                        {streams.length} stream(s). Edit names before creating.
                    </p>

                    <div className="max-h-60 overflow-y-auto rounded-md border">
                        <table className="w-full text-sm">
                            <thead className="bg-muted text-muted-foreground sticky top-0 z-10 text-xs tracking-wide uppercase">
                                <tr>
                                    <th className="px-3 py-2 text-left font-medium">
                                        Grade · Stream
                                    </th>
                                    <th className="px-3 py-2 text-left font-medium">Class name</th>
                                    <th className="w-10 px-3 py-2"></th>
                                </tr>
                            </thead>
                            <tbody>
                                {fields.map((field, index) => (
                                    <tr key={field.id} className="border-t">
                                        <td className="text-muted-foreground px-3 py-2 text-xs whitespace-nowrap">
                                            {fields[index].gradeLabel} · {fields[index].streamLabel}
                                        </td>
                                        <td className="px-3 py-2">
                                            <FormField
                                                control={form.control}
                                                name={`items.${index}.name`}
                                                render={({ field }) => (
                                                    <FormItem className="space-y-0">
                                                        <FormControl>
                                                            <Input
                                                                {...field}
                                                                className="h-7 border-0 bg-transparent text-sm shadow-none focus-visible:ring-1"
                                                                disabled={mutation.isPending}
                                                            />
                                                        </FormControl>
                                                    </FormItem>
                                                )}
                                            />
                                        </td>
                                        <td className="px-3 py-2 text-right">
                                            <Button
                                                type="button"
                                                variant="ghost"
                                                size="icon"
                                                className="text-muted-foreground hover:text-destructive h-6 w-6"
                                                onClick={() => remove(index)}
                                                disabled={mutation.isPending}
                                                aria-label="Remove"
                                            >
                                                <X className="h-3 w-3" />
                                            </Button>
                                        </td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>
                </div>

                <div className="flex items-center justify-between pt-2">
                    <Button
                        type="button"
                        variant="outline"
                        onClick={() => {
                            if (defaultItems.length > 0) {
                                form.reset({ items: defaultItems });
                            }
                        }}
                        disabled={mutation.isPending}
                    >
                        Reset to defaults
                    </Button>
                    <div className="flex gap-2">
                        <Button
                            type="button"
                            variant="outline"
                            onClick={() => router.back()}
                            disabled={mutation.isPending}
                        >
                            Cancel
                        </Button>
                        <Button
                            type="submit"
                            disabled={mutation.isPending || form.formState.isSubmitting}
                        >
                            {mutation.isPending ? "Creating…" : "Create all classes"}
                        </Button>
                    </div>
                </div>
            </form>
        </Form>
    );
}
