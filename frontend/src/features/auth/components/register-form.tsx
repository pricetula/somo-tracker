/**
 * Register Form — the inner form component for creating a school account.
 *
 * Reads `session_ref` from URL search params and handles the registration mutation.
 * Must be wrapped in <Suspense> at the call site due to useSearchParams.
 */

"use client";

import { useSearchParams, useRouter } from "next/navigation";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { useEffect } from "react";
import { Loader2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
    Form,
    FormControl,
    FormField,
    FormItem,
    FormLabel,
    FormMessage,
    FormDescription,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { isApiError } from "@/lib/errors";
import { useRegister } from "@/hooks/use-auth";
import { DocTooltip } from "@/components/ui/DocTooltip";

// ─── Schema ───────────────────────────────────────────────────────────────

const registerSchema = z.object({
    school_name: z
        .string()
        .min(2, "School name must be at least 2 characters")
        .max(100, "School name must be less than 100 characters"),
    first_name: z
        .string()
        .min(1, "First name is required")
        .max(100, "First name must be less than 100 characters"),
    last_name: z
        .string()
        .min(1, "Last name is required")
        .max(100, "Last name must be less than 100 characters"),
});

type RegisterValues = z.infer<typeof registerSchema>;

// ─── Types ─────────────────────────────────────────────────────────────────

export interface RegisterFormProps {
    tooltipSummary?: string;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function RegisterForm({ tooltipSummary }: RegisterFormProps) {
    const searchParams = useSearchParams();
    const router = useRouter();
    const sessionRef = searchParams.get("session_ref");
    const registerMutation = useRegister();

    const form = useForm<RegisterValues>({
        resolver: zodResolver(registerSchema),
        defaultValues: {
            school_name: "",
            first_name: "",
            last_name: "",
        },
    });

    // Redirect to login if no session_ref is present
    useEffect(() => {
        if (!sessionRef) {
            router.replace("/login");
        }
    }, [sessionRef, router]);

    if (!sessionRef) {
        return null;
    }

    function onSubmit(values: RegisterValues) {
        registerMutation.mutate(
            {
                school_name: values.school_name,
                session_ref: sessionRef!,
                first_name: values.first_name,
                last_name: values.last_name,
            },
            {
                onError: (err) => {
                    // Map 400 field validation errors to form fields
                    if (isApiError(err) && err.status === 400 && err.errors) {
                        Object.entries(err.errors).forEach(([field, messages]) => {
                            form.setError(field as keyof RegisterValues, {
                                type: "server",
                                message: messages[0],
                            });
                        });
                    } else if (isApiError(err) && err.status === 401) {
                        // Handled by useRegister's global onError
                    } else {
                        // Fallback: show a generic toast (handled by useRegister)
                    }
                },
            }
        );
    }

    return (
        <div className="flex min-h-screen items-center justify-center px-4">
            <Card className="w-full max-w-md">
                <CardHeader className="text-center">
                    <CardTitle className="text-2xl">Create Your School Account</CardTitle>
                    <CardDescription>
                        Set up your school or educational organization
                        {tooltipSummary && (
                            <DocTooltip summary={tooltipSummary} slug="authentication" />
                        )}
                    </CardDescription>
                </CardHeader>
                <CardContent>
                    <Form {...form}>
                        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
                            <FormField
                                control={form.control}
                                name="school_name"
                                render={({ field }) => (
                                    <FormItem>
                                        <FormLabel>School Name</FormLabel>
                                        <FormControl>
                                            <Input
                                                placeholder="e.g. Lincoln High School"
                                                autoFocus
                                                {...field}
                                            />
                                        </FormControl>
                                        <FormDescription>
                                            The name of your educational institution
                                        </FormDescription>
                                        <FormMessage />
                                    </FormItem>
                                )}
                            />
                            <div className="grid grid-cols-2 gap-3">
                                <FormField
                                    control={form.control}
                                    name="first_name"
                                    render={({ field }) => (
                                        <FormItem>
                                            <FormLabel>First Name</FormLabel>
                                            <FormControl>
                                                <Input placeholder="Jane" {...field} />
                                            </FormControl>
                                            <FormMessage />
                                        </FormItem>
                                    )}
                                />
                                <FormField
                                    control={form.control}
                                    name="last_name"
                                    render={({ field }) => (
                                        <FormItem>
                                            <FormLabel>Last Name</FormLabel>
                                            <FormControl>
                                                <Input placeholder="Doe" {...field} />
                                            </FormControl>
                                            <FormMessage />
                                        </FormItem>
                                    )}
                                />
                            </div>
                            <Button
                                type="submit"
                                className="w-full"
                                disabled={registerMutation.isPending}
                            >
                                {registerMutation.isPending && (
                                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                                )}
                                Create Account
                            </Button>
                        </form>
                    </Form>
                </CardContent>
            </Card>
        </div>
    );
}
