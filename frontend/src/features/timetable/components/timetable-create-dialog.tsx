"use client";

import React from "react";
import { useForm, type UseFormReturn } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useRouter } from "next/navigation";
import { useCreateTrack } from "../hooks";
import {
    Form,
    FormField,
    FormItem,
    FormLabel,
    FormControl,
    FormDescription,
    FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { Button } from "@/components/ui/button";
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
    DialogFooter,
} from "@/components/ui/dialog";
import { PlusIcon, MinusIcon } from "lucide-react";

export const TimeBlockSchema = z.object({
    day_of_week: z.number().int().min(1).max(7),
    period_name: z.string().min(1, "Period name is required"),
    start_time: z
        .string()
        .regex(/^([0-1]?[0-9]|2[0-3]):[0-5][0-9]$/, "Invalid time format (HH:MM)"),
    end_time: z.string().regex(/^([0-1]?[0-9]|2[0-3]):[0-5][0-9]$/, "Invalid time format (HH:MM)"),
    is_break: z.boolean(),
    order: z.string(),
});

export const CreateTimetableSchema = z.object({
    name: z.string().min(1, "Timetable name is required").max(100),
    description: z.string().max(500).optional(),
    blocks: z.array(TimeBlockSchema).min(1, "At least one time block is required"),
});

export type CreateTimetableFormData = z.infer<typeof CreateTimetableSchema>;
export type TimeBlockFormData = z.infer<typeof TimeBlockSchema>;

const DAY_OPTIONS = [
    { value: 1, label: "Monday" },
    { value: 2, label: "Tuesday" },
    { value: 3, label: "Wednesday" },
    { value: 4, label: "Thursday" },
    { value: 5, label: "Friday" },
    { value: 6, label: "Saturday" },
    { value: 7, label: "Sunday" },
];

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

            <div className="grid gap-3 sm:grid-cols-2">
                <FormField
                    control={form.control}
                    name={`blocks.${index}.day_of_week`}
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>Day</FormLabel>
                            <FormControl>
                                <Select onValueChange={field.onChange} defaultValue={field.value}>
                                    <SelectTrigger>
                                        <SelectValue placeholder="Select day" />
                                    </SelectTrigger>
                                    <SelectContent>
                                        {DAY_OPTIONS.map((day) => (
                                            <SelectItem key={day.value} value={day.value}>
                                                {day.label}
                                            </SelectItem>
                                        ))}
                                    </SelectContent>
                                </Select>
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />

                <FormField
                    control={form.control}
                    name={`blocks.${index}.period_name`}
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>Period Name</FormLabel>
                            <FormControl>
                                <Input {...field} placeholder="e.g., Period 1, Morning Assembly" />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />
            </div>

            <div className="grid gap-3 sm:grid-cols-3">
                <FormField
                    control={form.control}
                    name={`blocks.${index}.start_time`}
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>Start Time</FormLabel>
                            <FormControl>
                                <Input {...field} type="time" placeholder="HH:MM" />
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
                                <Input {...field} type="time" placeholder="HH:MM" />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />

                <FormField
                    control={form.control}
                    name={`blocks.${index}.order`}
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>Order</FormLabel>
                            <FormControl>
                                <Input {...field} type="number" min="0" placeholder="0" />
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

export function CreateTimetableDialog() {
    const router = useRouter();
    const createTrack = useCreateTrack();

    const form = useForm<CreateTimetableFormData>({
        resolver: zodResolver(CreateTimetableSchema),
        defaultValues: {
            name: "",
            description: "",
            blocks: [
                {
                    day_of_week: 1,
                    period_name: "",
                    start_time: "",
                    end_time: "",
                    is_break: false,
                    order: 0,
                },
            ],
        },
    });

    const addBlock = () => {
        const currentBlocks = form.getValues("blocks");
        form.setValue("blocks", [
            ...currentBlocks,
            {
                day_of_week: 1,
                period_name: "",
                start_time: "",
                end_time: "",
                is_break: false,
                order: currentBlocks.length,
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

    const handleRouteBack = React.useCallback(() => {
        router.back();
    }, [router]);

    const onSubmit = async (data: CreateTimetableFormData) => {
        try {
            const payload = {
                name: data.name,
                description: data.description,
                initial_blocks: data.blocks.map((block) => ({
                    track_id: "",
                    day_of_week: block.day_of_week,
                    period_name: block.period_name,
                    start_time: block.start_time,
                    end_time: block.end_time,
                    is_break: block.is_break,
                    order: block.order,
                })),
            };

            await createTrack.mutateAsync(payload);
            handleRouteBack();
        } catch (error) {
            console.error("Failed to create timetable:", error);
        }
    };

    return (
        <Dialog open onOpenChange={handleRouteBack}>
            <DialogContent className="max-h-[90vh] w-full overflow-y-auto sm:max-w-2xl">
                <DialogHeader>
                    <DialogTitle>Create Timetable</DialogTitle>
                    <DialogDescription>
                        Set up a new timetable track with time blocks for each day.
                    </DialogDescription>
                </DialogHeader>

                <Form {...form}>
                    <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
                        <FormField
                            control={form.control}
                            name="name"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>Timetable Name</FormLabel>
                                    <FormControl>
                                        <Input {...field} placeholder="e.g., 2024 Academic Year" />
                                    </FormControl>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />

                        <FormField
                            control={form.control}
                            name="description"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>Description (Optional)</FormLabel>
                                    <FormControl>
                                        <Textarea
                                            {...field}
                                            placeholder="Brief description of this timetable"
                                            rows={3}
                                        />
                                    </FormControl>
                                    <FormDescription>Maximum 500 characters.</FormDescription>
                                </FormItem>
                            )}
                        />

                        <div className="space-y-4">
                            <div className="flex items-center justify-between">
                                <h3 className="font-medium">Time Blocks</h3>
                                <Button
                                    type="button"
                                    variant="outline"
                                    size="sm"
                                    onClick={addBlock}
                                >
                                    <PlusIcon className="mr-1.5 size-3.5" />
                                    Add Block
                                </Button>
                            </div>

                            <div className="space-y-3">
                                {form.watch("blocks").map((_, index) => (
                                    <TimeBlockRow
                                        key={index}
                                        index={index}
                                        form={form}
                                        removeBlock={removeBlock}
                                    />
                                ))}
                            </div>
                        </div>

                        <DialogFooter>
                            <Button type="button" variant="outline" onClick={handleRouteBack}>
                                Cancel
                            </Button>
                            <Button type="submit" disabled={createTrack.isPending}>
                                {createTrack.isPending ? "Creating..." : "Create Timetable"}
                            </Button>
                        </DialogFooter>
                    </form>
                </Form>
            </DialogContent>
        </Dialog>
    );
}
