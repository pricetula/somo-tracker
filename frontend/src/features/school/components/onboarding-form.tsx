/**
 * OnboardingForm — school registration onboarding.
 *
 * Uses shadcn Form + zod + react-hook-form. Fields: school_name, user_name.
 * Integrates with useRegisterSchool mutation.
 */

"use client";

import React from "react";
import { z } from "zod";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
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
import { useRegisterSchool } from "../hooks/use-schools";

// ─── Zod Schema ────────────────────────────────────────────────────────────

const onboardingSchema = z.object({
    school_name: z
        .string()
        .min(2, "School name must be at least 2 characters")
        .max(100, "School name is too long"),
    user_name: z
        .string()
        .min(2, "Your name must be at least 2 characters")
        .max(100, "Name is too long"),
});

type OnboardingSchema = z.infer<typeof onboardingSchema>;

// ─── Props ─────────────────────────────────────────────────────────────
interface OnboardingFormProps {
    onSuccess: (schoolId: string) => void;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function OnboardingForm({ onSuccess }: OnboardingFormProps) {
    const { mutate, isPending } = useRegisterSchool();

    const form = useForm<OnboardingSchema>({
        resolver: zodResolver(onboardingSchema),
        defaultValues: {
            school_name: "",
            user_name: "",
        },
    });

    const onSubmit = React.useCallback(
        (data: OnboardingSchema) => {
            mutate(
                {
                    school_name: data.school_name.trim(),
                    user_name: data.user_name.trim(),
                },
                {
                    onSuccess(data) {
                        if (data?.school_id) {
                            onSuccess(data.school_id);
                        }
                    },
                    onError(error) {
                        toast.error(error?.message ?? "failed to register school");
                    },
                }
            );
        },
        [mutate, onSuccess]
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

                <FormField
                    control={form.control}
                    name="user_name"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>Your Name</FormLabel>
                            <FormControl>
                                <Input
                                    placeholder="e.g. Jane Doe"
                                    {...field}
                                    disabled={isPending}
                                />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />

                <Button type="submit" className="w-full" disabled={isPending}>
                    {isPending ? (
                        <>
                            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                            Registering…
                        </>
                    ) : (
                        "Register School"
                    )}
                </Button>
            </form>
        </Form>
    );
}
