/**
 * Invite Form — reusable form for creating multiple invitations.
 *
 * Used both as:
 * - A dialog (in parallel intercepted route)
 * - A standalone page (direct navigation)
 */

"use client";

import * as React from "react";
import { useRouter } from "next/navigation";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog";
import { X, Plus } from "lucide-react";

import { useCreateInvitations } from "@/features/invitations/hooks/use-invitations";
import type { InvitationRole } from "@/lib/api/invitations";

// ─── Types ─────────────────────────────────────────────────────────────────

interface InviteRow {
    key: string;
    email: string;
    first_name: string;
    last_name: string;
    role: InvitationRole;
}

interface InviteFormProps {
    /** When true, renders inside a Dialog (for modal interception). */
    asDialog?: boolean;
    open?: boolean;
    onOpenChange?: (open: boolean) => void;
}

// ─── Constants ─────────────────────────────────────────────────────────────

const DEFAULT_ROLE: InvitationRole = "TEACHER";

const ROLE_OPTIONS: { value: InvitationRole; label: string }[] = [
    { value: "SCHOOL_ADMIN", label: "School Admin" },
    { value: "TEACHER", label: "Teacher" },
    { value: "SUPPORT_STAFF", label: "Staff" },
];

// ─── Form Content ──────────────────────────────────────────────────────────

function InviteFormContent({ onSuccess }: { onSuccess: () => void }) {
    const [rows, setRows] = React.useState<InviteRow[]>([
        { key: "1", email: "", first_name: "", last_name: "", role: DEFAULT_ROLE },
    ]);
    const [sending, setSending] = React.useState(false);

    const createInvitations = useCreateInvitations();

    function addRow() {
        setRows((prev) => [
            ...prev,
            {
                key: crypto.randomUUID?.() ?? String(Date.now()),
                email: "",
                first_name: "",
                last_name: "",
                role: DEFAULT_ROLE,
            },
        ]);
    }

    function removeRow(key: string) {
        setRows((prev) => prev.filter((r) => r.key !== key));
    }

    function updateRow(key: string, field: keyof InviteRow, value: string) {
        setRows((prev) => prev.map((r) => (r.key === key ? { ...r, [field]: value } : r)));
    }

    async function handleSend() {
        const validRows = rows.filter((r) => r.email.trim() !== "");
        if (validRows.length === 0) return;

        setSending(true);
        try {
            await createInvitations.mutateAsync({
                invites: validRows.map((r) => ({
                    email: r.email.trim(),
                    first_name: r.first_name.trim(),
                    last_name: r.last_name.trim(),
                    role: r.role,
                })),
            });
            onSuccess();
        } catch {
            // Error toast is handled by the mutation
        } finally {
            setSending(false);
        }
    }

    const validCount = rows.filter((r) => r.email.trim() !== "").length;

    return (
        <>
            <div className="max-h-[400px] space-y-3 overflow-y-auto px-4">
                {/* Header */}
                <div className="grid grid-cols-[1fr_1fr_1.5fr_1fr_28px] gap-2 px-1">
                    <span className="text-muted-foreground text-xs font-medium">First Name</span>
                    <span className="text-muted-foreground text-xs font-medium">Last Name</span>
                    <span className="text-muted-foreground text-xs font-medium">Email *</span>
                    <span className="text-muted-foreground text-xs font-medium">Role *</span>
                    <span />
                </div>

                {/* Rows */}
                {rows.map((row) => (
                    <div
                        key={row.key}
                        className="grid grid-cols-[1fr_1fr_1.5fr_1fr_28px] items-start gap-2"
                    >
                        <Input
                            placeholder="Jane"
                            value={row.first_name}
                            onChange={(e) => updateRow(row.key, "first_name", e.target.value)}
                            className="h-9 text-sm"
                            disabled={sending}
                        />
                        <Input
                            placeholder="Doe"
                            value={row.last_name}
                            onChange={(e) => updateRow(row.key, "last_name", e.target.value)}
                            className="h-9 text-sm"
                            disabled={sending}
                        />
                        <Input
                            placeholder="jane@school.edu"
                            value={row.email}
                            onChange={(e) => updateRow(row.key, "email", e.target.value)}
                            className="h-9 text-sm"
                            disabled={sending}
                        />
                        <Select
                            value={row.role}
                            onValueChange={(val) =>
                                updateRow(row.key, "role", val as InvitationRole)
                            }
                            disabled={sending}
                        >
                            <SelectTrigger className="h-9 text-sm">
                                <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                                {ROLE_OPTIONS.map((opt) => (
                                    <SelectItem key={opt.value} value={opt.value}>
                                        {opt.label}
                                    </SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                        <Button
                            variant="ghost"
                            size="icon-sm"
                            onClick={() => removeRow(row.key)}
                            disabled={rows.length <= 1 || sending}
                            className="mt-0.5"
                        >
                            <X className="size-4" />
                        </Button>
                    </div>
                ))}

                {/* Add row button */}
                <Button
                    variant="ghost"
                    size="sm"
                    onClick={addRow}
                    disabled={sending}
                    className="text-muted-foreground w-full text-xs"
                >
                    <Plus className="mr-1.5 size-3.5" />
                    Add another
                </Button>
            </div>

            {/* Footer */}
            <div className="flex items-center justify-between px-4 pb-4">
                <p className="text-muted-foreground text-xs">
                    {validCount > 0
                        ? `${validCount} invitation${validCount !== 1 ? "s" : ""} ready`
                        : "Add at least one email with a role"}
                </p>
                <div className="flex gap-3">
                    <Button onClick={handleSend} disabled={validCount === 0 || sending}>
                        {sending ? "Sending..." : "Send Invitations"}
                    </Button>
                </div>
            </div>
        </>
    );
}

// ─── Dialog Variant ────────────────────────────────────────────────────────

export function InviteFormDialog({ open, onOpenChange, asDialog }: InviteFormProps) {
    const router = useRouter();

    function handleSuccess() {
        if (onOpenChange) {
            onOpenChange(false);
        } else {
            router.back();
        }
    }

    // When used as a dialog (modal interception)
    if (asDialog) {
        return (
            <InviteFormDialogContent
                open={open ?? true}
                onOpenChange={onOpenChange ?? (() => router.back())}
                onSuccess={handleSuccess}
            />
        );
    }

    // Standalone page variant
    return <InviteFormPageContent onSuccess={handleSuccess} />;
}

function InviteFormDialogContent({
    open,
    onOpenChange,
    onSuccess,
}: {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    onSuccess: () => void;
}) {
    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="sm:max-w-2xl">
                <DialogHeader>
                    <DialogTitle>Invite Users</DialogTitle>
                    <DialogDescription>
                        Send invitation emails to join your school. You can choose a role for each
                        person.
                    </DialogDescription>
                </DialogHeader>
                <InviteFormContent onSuccess={onSuccess} />
            </DialogContent>
        </Dialog>
    );
}

function InviteFormPageContent({ onSuccess }: { onSuccess: () => void }) {
    const router = useRouter();

    function handleSuccess() {
        onSuccess();
        router.push("/admins/invitations");
    }

    return (
        <div className="mx-auto flex w-full max-w-2xl flex-col gap-4 p-6">
            <div>
                <h1 className="text-2xl font-semibold tracking-tight">Invite Users</h1>
                <p className="text-muted-foreground mt-1 text-sm">
                    Send invitation emails to join your school. You can choose a role for each
                    person.
                </p>
            </div>
            <div className="bg-card border-border/40 rounded-lg border p-4">
                <InviteFormContent onSuccess={handleSuccess} />
            </div>
        </div>
    );
}
