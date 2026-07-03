/**
 * Inline class selector for the preview stage.
 *
 * Renders a Select dropdown for choosing a class when the original
 * class match was invalid or needs to be re-assigned.
 */

"use client";

import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import type { ClassMatchRecord } from "@/lib/import-data/matching";

// ─── Props ─────────────────────────────────────────────────────────────────

interface ClassSelectorProps {
    classes: ClassMatchRecord[];
    value: string;
    onChange: (classId: string) => void;
    placeholder: string;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function ClassSelector({ classes, value, onChange, placeholder }: ClassSelectorProps) {
    return (
        <Select value={value} onValueChange={onChange}>
            <SelectTrigger className="h-7 text-xs">
                <SelectValue placeholder={placeholder} />
            </SelectTrigger>
            <SelectContent>
                {classes.map((c) => (
                    <SelectItem key={c.id} value={c.id}>
                        {c.display_label}
                    </SelectItem>
                ))}
            </SelectContent>
        </Select>
    );
}
