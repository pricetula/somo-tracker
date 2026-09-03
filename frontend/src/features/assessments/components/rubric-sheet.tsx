"use client";

import { useState } from "react";
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

export interface RubricRow {
    student_id: string;
    student_name: string;
    indicator_id: string;
    awarded_level: "EE" | "ME" | "AE" | "BE" | null;
}

export interface RubricSheetProps {
    rows: RubricRow[];
    readOnly: boolean;
    onSave: (
        entries: { student_id: string; performance_indicator_id: string; awarded_level: string }[]
    ) => Promise<void> | void;
}

const LEVELS = [
    { value: "EE", label: "Exceeds Expectations (EE)" },
    { value: "ME", label: "Meets Expectations (ME)" },
    { value: "AE", label: "Approaching Expectations (AE)" },
    { value: "BE", label: "Below Expectations (BE)" },
];

export function RubricSheet({ rows, readOnly, onSave }: RubricSheetProps) {
    const [draft, setDraft] = useState<Record<string, string | null>>(() => {
        const map: Record<string, string | null> = {};
        for (const r of rows) map[`${r.student_id}:${r.indicator_id}`] = r.awarded_level;
        return map;
    });
    const [saving, setSaving] = useState(false);

    function handleChange(key: string, value: string) {
        setDraft((d) => ({ ...d, [key]: value === "" ? null : value }));
    }

    async function handleSave() {
        setSaving(true);
        try {
            const entries = rows.map((r) => ({
                student_id: r.student_id,
                performance_indicator_id: r.indicator_id,
                awarded_level: (draft[`${r.student_id}:${r.indicator_id}`] ?? "ME") as string,
            }));
            await onSave(entries);
        } finally {
            setSaving(false);
        }
    }

    return (
        <div className="space-y-4">
            <table className="w-full border-collapse text-sm">
                <thead>
                    <tr className="border-b">
                        <th className="py-2 pr-4 text-left">Student</th>
                        <th className="py-2 pr-4 text-left">Indicator</th>
                        <th className="py-2 pr-4 text-left">Level</th>
                    </tr>
                </thead>
                <tbody>
                    {rows.map((r) => (
                        <tr
                            key={`${r.student_id}:${r.indicator_id}`}
                            className="hover:bg-muted/50 border-b"
                        >
                            <td className="py-2 pr-4">{r.student_name}</td>
                            <td className="py-2 pr-4">{r.indicator_id.slice(0, 8)}</td>
                            <td className="py-2 pr-4">
                                <Select
                                    value={draft[`${r.student_id}:${r.indicator_id}`] ?? "ME"}
                                    disabled={readOnly}
                                    onValueChange={(v) =>
                                        handleChange(`${r.student_id}:${r.indicator_id}`, v)
                                    }
                                >
                                    <SelectTrigger className="w-48">
                                        <SelectValue />
                                    </SelectTrigger>
                                    <SelectContent>
                                        {LEVELS.map((l) => (
                                            <SelectItem key={l.value} value={l.value}>
                                                {l.label}
                                            </SelectItem>
                                        ))}
                                    </SelectContent>
                                </Select>
                            </td>
                        </tr>
                    ))}
                </tbody>
            </table>
            {!readOnly ? (
                <Button onClick={handleSave} disabled={saving}>
                    {saving ? "Saving..." : "Save rubric grades"}
                </Button>
            ) : null}
        </div>
    );
}
