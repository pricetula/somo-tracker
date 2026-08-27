"use client";

import React from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { toast } from "sonner";
import { PlusIcon, MinusIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
    Form,
    FormField,
    FormItem,
    FormLabel,
    FormControl,
    FormMessage,
} from "@/components/ui/form";
import { useCreateBlocks } from "@/features/timetable/hooks";

const BlockSchema = z.object({
    period_name: z.string().min(1, "Period name is required"),
    start_time: z.string().regex(/^([0-1]?[0-9]|2[0-3]):[0-5][0-9]$/, "Invalid time format"),
    end_time: z.string().regex(/^([0-1]?[0-9]|2[0-3]):[0-5][0-9]$/, "Invalid time format"),
    is_break: z.boolean(),
});

const Schema = z.object({ blocks: z.array(BlockSchema).min(1) });

type FormData = z.infer<typeof Schema>;

const parseMin = (t: string): number => {
    const [h, m] = t.split(":").map(Number);
    return h * 60 + m;
};
const fmtMin = (m: number): string => {
    const h = Math.floor(m / 60)
        .toString()
        .padStart(2, "0");
    const mm = (m % 60).toString().padStart(2, "0");
    return `${h}:${mm}`;
};

function TimeBlockRow({
    index,
    form,
    removeBlock,
}: {
    index: number;
    form: ReturnType<typeof useForm<FormData>>;
    removeBlock: (i: number) => void;
}) {
    return (
        <div className="bg-muted/30 space-y-3 rounded-md p-4">
            <div className="flex items-center justify-between">
                <span className="text-sm font-medium">Block {index + 1}</span>
                <Button
                    type="button"
                    variant="ghost"
                    size="icon-sm"
                    onClick={() => removeBlock(index)}
                    aria-label="Remove block"
                >
                    <MinusIcon className="size-4" />
                </Button>
            </div>

            <div className="grid gap-3 sm:grid-cols-3">
                <FormField
                    control={form.control}
                    name={`blocks.${index}.period_name`}
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>Period Name</FormLabel>
                            <FormControl>
                                <Input {...field} placeholder="e.g., Lesson 1" />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />
                <FormField
                    control={form.control}
                    name={`blocks.${index}.start_time`}
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>Start Time</FormLabel>
                            <FormControl>
                                <Input {...field} type="time" />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />
                <FormField
                    control={form.control}
                    name={`blocks.${index}.end_time`}
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>End Time</FormLabel>
                            <FormControl>
                                <Input {...field} type="time" />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />
            </div>

            <FormField
                control={form.control}
                name={`blocks.${index}.is_break`}
                render={({ field }) => (
                    <FormItem className="flex items-center gap-2">
                        <FormControl>
                            <input
                                type="checkbox"
                                checked={field.value}
                                onChange={(e) => field.onChange(e.target.checked)}
                                className="border-input size-4 rounded"
                            />
                        </FormControl>
                        <FormLabel className="mb-0 cursor-pointer">Break period</FormLabel>
                    </FormItem>
                )}
            />
        </div>
    );
}

export function BlockCreateForm({
    trackId,
    onSuccess,
}: {
    trackId: string;
    onSuccess?: () => void;
}) {
    const { mutate, isPending, isError, error, isSuccess } = useCreateBlocks();

    const form = useForm<FormData>({
        resolver: zodResolver(Schema),
        defaultValues: {
            blocks: [
                {
                    period_name: "Lesson 1",
                    start_time: "08:00",
                    end_time: "09:00",
                    is_break: false,
                },
            ],
        },
    });

    React.useEffect(() => {
        if (isSuccess && onSuccess) {
            onSuccess();
        }
    }, [isSuccess, onSuccess]);

    React.useEffect(() => {
        if (isError && error) toast.error(error.message);
    }, [isError, error]);

    const addBlock = React.useCallback(() => {
        const current = form.getValues("blocks");
        const first = current[0];
        const last = current[current.length - 1];

        const durationMin = parseMin(first.end_time) - parseMin(first.start_time);
        const newStartMin = parseMin(last.end_time);
        const newEndMin = newStartMin + durationMin;

        const nextIndex = current.length + 1;
        form.setValue("blocks", [
            ...current,
            {
                period_name: `Lesson ${nextIndex}`,
                start_time: fmtMin(newStartMin),
                end_time: fmtMin(newEndMin),
                is_break: false,
            },
        ]);
    }, [form]);

    const removeBlock = React.useCallback(
        (i: number) => {
            const current = form.getValues("blocks");
            if (current.length <= 1) return;
            form.setValue(
                "blocks",
                current.filter((_, idx) => idx !== i)
            );
        },
        [form]
    );

    const onSubmit = React.useCallback(
        (data: FormData) => {
            if (!trackId) {
                toast.error("No track context");
                return;
            }
            const payloads = data.blocks.map((b, i) => ({
                track_id: trackId,
                day_of_week: 1,
                period_name: b.period_name,
                start_time: b.start_time,
                end_time: b.end_time,
                is_break: b.is_break,
                order: i + 1,
            }));
            mutate(payloads);
        },
        [mutate, trackId]
    );

    return (
        <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
                <div className="flex items-center justify-between">
                    <h3 className="font-medium">Time Blocks</h3>
                    <Button type="button" variant="outline" size="sm" onClick={addBlock}>
                        <PlusIcon className="mr-1.5 size-3.5" />
                        Add Block
                    </Button>
                </div>

                <div className="space-y-3">
                    {form.watch("blocks").map((_, i) => (
                        <TimeBlockRow key={i} index={i} form={form} removeBlock={removeBlock} />
                    ))}
                </div>

                <div className="flex justify-end pt-2">
                    <Button type="submit" disabled={isPending}>
                        {isPending ? "Saving..." : "Save Blocks"}
                    </Button>
                </div>
            </form>
        </Form>
    );
}
