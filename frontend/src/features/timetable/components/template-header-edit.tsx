"use client";

import React from "react";
import { Pencil, Check, X } from "lucide-react";
import { z } from "zod";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Form, FormControl, FormField, FormItem, FormMessage } from "@/components/ui/form";
import { useUpdateTimetableTemplate } from "../hooks/use-timetable-templates";
import { useTimetableTemplate } from "../hooks/use-timetable-template";
import { Skeleton } from "@/components/ui/skeleton";

const editSchema = z.object({
    name: z.string().min(1, "Name is required").max(200, "Name too long"),
    description: z.string().max(500, "Description too long").optional(),
});

type EditSchema = z.infer<typeof editSchema>;

export function TemplateHeaderEdit({
    templateId,
    initialName,
    initialDescription,
    loading,
}: {
    templateId: string;
    initialName: string;
    loading: boolean;
    initialDescription?: string | null;
}) {
    const [isEditing, setIsEditing] = React.useState(false);
    const mutation = useUpdateTimetableTemplate();
    const { refetch } = useTimetableTemplate(templateId);

    const form = useForm<EditSchema>({
        resolver: zodResolver(editSchema),
        defaultValues: {
            name: initialName || "",
            description: initialDescription || "",
        },
    });

    React.useEffect(() => {
        if (!isEditing) {
            form.reset({
                name: initialName || "",
                description: initialDescription || "",
            });
        }
    }, [isEditing, initialName, initialDescription, form]);

    const handleSubmit = React.useCallback(
        (data: EditSchema) => {
            mutation.mutate(
                {
                    id: templateId,
                    name: data.name.trim(),
                    description: data.description?.trim() ?? "",
                },
                {
                    onSuccess: () => {
                        setIsEditing(false);
                        refetch();
                    },
                    onError: () => {
                        // mutation surfaces error; form-level errors handled via backend 400 if needed
                    },
                }
            );
        },
        [mutation, templateId, refetch]
    );

    const isDirty = form.formState.isDirty;
    const name = form.getValues("name") || initialName;
    const description = form.getValues("description") || initialDescription;

    if (loading) {
        return (
            <div className="h-16 w-38 space-y-2 pt-2">
                <Skeleton className="size-6 w-full rounded-xs" />
                <Skeleton className="size-4 w-3/4 rounded-xs" />
            </div>
        );
    }

    if (!isEditing) {
        return (
            <div className="flex h-16 gap-2">
                <div className="text-muted-foreground">
                    <h1 className="text-2xl capitalize">{name}</h1>
                    <p>{description}</p>
                </div>
                <Button
                    size="icon"
                    variant="ghost"
                    type="button"
                    aria-label="Edit template"
                    onClick={() => setIsEditing(true)}
                    className="text-muted-foreground hover:text-foreground ml-3 inline-flex items-center gap-1.5 text-sm"
                >
                    <Pencil className="h-4 w-4" />
                </Button>
            </div>
        );
    }

    return (
        <div className="ml-3 flex items-start gap-2">
            <Form {...form}>
                <form onSubmit={form.handleSubmit(handleSubmit)} className="flex items-start gap-2">
                    <div className="flex min-w-60 flex-col gap-2">
                        <FormField
                            control={form.control}
                            name="name"
                            render={({ field }) => (
                                <FormItem>
                                    <FormControl>
                                        <Input
                                            {...field}
                                            placeholder="Template name"
                                            disabled={mutation.isPending}
                                        />
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
                                    <FormControl>
                                        <Input
                                            {...field}
                                            placeholder="Description"
                                            disabled={mutation.isPending}
                                        />
                                    </FormControl>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />
                    </div>
                    <div className="mt-1 flex items-center gap-1 self-start">
                        <Button
                            type="submit"
                            size="icon"
                            variant="ghost"
                            disabled={mutation.isPending || !isDirty}
                            aria-label="Save"
                        >
                            <Check className="h-4 w-4" />
                        </Button>
                        <Button
                            type="button"
                            size="icon"
                            variant="ghost"
                            onClick={() => setIsEditing(false)}
                            aria-label="Cancel"
                        >
                            <X className="h-4 w-4" />
                        </Button>
                    </div>
                </form>
            </Form>
        </div>
    );
}
