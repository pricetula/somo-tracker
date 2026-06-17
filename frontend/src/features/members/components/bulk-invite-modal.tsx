/**
 * Bulk Invite Modal — send Stytch invitation emails to multiple people.
 *
 * Accepts a list of email + name entries and sends them all at once.
 */

"use client";

import * as React from "react";
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
    DialogClose,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { X, Plus } from "lucide-react";

import { useBulkInvite } from "@/features/members/hooks/use-members";

// ─── Types ─────────────────────────────────────────────────────────────────

interface InviteRow {
    key: string;
    email: string;
    first_name: string;
    last_name: string;
}

interface BulkInviteModalProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    role: "TEACHER" | "SUPPORT_STAFF";
}

// ─── Component ─────────────────────────────────────────────────────────────

export function BulkInviteModal({ open, onOpenChange, role }: BulkInviteModalProps) {
    const [rows, setRows] = React.useState<InviteRow[]>([
        { key: "1", email: "", first_name: "", last_name: "" },
    ]);
    const [sending, setSending] = React.useState(false);

    const bulkInvite = useBulkInvite();

    // Reset state when modal opens — handle via onOpenChange callback
    function handleOpenChange(newOpen: boolean) {
        if (!newOpen) {
            setRows([{ key: "1", email: "", first_name: "", last_name: "" }]);
            setSending(false);
        }
        onOpenChange(newOpen);
    }

    function addRow() {
        setRows((prev) => [
            ...prev,
            {
                key: crypto.randomUUID?.() ?? String(Date.now()),
                email: "",
                first_name: "",
                last_name: "",
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
            await bulkInvite.mutateAsync({
                role,
                invites: validRows.map((r) => ({
                    email: r.email.trim(),
                    first_name: r.first_name.trim(),
                    last_name: r.last_name.trim(),
                })),
            });
            handleOpenChange(false);
        } catch {
            // Error toast is handled by the mutation
        } finally {
            setSending(false);
        }
    }

    const validCount = rows.filter((r) => r.email.trim() !== "").length;
    const roleLabel = role === "TEACHER" ? "Teachers" : "Staff";

    return (
        <Dialog open={open} onOpenChange={handleOpenChange}>
            <DialogContent className="sm:max-w-xl">
                <DialogHeader>
                    <DialogTitle>Invite {roleLabel}</DialogTitle>
                    <DialogDescription>
                        Send invitation emails to join as {role.toLowerCase()}.
                    </DialogDescription>
                </DialogHeader>

                <div className="max-h-[400px] space-y-3 overflow-y-auto p-4">
                    {/* Header */}
                    <div className="grid grid-cols-[1fr_1fr_1fr_32px] gap-2 px-1">
                        <span className="text-muted-foreground text-xs font-medium">
                            First Name
                        </span>
                        <span className="text-muted-foreground text-xs font-medium">Last Name</span>
                        <span className="text-muted-foreground text-xs font-medium">Email *</span>
                        <span />
                    </div>

                    {/* Rows */}
                    {rows.map((row) => (
                        <div
                            key={row.key}
                            className="grid grid-cols-[1fr_1fr_1fr_32px] items-start gap-2"
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
                            : "Add at least one email"}
                    </p>
                    <div className="flex gap-3">
                        <DialogClose asChild>
                            <Button variant="outline" type="button" disabled={sending}>
                                Cancel
                            </Button>
                        </DialogClose>
                        <Button onClick={handleSend} disabled={validCount === 0 || sending}>
                            {sending ? "Sending..." : `Send Invitations`}
                        </Button>
                    </div>
                </div>
            </DialogContent>
        </Dialog>
    );
}
