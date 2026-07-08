"use client";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { DatePicker } from "@/components/ui/date-picker";
import { X } from "lucide-react";

// ─── Types ─────────────────────────────────────────────────────────────────

export interface StudentEntry {
    key: string;
    fullName: string;
    gender: string;
    dateOfBirth: string;
    upiNumber: string;
    knecNumber: string;
}

interface StudentEntryRowProps {
    entry: StudentEntry;
    index: number;
    isSubmitting: boolean;
    fieldErrors: Record<string, string[]>;
    canRemove: boolean;
    onUpdate: (patch: Partial<StudentEntry>) => void;
    onRemove: () => void;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function StudentEntryRow({
    entry,
    index,
    isSubmitting,
    fieldErrors,
    canRemove,
    onUpdate,
    onRemove,
}: StudentEntryRowProps) {
    return (
        <div className="bg-muted/30 space-y-4 rounded-md p-4">
            {/* Header row with entry number and remove button */}
            <div className="flex items-center justify-between gap-2">
                <p className="text-muted-foreground text-sm font-medium">Student {index + 1}</p>
                {canRemove && (
                    <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        className="size-6"
                        onClick={onRemove}
                        disabled={isSubmitting}
                        aria-label={`Remove student ${index + 1}`}
                    >
                        <X className="size-4" />
                    </Button>
                )}
            </div>

            {/* Full Name */}
            <div className="space-y-1.5">
                <Label htmlFor={`full_name_${entry.key}`}>
                    Full Name <span className="text-destructive">*</span>
                </Label>
                <Input
                    id={`full_name_${entry.key}`}
                    value={entry.fullName}
                    onChange={(e) => onUpdate({ fullName: e.target.value })}
                    placeholder="e.g. John Kiprop"
                    disabled={isSubmitting}
                />
                {fieldErrors[`students.${index}.full_name`] && (
                    <p className="text-destructive text-xs">
                        {fieldErrors[`students.${index}.full_name`][0]}
                    </p>
                )}
            </div>

            {/* Gender + DOB row */}
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <div className="space-y-1.5">
                    <Label htmlFor={`gender_${entry.key}`}>Gender</Label>
                    <Select
                        value={entry.gender}
                        onValueChange={(v) => onUpdate({ gender: v })}
                        disabled={isSubmitting}
                    >
                        <SelectTrigger id={`gender_${entry.key}`}>
                            <SelectValue placeholder="Select gender" />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectItem value="M">Male</SelectItem>
                            <SelectItem value="F">Female</SelectItem>
                        </SelectContent>
                    </Select>
                </div>
                <div className="space-y-1.5">
                    <Label htmlFor={`dob_${entry.key}`}>Date of Birth</Label>
                    <DatePicker
                        id={`dob_${entry.key}`}
                        value={entry.dateOfBirth}
                        onChange={(v) => onUpdate({ dateOfBirth: v })}
                        placeholder="Select date"
                        disabled={isSubmitting}
                    />
                </div>
            </div>

            {/* UPI + KNEC row */}
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <div className="space-y-1.5">
                    <Label htmlFor={`upi_${entry.key}`}>UPI Number</Label>
                    <Input
                        id={`upi_${entry.key}`}
                        value={entry.upiNumber}
                        onChange={(e) => onUpdate({ upiNumber: e.target.value })}
                        placeholder="e.g. UP123456789"
                        disabled={isSubmitting}
                    />
                </div>
                <div className="space-y-1.5">
                    <Label htmlFor={`knec_${entry.key}`}>KNEC Assessment Number</Label>
                    <Input
                        id={`knec_${entry.key}`}
                        value={entry.knecNumber}
                        onChange={(e) => onUpdate({ knecNumber: e.target.value })}
                        placeholder="e.g. KNEC123456"
                        disabled={isSubmitting}
                    />
                </div>
            </div>
        </div>
    );
}
