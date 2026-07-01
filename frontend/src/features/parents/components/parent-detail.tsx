/**
 * Parent Detail — displays parent info and linked students.
 *
 * Section 1: Parent Info (name, email, phone, active toggle)
 * Section 2: Linked Students (table with link/unlink actions)
 */

"use client";

import * as React from "react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { ArrowLeft, Link2, Trash2, UserPlus } from "lucide-react";

import { useParentDetail, useUpdateParent, useUnlinkStudent } from "../hooks/use-parents";
import { LinkStudentDialog } from "./link-student-dialog";

// ─── Props ─────────────────────────────────────────────────────────────────

interface ParentDetailViewProps {
    parentId: string;
    onBack: () => void;
}

// ─── Empty State ──────────────────────────────────────────────────────────

function EmptyState({ onCreateLink }: { onCreateLink: () => void }) {
    return (
        <div className="bg-muted/30 flex items-center justify-center rounded-md px-4 py-8">
            <div className="text-center">
                <p className="text-muted-foreground text-sm font-medium">No linked students</p>
                <p className="text-muted-foreground mt-1 text-xs">
                    Link a student to this parent to manage guardian relationships.
                </p>
                <Button variant="outline" size="sm" className="mt-4" onClick={onCreateLink}>
                    <UserPlus className="mr-1.5 size-3.5" />
                    Link Student
                </Button>
            </div>
        </div>
    );
}

// ─── Component ─────────────────────────────────────────────────────────────

export function ParentDetailView({ parentId, onBack }: ParentDetailViewProps) {
    const { data: detailData, isLoading, isError } = useParentDetail(parentId);

    const updateParent = useUpdateParent();
    const unlinkStudent = useUnlinkStudent();

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
                    <p className="text-destructive text-sm font-medium">
                        Failed to load parent details.
                    </p>
                    <Button variant="outline" size="sm" className="mt-4" onClick={onBack}>
                        Back to Parents
                    </Button>
                </div>
            </div>
        );
    }

    const linkedCount = detail.linked_students?.length ?? 0;

    return (
        <div className="flex flex-1 flex-col px-6 pt-6 pb-8">
            {/* Back link */}
            <Button variant="ghost" size="sm" className="mb-4 w-fit" onClick={onBack}>
                <ArrowLeft className="mr-1.5 size-4" />
                Back to Parents
            </Button>

            {/* Section 1: Parent Info */}
            <div className="mb-8">
                <h1 className="text-2xl font-semibold tracking-tight">{detail.full_name}</h1>
                <div className="mt-4 space-y-4">
                    {/* Email (read-only) */}
                    <div>
                        <Label className="text-muted-foreground text-xs">Email</Label>
                        <p className="text-sm">{detail.email}</p>
                    </div>

                    {/* Phone (editable) */}
                    <div>
                        <Label className="text-muted-foreground text-xs">Phone Number</Label>
                        {isEditingPhone ? (
                            <div className="mt-1 flex items-center gap-2">
                                <Input
                                    value={displayPhone}
                                    onChange={(e) => setEditPhone(e.target.value)}
                                    className="h-8 max-w-xs text-sm"
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
                                <span className="text-sm">{detail.phone_number}</span>
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
                        <Label htmlFor="parent-active" className="text-sm">
                            {detail.is_active ? "Active" : "Inactive"}
                        </Label>
                    </div>
                </div>
            </div>

            {/* Section 2: Linked Students */}
            <div>
                <div className="mb-3 flex items-center justify-between">
                    <h2 className="text-lg font-medium">
                        Linked Students
                        {linkedCount > 0 && (
                            <span className="text-muted-foreground ml-2 text-sm font-normal">
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
                    <div className="ring-foreground/10 rounded-lg ring-1">
                        <table className="w-full">
                            <thead>
                                <tr className="border-border/40 border-b">
                                    <th className="text-muted-foreground px-3 py-2 text-left text-xs font-medium tracking-wider uppercase">
                                        Student Name
                                    </th>
                                    <th className="text-muted-foreground px-3 py-2 text-left text-xs font-medium tracking-wider uppercase">
                                        Relationship
                                    </th>
                                    <th className="text-muted-foreground px-3 py-2 text-left text-xs font-medium tracking-wider uppercase">
                                        Primary
                                    </th>
                                    <th className="w-16 px-3 py-2" />
                                </tr>
                            </thead>
                            <tbody>
                                {detail.linked_students.map((link) => (
                                    <tr
                                        key={link.student_id}
                                        className="group border-border/40 hover:bg-muted/30 border-b transition-colors"
                                    >
                                        <td className="px-3 py-2.5 text-sm font-medium">
                                            {link.full_name}
                                        </td>
                                        <td className="text-muted-foreground px-3 py-2.5 text-sm">
                                            {link.relationship || "—"}
                                        </td>
                                        <td className="px-3 py-2.5">
                                            {link.is_primary ? (
                                                <Badge
                                                    variant="secondary"
                                                    className="bg-sky-100 text-sky-700 dark:bg-sky-900/30 dark:text-sky-400"
                                                >
                                                    Primary
                                                </Badge>
                                            ) : (
                                                <span className="text-muted-foreground text-sm">
                                                    —
                                                </span>
                                            )}
                                        </td>
                                        <td className="px-3 py-2.5">
                                            <Button
                                                variant="ghost"
                                                size="icon-sm"
                                                className="opacity-0 transition-opacity group-hover:opacity-100"
                                                onClick={() =>
                                                    handleUnlink(link.student_id, link.full_name)
                                                }
                                                title="Unlink student"
                                            >
                                                <Trash2 className="text-destructive size-3.5" />
                                                <span className="sr-only">Unlink</span>
                                            </Button>
                                        </td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>
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
