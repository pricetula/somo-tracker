"use client";

import { Button } from "@/components/ui/button";
import { PlusIcon, MinusIcon } from "lucide-react";
import { FormField, FormItem, FormLabel, FormControl, FormMessage } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { type UseFormReturn } from "react-hook-form";
import type { CreateTimetableFormData } from "./timetable-create";

export interface BlocksStepProps {
    form: UseFormReturn<CreateTimetableFormData>;
    onBack: () => void;
    onSubmit: () => void;
    isPending: boolean;
}

function TimeBlockRow({
    index,
    form,
    removeBlock,
}: {
    index: number;
    form: UseFormReturn<CreateTimetableFormData>;
    removeBlock: (index: number) => void;
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

export function TimetableCreateBlocks({ form, onBack, onSubmit, isPending }: BlocksStepProps) {
    const addBlock = () => {
        const currentBlocks = form.getValues("blocks");
        const nextIndex = currentBlocks.length + 1;
        form.setValue("blocks", [
            ...currentBlocks,
            {
                day_of_week: 1,
                period_name: `Lesson ${nextIndex}`,
                start_time: "08:00",
                end_time: "09:00",
                is_break: false,
            },
        ]);
    };

    const removeBlock = (index: number) => {
        const currentBlocks = form.getValues("blocks");
        if (currentBlocks.length <= 1) return;
        form.setValue(
            "blocks",
            currentBlocks.filter((_, i) => i !== index)
        );
    };

    return (
        <div className="space-y-4">
            <div className="flex items-center justify-between">
                <h3 className="font-medium">Time Blocks</h3>
                <Button type="button" variant="outline" size="sm" onClick={addBlock}>
                    <PlusIcon className="mr-1.5 size-3.5" />
                    Add Block
                </Button>
            </div>

            <div className="space-y-3">
                {form.watch("blocks").map((_, index) => (
                    <TimeBlockRow key={index} index={index} form={form} removeBlock={removeBlock} />
                ))}
            </div>

            <div className="flex items-center gap-2 pt-2">
                <Button type="button" variant="outline" onClick={onBack}>
                    Back
                </Button>
                <Button type="button" onClick={onSubmit} disabled={isPending}>
                    {isPending ? "Creating..." : "Create Timetable"}
                </Button>
            </div>
        </div>
    );
}
