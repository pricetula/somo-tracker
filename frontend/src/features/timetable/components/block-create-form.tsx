"use client";

import React from "react";
import { useRouter, usePathname } from "next/navigation";
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
    period_name: z.string().min(1),
    start_time: z.string(),
    end_time: z.string(),
    is_break: z.boolean(),
});

const Schema = z.object({ blocks: z.array(BlockSchema).min(1) });

type Data = z.infer<typeof Schema>;

export function BlockCreateForm({ trackId }: { trackId: string }) {
    const router = useRouter();
    const { mutate, isPending } = useCreateBlocks();

    const form = useForm<Data>({
        resolver: zodResolver(Schema),
        defaultValues: {
            blocks: [{ period_name: "", start_time: "08:00", end_time: "09:00", is_break: false }],
        },
    });

    const add = () => {
        const c = form.getValues("blocks");
        form.setValue("blocks", [
            ...c,
            { period_name: "", start_time: "", end_time: "", is_break: false },
        ]);
    };
    const remove = (i: number) => {
        const c = form.getValues("blocks");
        if (c.length <= 1) return;
        form.setValue(
            "blocks",
            c.filter((_, idx) => idx !== i)
        );
    };

    const onSubmit = (data: Data) => {
        if (!trackId) {
            toast.error("No track");
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
        mutate(payloads, { onSuccess: () => router.push(`/timetable/${trackId}`) });
    };

    return (
        <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
                {form.watch("blocks").map((_, i) => (
                    <div key={i} className="bg-muted/30 space-y-3 rounded-md p-4">
                        <div className="flex items-center justify-between">
                            <span className="text-sm font-medium">Block {i + 1}</span>
                            <Button
                                type="button"
                                variant="ghost"
                                size="icon-sm"
                                onClick={() => remove(i)}
                                aria-label="Remove block"
                            >
                                <MinusIcon className="size-4" />
                            </Button>
                        </div>
                        <div className="grid gap-3 sm:grid-cols-3">
                            <FormField
                                control={form.control}
                                name={`blocks.${i}.period_name`}
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
                                name={`blocks.${i}.start_time`}
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
                                name={`blocks.${i}.end_time`}
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
                            name={`blocks.${i}.is_break`}
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
                                    <FormLabel className="mb-0 cursor-pointer">
                                        Break period
                                    </FormLabel>
                                </FormItem>
                            )}
                        />
                    </div>
                ))}
                <div className="flex items-center gap-2 pt-2">
                    <Button type="button" variant="outline" size="sm" onClick={add}>
                        <PlusIcon className="mr-1.5 size-3.5" /> Add Block
                    </Button>
                    <Button type="submit" size="sm" disabled={isPending}>
                        {isPending ? "Saving..." : "Save Blocks"}
                    </Button>
                </div>
            </form>
        </Form>
    );
}
