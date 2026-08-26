"use client";

import React from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { toast } from "sonner";
import { useCreateTrack } from "../hooks";
import { Form } from "@/components/ui/form";
import { TimetableCreateDetails } from "./timetable-create-details";
import { TimetableCreateBlocks } from "./timetable-create-blocks";

interface CreateTimetableProps {
    handleRouteBack?: () => void;
}

export const TimeBlockSchema = z.object({
    day_of_week: z.number().int().min(1).max(7),
    period_name: z.string().min(1, "Period name is required"),
    start_time: z
        .string()
        .regex(/^([0-1]?[0-9]|2[0-3]):[0-5][0-9]$/, "Invalid time format (HH:MM)"),
    end_time: z.string().regex(/^([0-1]?[0-9]|2[0-3]):[0-5][0-9]$/, "Invalid time format (HH:MM)"),
    is_break: z.boolean(),
});

export const CreateTimetableSchema = z.object({
    name: z.string().min(1, "Timetable name is required").max(100),
    description: z.string().max(500).optional(),
    blocks: z.array(TimeBlockSchema).min(1, "At least one time block is required"),
});

export type CreateTimetableFormData = z.infer<typeof CreateTimetableSchema>;
export type TimeBlockFormData = z.infer<typeof TimeBlockSchema>;

type Step = "details" | "blocks";

function sanitizeName(value: string): string {
    return value
        .trim()
        .replace(/\s+/g, " ")
        .replace(/[<>"'&]/g, "");
}

export function CreateTimetable({ handleRouteBack }: CreateTimetableProps) {
    const { mutate, isPending, isSuccess, isError, error } = useCreateTrack();
    const [step, setStep] = React.useState<Step>("details");

    const form = useForm<CreateTimetableFormData>({
        resolver: zodResolver(CreateTimetableSchema),
        defaultValues: {
            name: "",
            description: "",
            blocks: [
                {
                    day_of_week: 1,
                    period_name: "Lesson 1",
                    start_time: "08:00",
                    end_time: "09:00",
                    is_break: false,
                },
            ],
        },
    });

    React.useEffect(() => {
        if (isSuccess && handleRouteBack) {
            handleRouteBack();
        }
    }, [isSuccess, handleRouteBack]);

    React.useEffect(() => {
        if (isError && error) {
            toast.error(error?.message);
        }
    }, [isError, error]);

    const handleNext = async () => {
        const nameValue = form.getValues("name");
        const trimmed = sanitizeName(nameValue);
        if (!trimmed) {
            form.setError("name", { message: "Timetable name is required" });
            return;
        }
        form.setValue("name", trimmed, { shouldValidate: true });
        const valid = await form.trigger("name");
        if (valid) {
            setStep("blocks");
        }
    };

    const handleBack = () => {
        setStep("details");
    };

    const onSubmit = async (data: CreateTimetableFormData) => {
        const payload = {
            name: data.name,
            description: data.description,
            initial_blocks: data.blocks.map((block, i) => ({
                track_id: "",
                day_of_week: block.day_of_week,
                period_name: block.period_name,
                start_time: block.start_time,
                end_time: block.end_time,
                is_break: block.is_break,
                order: i + 1,
            })),
        };

        mutate(payload);
    };

    return (
        <div className="mx-auto max-h-[90vh] w-full space-y-6 overflow-y-auto p-6 sm:max-w-2xl">
            <div className="mb-4">
                <h2 className="text-2xl font-semibold tracking-tight">Create Timetable</h2>
                <p className="text-muted-foreground mt-1 text-sm">
                    {step === "details"
                        ? "Enter timetable track details."
                        : "Add time blocks (replayed for all days)."}
                </p>
            </div>

            <Form {...form}>
                <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6" noValidate>
                    {step === "details" && (
                        <TimetableCreateDetails
                            control={form.control}
                            onNext={handleNext}
                            onCancel={() => {
                                if (handleRouteBack) {
                                    handleRouteBack();
                                }
                            }}
                        />
                    )}

                    {step === "blocks" && (
                        <TimetableCreateBlocks
                            form={form}
                            onBack={handleBack}
                            onSubmit={() => form.handleSubmit(onSubmit)()}
                            isPending={isPending}
                        />
                    )}
                </form>
            </Form>
        </div>
    );
}
