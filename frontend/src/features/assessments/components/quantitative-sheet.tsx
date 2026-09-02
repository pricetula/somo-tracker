"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table";

export interface ScoreRow {
    student_id: string;
    student_name: string;
    raw_score: number | null;
}

export interface QuantitativeSheetProps {
    maxPoints: number;
    rows: ScoreRow[];
    readOnly: boolean;
    onSave: (entries: { student_id: string; raw_score: number | null }[]) => Promise<void> | void;
}

export function QuantitativeSheet({ maxPoints, rows, readOnly, onSave }: QuantitativeSheetProps) {
    const [draft, setDraft] = useState<Record<string, number | null>>(() => {
        const map: Record<string, number | null> = {};
        for (const r of rows) map[r.student_id] = r.raw_score;
        return map;
    });
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState<string | null>(null);

    function handleChange(studentId: string, value: string) {
        if (value === "") {
            setDraft((d) => ({ ...d, [studentId]: null }));
            return;
        }
        const n = Number(value);
        if (Number.isFinite(n) && n >= 0 && n <= maxPoints) {
            setDraft((d) => ({ ...d, [studentId]: n }));
        }
    }

    async function handleSave() {
        setSaving(true);
        setError(null);
        try {
            const entries = rows.map((r) => ({
                student_id: r.student_id,
                raw_score: draft[r.student_id] ?? null,
            }));
            await onSave(entries);
        } catch (e) {
            setError(e instanceof Error ? e.message : "Failed to save scores");
        } finally {
            setSaving(false);
        }
    }

    return (
        <div className="space-y-4">
            <Table>
                <TableHeader>
                    <TableRow>
                        <TableHead>Student</TableHead>
                        <TableHead className="w-32">Score (/{maxPoints})</TableHead>
                        <TableHead className="w-24">%</TableHead>
                    </TableRow>
                </TableHeader>
                <TableBody>
                    {rows.map((r) => {
                        const v = draft[r.student_id];
                        const pct = v == null ? null : Math.round((v / maxPoints) * 1000) / 10;
                        return (
                            <TableRow key={r.student_id}>
                                <TableCell>{r.student_name}</TableCell>
                                <TableCell>
                                    <Input
                                        type="number"
                                        inputMode="decimal"
                                        min={0}
                                        max={maxPoints}
                                        step="0.5"
                                        value={v ?? ""}
                                        disabled={readOnly}
                                        onChange={(e) => handleChange(r.student_id, e.target.value)}
                                    />
                                </TableCell>
                                <TableCell className="text-muted-foreground">
                                    {pct ?? "—"}
                                </TableCell>
                            </TableRow>
                        );
                    })}
                </TableBody>
            </Table>
            {error ? <p className="text-destructive text-sm">{error}</p> : null}
            {!readOnly ? (
                <Button onClick={handleSave} disabled={saving}>
                    {saving ? "Saving..." : "Save scores"}
                </Button>
            ) : null}
        </div>
    );
}
