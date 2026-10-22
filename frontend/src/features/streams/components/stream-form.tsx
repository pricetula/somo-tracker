"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useForm, useFieldArray } from "react-hook-form";
import { z } from "zod";
import { useRouter } from "next/navigation";
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
import { Plus, X } from "lucide-react";
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

/**
 * StreamForm — supports both:
 *   • Edit mode (single stream) when `streamId` is provided
 *   • Create mode with multiple stream entries when no `streamId`
 */
export function StreamForm({ streamId, onSuccess }: StreamFormProps) {
    const router = useRouter();
    const isEdit = !!streamId;
    const { data: stream, isLoading } = useStream(streamId || "");
    const update = useUpdateStream();
    const create = useCreateStreams();

    // ── Edit mode: single‑stream handling (identical to original) ──
    const editForm = useForm<StreamValues>({
        resolver: zodResolver(streamSchema),
        defaultValues: {
            name: stream?.name ?? "",
            color: stream?.color ?? "",
        },
    });

    const onEditSubmit = async (values: StreamValues) => {
        try {
            await update.mutateAsync({
                id: streamId!,
                data: { name: values.name.trim(), color: values.color },
            });
            if (onSuccess) onSuccess();
            else router.back();
        } catch (err) {
            toast.error(getErrorMessage(err));
        }
    };

    // ── Create mode: multiple‑stream handling using useFieldArray ──
    const createForm = useForm<{ streams: StreamValues[] }>({
        resolver: zodResolver(
            z.object({
                streams: z.array(streamSchema).min(1, "At least one stream required"),
            })
        ),
        defaultValues: {
            streams: [{ name: "", color: "" }], // start with one empty entry
        },
    });

    const { fields, remove, insert } = useFieldArray({
        control: createForm.control,
        name: "streams",
    });

    const onCreateSubmit = async (data: { streams: StreamValues[] }) => {
        try {
            await create.mutateAsync(
                data.streams.map((s) => ({
                    name: s.name.trim(),
                    color: s.color ?? "",
                }))
            );
            if (onSuccess) onSuccess();
            else router.back();
        } catch (err) {
            toast.error(getErrorMessage(err));
        }
    };

    const handleAddStream = () => {
        // Insert new stream at the top
        insert(0, { name: "", color: "" });
    };

    const handleRemoveStream = (index: number) => {
        remove(index);
    };

    // ── UI ───────────────────────────────────────────────────────────
    if (isEdit && isLoading && !stream) {
        return <div className="text-muted-foreground">Loading stream…</div>;
    }

    if (isEdit) {
        // ---- Edit mode UI (single stream) ----
        return (
            <Form {...editForm}>
                <form onSubmit={editForm.handleSubmit(onEditSubmit)} className="space-y-4">
                    <FormField
                        control={editForm.control}
                        name="name"
                        render={({ field }) => (
                            <FormItem>
                                <FormLabel>Stream name</FormLabel>
                                <FormControl>
                                    <Input
                                        id="stream-name"
                                        placeholder="e.g., Arts"
                                        {...field}
                                        disabled={update.isPending}
                                    />
                                </FormControl>
                                <FormMessage />
                            </FormItem>
                        )}
                    />
                    <FormField
                        control={editForm.control}
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
                                            disabled={update.isPending}
                                            className="m-0 h-6 w-7 cursor-pointer appearance-none border-0 bg-transparent p-0"
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
                        <Button type="submit" disabled={update.isPending}>
                            Save changes
                        </Button>
                    </div>
                </form>
            </Form>
        );
    }

    // ---- Create mode UI (multiple streams) ----
    return (
        <Form {...createForm}>
            <form onSubmit={createForm.handleSubmit(onCreateSubmit)} className="space-y-4">
                {/* Plus button — top right of list */}
                <div className="flex justify-end pt-2">
                    <Button
                        type="button"
                        variant="outline"
                        size="icon"
                        onClick={handleAddStream}
                        disabled={createForm.formState.isSubmitting}
                        aria-label="Add stream"
                    >
                        <Plus className="h-4 w-4" />
                    </Button>
                </div>

                {/* Render each stream entry */}
                {fields.map((field, index) => (
                    <div key={field.id} className="flex items-end">
                        <FormField
                            control={createForm.control}
                            name={`streams.${index}.name`}
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>Stream name</FormLabel>
                                    <FormControl>
                                        <Input
                                            id={`stream-name-${index}`}
                                            placeholder="e.g., Arts"
                                            {...field}
                                            disabled={createForm.formState.isSubmitting}
                                            className="w-full"
                                        />
                                    </FormControl>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />
                        <FormField
                            control={createForm.control}
                            name={`streams.${index}.color`}
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>Color</FormLabel>
                                    <FormControl>
                                        <input
                                            id={`stream-color-${index}`}
                                            type="color"
                                            {...field}
                                            value={field.value || "#000000"}
                                            disabled={createForm.formState.isSubmitting}
                                            className="m-0 h-6 w-7 cursor-pointer appearance-none border-0 bg-transparent p-0"
                                        />
                                    </FormControl>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />
                        <Button
                            type="button"
                            variant="outline"
                            size="sm"
                            onClick={() => handleRemoveStream(index)}
                            disabled={createForm.formState.isSubmitting}
                        >
                            <X />
                        </Button>
                    </div>
                ))}

                {/* Submit button */}
                <div className="flex items-center gap-2 pt-2">
                    <Button type="submit" disabled={createForm.formState.isSubmitting}>
                        Add streams
                    </Button>
                </div>
            </form>
        </Form>
    );
}
