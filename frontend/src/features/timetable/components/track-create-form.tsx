"use client";

import React from "react";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { toast } from "sonner";
import { useCreateTrack } from "@/features/timetable/hooks";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import {
    Form,
    FormField,
    FormItem,
    FormLabel,
    FormControl,
    FormMessage,
} from "@/components/ui/form";

const Schema = z.object({
    name: z.string().min(1, "Name required").max(100),
    description: z.string().max(500).optional(),
});

export function TrackCreateForm() {
    const router = useRouter();
    const { mutate, isPending } = useCreateTrack();

    const form = useForm<{ name: string; description?: string }>({
        resolver: zodResolver(Schema),
        defaultValues: { name: "", description: "" },
    });

    const onSubmit = (data: { name: string; description?: string }) => {
        mutate(
            { name: data.name, description: data.description },
            {
                onSuccess: () => router.push("/timetable"),
            }
        );
    };

    return (
        <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
                <FormField
                    control={form.control}
                    name="name"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>Track Name</FormLabel>
                            <FormControl>
                                <Input {...field} placeholder="e.g., Lower Primary Track" />
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
                            <FormLabel>Description (optional)</FormLabel>
                            <FormControl>
                                <Textarea {...field} placeholder="Brief description" rows={3} />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />
                <div className="flex justify-end pt-2">
                    <Button type="submit" disabled={isPending}>
                        {isPending ? "Creating..." : "Create Track"}
                    </Button>
                </div>
            </form>
        </Form>
    );
}
