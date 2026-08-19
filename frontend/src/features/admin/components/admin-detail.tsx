/**
 * AdminDetail — displays and edits an admin's profile.
 *
 * Rendered both on the full page /admins/[id] and inside a
 * modal sheet when client-navigated from the admins listing.
 *
 * All forms are validated before submission.
 */

"use client";

import React from "react";
import { Loader2 } from "lucide-react";
import { z } from "zod";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import {
    Form,
    FormControl,
    FormField,
    FormItem,
    FormLabel,
    FormMessage,
} from "@/components/ui/form";
import { useAdminDetail, useUpdateAdmin } from "../hooks/use-admins";
import { useRouter } from "next/navigation";

interface AdminDetailProps {
    id: string;
}

const adminDetailSchema = z.object({
    fullName: z.string().trim().min(2, "Full name required with a minimum of 2 characters"),
});

type AdminDetailSchema = z.infer<typeof adminDetailSchema>;

export function AdminDetail({ id }: AdminDetailProps) {
    const router = useRouter();
    const { data: admin, isLoading } = useAdminDetail(id);
    const updateMutation = useUpdateAdmin();

    const form = useForm<AdminDetailSchema>({
        resolver: zodResolver(adminDetailSchema),
        defaultValues: {
            fullName: "",
        },
    });

    const onSubmit = React.useCallback(
        (values: AdminDetailSchema) => {
            updateMutation.mutate({
                userId: id,
                payload: { full_name: values.fullName },
            });
        },
        [updateMutation, id]
    );

    React.useEffect(() => {
        if (updateMutation.isSuccess) {
            router.back();
        }
    }, [updateMutation, router]);

    React.useEffect(() => {
        if (admin?.full_name) {
            form.setValue("fullName", admin.full_name);
        }
    }, [admin, form]);

    return (
        <div className="space-y-6 py-2">
            <div className="space-y-1.5">
                <Label>Email</Label>
                {isLoading ? (
                    <Skeleton className="h-5 w-40" />
                ) : (
                    <p className="text-muted-foreground text-sm">{admin?.email}</p>
                )}
            </div>

            <Form {...form}>
                <form onSubmit={form.handleSubmit(onSubmit)} className="mb-4 space-y-4">
                    <FormField
                        control={form.control}
                        name="fullName"
                        render={({ field }) => (
                            <FormItem>
                                <FormLabel>Full name</FormLabel>
                                <FormControl>
                                    <Input placeholder="Admin name" autoFocus {...field} />
                                </FormControl>
                                <FormMessage />
                            </FormItem>
                        )}
                    />
                    <Button type="submit" disabled={isLoading || updateMutation.isPending}>
                        {updateMutation.isPending && (
                            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                        )}
                        {`Updat${updateMutation.isPending ? "ing" : "e"}`}
                    </Button>
                </form>
            </Form>
        </div>
    );
}
