"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { useRouter } from "next/navigation";
import { useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
    Form,
    FormControl,
    FormField,
    FormItem,
    FormLabel,
    FormMessage,
} from "@/components/ui/form";
import { useStream, useUpdateStream, useCreateStreams } from "../hooks/use-streams";
import { getErrorMessage } from "@/lib/errors";
import { toast } from "sonner";

const streamSchema = z.object({
    name: z.string().min(1, "Name is required"),
    color: z.string().min(1, "Color is required"),
});

type StreamValues = z.infer<typeof streamSchema>;

interface StreamFormProps {
    streamId?: string;
    onSuccess?: () => void;
}

export function StreamForm({ streamId, onSuccess }: StreamFormProps) {
    const router = useRouter();
    const isEdit = !!streamId;
    const { data: stream, isLoading } = useStream(streamId || "");
    const update = useUpdateStream();
    const create = useCreateStreams();

    const form = useForm<StreamValues>({
        resolver: zodResolver(streamSchema),
        defaultValues: {
            name: "",
            color: "#0050d1",
        },
    });

    useEffect(() => {
        if (stream && form.reset) {
            form.reset({ name: stream.name ?? "", color: stream.color ?? "#0050d1" });
        }
    }, [stream, form]);

    const onSubmit = async (values: StreamValues) => {
        try {
            if (isEdit) {
                await update.mutateAsync({
                    id: streamId!,
                    data: { name: values.name.trim(), color: values.color },
                });
            } else {
                await create.mutateAsync([{ name: values.name.trim(), color: values.color }]);
            }
            if (onSuccess) onSuccess();
            else router.back();
        } catch (err) {
            toast.error(getErrorMessage(err));
        }
    };

    if (isEdit && isLoading && !stream) {
        return <div className="text-muted-foreground">Loading stream…</div>;
    }

    return (
        <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
                <FormField
                    control={form.control}
                    name="name"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>Stream name</FormLabel>
                            <FormControl>
                                <Input
                                    id="stream-name"
                                    placeholder="e.g., Arts"
                                    {...field}
                                    disabled={update.isPending || create.isPending}
                                />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />
                <FormField
                    control={form.control}
                    name="color"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>Color</FormLabel>
                            <FormControl>
                                <div className="flex items-center gap-3">
                                    <input
                                        id="stream-color"
                                        type="color"
                                        {...field}
                                        disabled={update.isPending || create.isPending}
                                        className="m-0 h-6 w-7 cursor-pointer appearance-none border-0 bg-transparent p-0 [&::-moz-color-swatch]:rounded-md [&::-moz-color-swatch]:border-0 [&::-webkit-color-swatch]:rounded-md [&::-webkit-color-swatch]:border-0 [&::-webkit-color-swatch-wrapper]:p-0"
                                    />
                                    <div className="h-7 w-full rounded-md border p-1 px-2 text-xs">
                                        {field.value}
                                    </div>
                                </div>
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />
                <div className="flex items-center gap-2 pt-2">
                    <Button type="submit" disabled={update.isPending || create.isPending}>
                        {isEdit ? "Save changes" : "Add stream"}
                    </Button>
                </div>
            </form>
        </Form>
    );
}
