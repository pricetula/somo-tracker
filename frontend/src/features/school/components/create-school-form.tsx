/**
 * CreateSchoolForm — dialog-friendly form to create a new school.
 * Uses shadcn Form + zod + react-hook-form. Field: school_name.
 * Integrates with useCreateSchool mutation.
 */

"use client";

import React from "react";
import { z } from "zod";
import { Loader2 } from "lucide-react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
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
import { useCreateSchool } from "../hooks/use-create-school";

// ─── Zod Schema ────────────────────────────────────────────────────────────

const schema = z.object({
    school_name: z
        .string()
        .min(2, "School name must be at least 2 characters")
        .max(100, "School name is too long"),
});

type Schema = z.infer<typeof schema>;

// ─── Component ─────────────────────────────────────────────────────────────

export function CreateSchoolForm() {
    const router = useRouter();
    const { mutate, isPending } = useCreateSchool();

    const form = useForm<Schema>({
        resolver: zodResolver(schema),
        defaultValues: {
            school_name: "",
        },
    });

    const onSubmit = React.useCallback(
        (data: Schema) => {
            mutate(
                {
                    school_name: data.school_name.trim(),
                },
                {
                    onSuccess() {
                        router.back();
                    },
                }
            );
        },
        [mutate, router]
    );

    return (
        <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-5">
                <FormField
                    control={form.control}
                    name="school_name"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>School Name</FormLabel>
                            <FormControl>
                                <Input
                                    placeholder="e.g. Moi Girls School"
                                    {...field}
                                    disabled={isPending}
                                />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />

                <div className="flex items-center justify-end gap-2 pt-2">
                    <Button
                        type="button"
                        variant="outline"
                        onClick={() => router.back()}
                        disabled={isPending}
                    >
                        Cancel
                    </Button>
                    <Button type="submit" disabled={isPending}>
                        {isPending ? (
                            <>
                                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                                Creating…
                            </>
                        ) : (
                            "Create School"
                        )}
                    </Button>
                </div>
            </form>
        </Form>
    );
}
