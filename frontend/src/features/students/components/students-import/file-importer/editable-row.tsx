"use client";

import { Edit, Save, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { cn } from "@/lib/utils";
import { type StagedStudentRecord } from "./types";
import * as React from "react";

import { StatusBadge } from "./status-badge";

export function EditableRow({
    record,
    onSave,
}: {
    record: StagedStudentRecord;
    onSave: (updated: StagedStudentRecord) => void;
}) {
    const [editing, setEditing] = React.useState(false);
    const [fullName, setFullName] = React.useState(record.payload.full_name);
    const [gender, setGender] = React.useState(record.payload.gender ?? "none");
    const [upi, setUpi] = React.useState(record.payload.upi_number ?? "");
    const [knec, setKnec] = React.useState(record.payload.knec_assessment_number ?? "");
    const [dob, setDob] = React.useState(record.payload.date_of_birth ?? "");
    const [saving, setSaving] = React.useState(false);

    const handleSave = React.useCallback(() => {
        setSaving(true);
        const updated: StagedStudentRecord = {
            ...record,
            payload: {
                ...record.payload,
                full_name: fullName.trim(),
                gender: gender === "none" ? undefined : gender,
                upi_number: upi || undefined,
                knec_assessment_number: knec || undefined,
                date_of_birth: dob || undefined,
            },
            // Clear previous errors — re-validation happens in parent
            errors: [],
        };
        onSave(updated);
        setEditing(false);
        setSaving(false);
    }, [record, fullName, gender, upi, knec, dob, onSave]);

    const handleCancel = React.useCallback(() => {
        setFullName(record.payload.full_name);
        setGender(record.payload.gender ?? "none");
        setUpi(record.payload.upi_number ?? "");
        setKnec(record.payload.knec_assessment_number ?? "");
        setDob(record.payload.date_of_birth ?? "");
        setEditing(false);
    }, [record]);

    // Enter key saves, Escape cancels
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
                <div className="min-w-0 flex-1">
                    <Input
                        value={fullName}
                        onChange={(e) => setFullName(e.target.value)}
                        onKeyDown={handleKeyDown}
                        className="h-7 text-xs"
                        autoFocus
                    />
                </div>
                <div className="w-20 shrink-0">
                    <Select value={gender} onValueChange={setGender}>
                        <SelectTrigger className="h-7 text-xs">
                            <SelectValue placeholder="-" />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectItem value="none">-</SelectItem>
                            <SelectItem value="M">Male</SelectItem>
                            <SelectItem value="F">Female</SelectItem>
                        </SelectContent>
                    </Select>
                </div>
                <div className="w-24 shrink-0">
                    <Input
                        value={upi}
                        onChange={(e) => setUpi(e.target.value)}
                        onKeyDown={handleKeyDown}
                        className="h-7 text-xs"
                        placeholder="-"
                    />
                </div>
                <div className="w-24 shrink-0">
                    <Input
                        value={knec}
                        onChange={(e) => setKnec(e.target.value)}
                        onKeyDown={handleKeyDown}
                        className="h-7 text-xs"
                        placeholder="-"
                    />
                </div>
                <div className="w-32 shrink-0">
                    <Input
                        value={dob}
                        onChange={(e) => setDob(e.target.value)}
                        onKeyDown={handleKeyDown}
                        className="h-7 text-xs"
                        placeholder="-"
                        type="date"
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
            <div className="min-w-0 flex-1 truncate">{record.payload.full_name}</div>
            <div className="text-muted-foreground w-20 shrink-0">
                {record.payload.gender ?? "-"}
            </div>
            <div className="text-muted-foreground w-24 shrink-0 truncate">
                {record.payload.upi_number ?? "-"}
            </div>
            <div className="text-muted-foreground w-24 shrink-0 truncate">
                {record.payload.knec_assessment_number ?? "-"}
            </div>
            <div className="text-muted-foreground w-32 shrink-0">
                {record.payload.date_of_birth ?? "-"}
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
