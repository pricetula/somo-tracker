"use client";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Trash2, Link2 } from "lucide-react";
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
    useParentDetail,
    useUpdateParent,
    useUnlinkStudent,
    useDeleteParent,
} from "../hooks/use-parents";
import { LinkStudentDialog } from "./link-student-dialog";
import * as React from "react";

interface ParentDetailViewProps {
    parentId: string;
    onBack: () => void;
}

import { EmptyState } from "./empty-state";

export function ParentDetailView({ parentId, onBack }: ParentDetailViewProps) {
    const { data: detailData, isLoading, isError } = useParentDetail(parentId);

    const updateParent = useUpdateParent();
    const unlinkStudent = useUnlinkStudent();
    const deleteMutation = useDeleteParent();

    const [linkDialogOpen, setLinkDialogOpen] = React.useState(false);
    const [editPhone, setEditPhone] = React.useState<string | null>(null);

    const detail = detailData?.data;
    const displayPhone = editPhone ?? detail?.phone_number ?? "";
    const isEditingPhone = editPhone !== null;

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

    const handleSavePhone = async () => {
        const phone = displayPhone.trim();
        if (!detail || !phone) return;
        try {
            await updateParent.mutateAsync({
                id: parentId,
                data: { phone_number: phone },
            });
            setEditPhone(null);
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

    if (isError || !detail) {
        return (
            <div className="flex items-center justify-center py-16">
                <div className="text-center">
                    <p className="text-destructive font-medium">Failed to load parent details.</p>
                    <Button variant="outline" size="sm" className="mt-4" onClick={onBack}>
                        Go to Parents
                    </Button>
                </div>
            </div>
        );
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
                        <Label className="text-muted-foreground text-xs">Email</Label>
                        <p className="">{detail.email}</p>
                    </div>

                    {/* Phone (editable) */}
                    <div>
                        <Label className="text-muted-foreground text-xs">Phone Number</Label>
                        {isEditingPhone ? (
                            <div className="mt-1 flex items-center gap-2">
                                <Input
                                    value={displayPhone}
                                    onChange={(e) => setEditPhone(e.target.value)}
                                    className="h-8 max-w-xs"
                                />
                                <Button size="sm" variant="outline" onClick={handleSavePhone}>
                                    Save
                                </Button>
                                <Button
                                    size="sm"
                                    variant="ghost"
                                    onClick={() => setEditPhone(null)}
                                >
                                    Cancel
                                </Button>
                            </div>
                        ) : (
                            <div className="mt-1 flex items-center gap-2">
                                <span className="">{detail.phone_number}</span>
                                <Button
                                    variant="ghost"
                                    size="icon-sm"
                                    onClick={() => setEditPhone(detail?.phone_number ?? "")}
                                >
                                    <svg
                                        className="size-3.5"
                                        fill="none"
                                        stroke="currentColor"
                                        viewBox="0 0 24 24"
                                    >
                                        <path
                                            strokeLinecap="round"
                                            strokeLinejoin="round"
                                            strokeWidth={2}
                                            d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z"
                                        />
                                    </svg>
                                    <span className="sr-only">Edit phone</span>
                                </Button>
                            </div>
                        )}
                    </div>

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

                    {/* Delete parent */}
                    <AlertDialog>
                        <AlertDialogTrigger asChild>
                            <Button variant="outline" size="sm" className="text-destructive">
                                <Trash2 className="mr-1.5 size-3.5" />
                                Delete Parent
                            </Button>
                        </AlertDialogTrigger>
                        <AlertDialogContent>
                            <AlertDialogHeader>
                                <AlertDialogTitle>Delete Parent</AlertDialogTitle>
                                <AlertDialogDescription>
                                    Are you sure you want to delete &ldquo;{detail.full_name}
                                    &rdquo;? This action cannot be undone.
                                </AlertDialogDescription>
                            </AlertDialogHeader>
                            <AlertDialogFooter>
                                <AlertDialogCancel>Cancel</AlertDialogCancel>
                                <AlertDialogAction
                                    variant="destructive"
                                    onClick={async () => {
                                        try {
                                            await deleteMutation.mutateAsync(parentId);
                                            onBack();
                                        } catch {
                                            // handled by hook onError
                                        }
                                    }}
                                    disabled={deleteMutation.isPending}
                                >
                                    {deleteMutation.isPending ? "Deleting…" : "Delete"}
                                </AlertDialogAction>
                            </AlertDialogFooter>
                        </AlertDialogContent>
                    </AlertDialog>
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
                                        {link.relationship || "—"}
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
