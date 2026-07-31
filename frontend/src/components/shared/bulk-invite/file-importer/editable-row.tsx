"use client";

import { Edit, Save, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { type StagedInviteRecord } from "./types";
import * as React from "react";

import { StatusBadge } from "./status-badge";

export function EditableRow({
    record,
    onSave,
}: {
    record: StagedInviteRecord;
    onSave: (updated: StagedInviteRecord) => void;
}) {
    const [editing, setEditing] = React.useState(false);
    const [email, setEmail] = React.useState(record.email);
    const [fullName, setFullName] = React.useState(record.full_name);
    const [saving, setSaving] = React.useState(false);

    const handleSave = React.useCallback(() => {
        setSaving(true);
        const updated: StagedInviteRecord = {
            ...record,
            email: email.trim(),
            full_name: fullName.trim(),
            errors: [],
        };
        onSave(updated);
        setEditing(false);
        setSaving(false);
    }, [record, email, fullName, onSave]);

    const handleCancel = React.useCallback(() => {
        setEmail(record.email);
        setFullName(record.full_name);
        setEditing(false);
    }, [record]);

    const handleKeyDown = React.useCallback(
        (e: React.KeyboardEvent) => {
            if (e.key === "Enter" && !e.shiftKey) {
                e.preventDefault();
                handleSave();
            } else if (e.key === "Escape") {
                handleCancel();
            }
        },
        [handleSave, handleCancel]
    );

    const rowBase = cn(
        "flex items-center gap-2 px-2 py-1.5 ",
        record.status !== "valid" && !editing && "bg-destructive/5"
    );

    if (editing) {
        return (
            <div className={cn(rowBase, "bg-muted/30")}>
                <div className="min-w-0 flex-[2]">
                    <Input
                        value={email}
                        onChange={(e) => setEmail(e.target.value)}
                        onKeyDown={handleKeyDown}
                        className="h-7 text-xs"
                        autoFocus
                    />
                </div>
                <div className="min-w-0 flex-[2]">
                    <Input
                        value={fullName}
                        onChange={(e) => setFullName(e.target.value)}
                        onKeyDown={handleKeyDown}
                        className="h-7 text-xs"
                        placeholder="(optional)"
                    />
                </div>
                <div className="w-16 shrink-0">
                    <StatusBadge status={record.status} />
                </div>
                <div className="flex w-16 shrink-0 items-center gap-1">
                    <Button
                        variant="ghost"
                        size="icon"
                        className="size-6"
                        onClick={handleSave}
                        disabled={saving}
                    >
                        <Save className="size-3 text-emerald-500" />
                    </Button>
                    <Button variant="ghost" size="icon" className="size-6" onClick={handleCancel}>
                        <X className="text-muted-foreground size-3" />
                    </Button>
                </div>
            </div>
        );
    }

    return (
        <div className={cn(rowBase, "hover:bg-muted/30 transition-colors")}>
            <div className="min-w-0 flex-[2] truncate">
                {record.email || <span className="text-muted-foreground italic">empty</span>}
            </div>
            <div className="text-muted-foreground min-w-0 flex-[2] truncate">
                {record.full_name || <span className="italic">—</span>}
            </div>
            <div className="w-16 shrink-0">
                <StatusBadge status={record.status} />
            </div>
            <div className="flex w-16 shrink-0 items-center">
                <Button
                    variant="ghost"
                    size="icon"
                    className="size-6"
                    onClick={() => setEditing(true)}
                >
                    <Edit className="text-muted-foreground size-3" />
                </Button>
            </div>
        </div>
    );
}
