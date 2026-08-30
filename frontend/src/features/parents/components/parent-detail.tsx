"use client";

import React from "react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Trash2, Link2, Loader2 } from "lucide-react";
import { StaticTable } from "@/components/shared/static-table";
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
    AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import {
    Form,
    FormControl,
    FormField,
    FormItem,
    FormLabel,
    FormMessage,
} from "@/components/ui/form";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { getErrorMessage } from "@/lib/errors";
import {
    useParentDetail,
    useUpdateParent,
    useUnlinkStudent,
    useDeleteParent,
} from "../hooks/use-parents";
import { LinkStudentDialog } from "./link-student-dialog";
import { EmptyState } from "./empty-state";

interface ParentDetailViewProps {
    parentId: string;
}

const parentDetailSchema = z.object({
    phoneNumber: z.string().trim().min(1, "Phone number is required"),
});

type ParentDetailSchema = z.infer<typeof parentDetailSchema>;

export function ParentDetailView({ parentId }: ParentDetailViewProps) {
    const router = useRouter();
    const { data: detailData, isLoading, isError, error } = useParentDetail(parentId);
    const updateParent = useUpdateParent();
    const unlinkStudent = useUnlinkStudent();
    const deleteMutation = useDeleteParent();

    const [linkDialogOpen, setLinkDialogOpen] = React.useState(false);

    const detail = detailData?.data;

    const form = useForm<ParentDetailSchema>({
        resolver: zodResolver(parentDetailSchema),
        defaultValues: {
            phoneNumber: "",
        },
    });

    React.useEffect(() => {
        if (isError) {
            toast.error(getErrorMessage(error));
        }
    }, [isError, error]);

    React.useEffect(() => {
        if (updateParent.error) {
            toast.error(getErrorMessage(updateParent.error));
        }
        if (updateParent.isSuccess) {
            router.back();
            toast.success("Parent updated successfully.");
        }
    }, [updateParent, router]);

    const onSubmit = React.useCallback(
        (values: ParentDetailSchema) => {
            updateParent.mutate({
                id: parentId,
                data: { phone_number: values.phoneNumber.trim() },
            });
        },
        [updateParent, parentId]
    );

    React.useEffect(() => {
        if (detail?.phone_number) {
            form.setValue("phoneNumber", detail.phone_number);
        }
    }, [detail, form]);

    const handleToggleActive = async () => {
        if (!detail) return;
        try {
            await updateParent.mutateAsync({
                id: parentId,
                data: { is_active: !detail.is_active },
            });
        } catch {
            // handled by mutation onError
        }
    };

    const handleUnlink = async (studentId: string, studentName: string) => {
        if (!window.confirm(`Unlink ${studentName} from this parent?`)) {
            return;
        }

        try {
            await unlinkStudent.mutateAsync({ parentId, studentId });
        } catch {
            // handled by mutation onError
        }
    };

    const handleDelete = async () => {
        try {
            await deleteMutation.mutateAsync(parentId);
            router.back();
        } catch {
            // Error handled by the hook
        }
    };

    // Loading state
    if (isLoading) {
        return (
            <div className="flex flex-col gap-4 px-6 pt-6 pb-8">
                <Skeleton className="h-8 w-64" />
                <Skeleton className="h-4 w-48" />
                <Skeleton className="mt-4 h-32 w-full" />
                <Skeleton className="h-32 w-full" />
            </div>
        );
    }

    // Error state
    if (isError) {
        return null;
    }

    // Not found state
    if (!detail) {
        return <p className="text-muted-foreground py-4">Parent not found.</p>;
    }

    const linkedCount = detail.linked_students?.length ?? 0;

    return (
        <div className="flex flex-1 flex-col px-6 pt-6 pb-8">
            {/* Section 1: Parent Info */}
            <div className="mb-8">
                <h1 className="text-2xl font-semibold tracking-tight">{detail.full_name}</h1>
                <div className="mt-4 space-y-4">
                    {/* Email (read-only) */}
                    <div>
                        <Label className="text-muted-foreground">Email</Label>
                        <p className="">{detail.email}</p>
                    </div>

                    {/* Phone (editable) - using Form */}
                    <Form {...form}>
                        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
                            <FormField
                                control={form.control}
                                name="phoneNumber"
                                render={({ field }) => (
                                    <FormItem>
                                        <FormLabel htmlFor="phone-number">Phone Number</FormLabel>
                                        <FormControl>
                                            <Input
                                                id="phone-number"
                                                placeholder="Phone number"
                                                autoFocus
                                                {...field}
                                            />
                                        </FormControl>
                                        <FormMessage />
                                    </FormItem>
                                )}
                            />

                            {/* Active toggle */}
                            <div className="flex items-center gap-3">
                                <Switch
                                    id="parent-active"
                                    checked={detail.is_active}
                                    onCheckedChange={handleToggleActive}
                                    disabled={updateParent.isPending}
                                />
                                <Label htmlFor="parent-active" className="">
                                    {detail.is_active ? "Active" : "Inactive"}
                                </Label>
                            </div>

                            <footer className="flex gap-4">
                                {/* Save button */}
                                <Button type="submit" disabled={updateParent.isPending}>
                                    {updateParent.isPending ? (
                                        <>
                                            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                                            Saving…
                                        </>
                                    ) : (
                                        "Save Changes"
                                    )}
                                </Button>
                                {/* Delete parent */}
                                <AlertDialog>
                                    <AlertDialogTrigger>
                                        <Button variant="outline" className="text-destructive">
                                            <Trash2 className="mr-1.5 size-3.5" />
                                            Delete Parent
                                        </Button>
                                    </AlertDialogTrigger>
                                    <AlertDialogContent>
                                        <AlertDialogHeader>
                                            <AlertDialogTitle>Delete Parent</AlertDialogTitle>
                                            <AlertDialogDescription>
                                                Are you sure you want to delete &ldquo;
                                                {detail.full_name}&rdquo;? This action cannot be
                                                undone.
                                            </AlertDialogDescription>
                                        </AlertDialogHeader>
                                        <AlertDialogFooter>
                                            <AlertDialogCancel>Cancel</AlertDialogCancel>
                                            <AlertDialogAction
                                                variant="destructive"
                                                onClick={handleDelete}
                                                disabled={deleteMutation.isPending}
                                            >
                                                {deleteMutation.isPending ? "Deleting…" : "Delete"}
                                            </AlertDialogAction>
                                        </AlertDialogFooter>
                                    </AlertDialogContent>
                                </AlertDialog>
                            </footer>
                        </form>
                    </Form>
                </div>
            </div>

            {/* Section 2: Linked Students */}
            <div>
                <div className="mb-3 flex items-center justify-between">
                    <h2 className="text-lg font-medium">
                        Linked Students
                        {linkedCount > 0 && (
                            <span className="text-muted-foreground ml-2 font-normal">
                                ({linkedCount})
                            </span>
                        )}
                    </h2>
                    <Button variant="outline" size="sm" onClick={() => setLinkDialogOpen(true)}>
                        <Link2 className="mr-1.5 size-3.5" />
                        Link Student
                    </Button>
                </div>

                {linkedCount === 0 ? (
                    <EmptyState onCreateLink={() => setLinkDialogOpen(true)} />
                ) : (
                    <StaticTable
                        columns={[
                            {
                                id: "name",
                                header: "Student Name",
                                cell: (link) => (
                                    <span className="font-medium">{link.full_name}</span>
                                ),
                            },
                            {
                                id: "relationship",
                                header: "Relationship",
                                cell: (link) => (
                                    <span className="text-muted-foreground">
                                        {link.relationship || "\u2014"}
                                    </span>
                                ),
                            },
                            {
                                id: "primary",
                                header: "Primary",
                                cell: (link) =>
                                    link.is_primary ? (
                                        <Badge
                                            variant="secondary"
                                            className="bg-sky-100 text-sky-700 dark:bg-sky-900/30 dark:text-sky-400"
                                        >
                                            Primary
                                        </Badge>
                                    ) : (
                                        <span className="text-muted-foreground">—</span>
                                    ),
                            },
                            {
                                id: "actions",
                                header: "",
                                width: "64px",
                                cell: (link) => (
                                    <Button
                                        variant="ghost"
                                        size="icon-sm"
                                        onClick={() =>
                                            handleUnlink(link.student_id, link.full_name)
                                        }
                                        title="Unlink student"
                                    >
                                        <Trash2 className="text-destructive size-3.5" />
                                        <span className="sr-only">Unlink</span>
                                    </Button>
                                ),
                            },
                        ]}
                        data={detail.linked_students}
                        getRowId={(link) => link.student_id}
                        height={280}
                    />
                )}
            </div>

            {/* Link Student Dialog */}
            <LinkStudentDialog
                open={linkDialogOpen}
                onOpenChange={setLinkDialogOpen}
                parentId={parentId}
            />
        </div>
    );
}
