"use client";

import * as React from "react";
import { Copy, ArrowRightFromLine, Loader2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
    Dialog,
    DialogTrigger,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
    DialogFooter,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import {
    Select,
    SelectTrigger,
    SelectValue,
    SelectContent,
    SelectItem,
} from "@/components/ui/select";

import { useDuplicateDay, useCopyTimetableFromClass } from "@/features/cbc/hooks/use-cbc-timetable";
import type { CbcTimetableSlot, OperatingDay } from "@/features/cbc/types";

// ─── Props ────────────────────────────────────────────────────────────────

interface CbcBulkActionsProps {
    classId: string;
    academicYearId: string;
    operatingDays: OperatingDay[];
    slots: CbcTimetableSlot[];
}

// ─── Component ────────────────────────────────────────────────────────────

export function CbcBulkActions({
    classId,
    academicYearId,
    operatingDays,
    slots,
}: CbcBulkActionsProps) {
    // ── Duplicate day state ──────────────────────────────────────────
    const [duplicateOpen, setDuplicateOpen] = React.useState(false);
    const [sourceDay, setSourceDay] = React.useState<number | null>(null);
    const [targetDays, setTargetDays] = React.useState<number[]>([]);
    const { mutateAsync: duplicate, isPending: isDuplicating } = useDuplicateDay(classId);

    // ── Copy from class state ────────────────────────────────────────
    const [copyOpen, setCopyOpen] = React.useState(false);
    const [sourceClassId, setSourceClassId] = React.useState("");
    const { mutateAsync: copyFromClass, isPending: isCopying } = useCopyTimetableFromClass(classId);

    // ── Days with slots (for source selection) ───────────────────────
    const populatedDays = React.useMemo(() => {
        const days = new Set<number>();
        for (const slot of slots) {
            days.add(slot.day_of_week);
        }
        return operatingDays.filter((d) => days.has(d.value));
    }, [slots, operatingDays]);

    // ── Available target days (days that aren't the source) ──────────
    const availableTargetDays = React.useMemo(() => {
        return operatingDays.filter((d) => d.value !== sourceDay);
    }, [operatingDays, sourceDay]);

    // ── Toggle target day ────────────────────────────────────────────
    const handleToggleTargetDay = (day: number) => {
        setTargetDays((prev) =>
            prev.includes(day) ? prev.filter((d) => d !== day) : [...prev, day]
        );
    };

    // ── Handle duplicate ─────────────────────────────────────────────
    const handleDuplicate = async () => {
        if (!sourceDay || targetDays.length === 0) return;
        await duplicate({
            source_day: sourceDay,
            target_days: targetDays,
            academic_year_id: academicYearId,
            class_id: classId,
        });
        setDuplicateOpen(false);
        setTargetDays([]);
    };

    // ── Handle copy from class ───────────────────────────────────────
    const handleCopyFromClass = async () => {
        if (!sourceClassId) return;
        await copyFromClass({
            source_class_id: sourceClassId,
            academic_year_id: academicYearId,
            target_class_id: classId,
        });
        setCopyOpen(false);
        setSourceClassId("");
    };

    return (
        <div className="flex items-center gap-1.5">
            {/* ── Duplicate day ──────────────────────────────────────── */}
            <Dialog open={duplicateOpen} onOpenChange={setDuplicateOpen}>
                <DialogTrigger asChild>
                    <Button
                        variant="ghost"
                        size="sm"
                        className="h-8 text-xs"
                        disabled={populatedDays.length === 0}
                    >
                        <Copy className="mr-1 size-3.5" />
                        Duplicate day
                    </Button>
                </DialogTrigger>
                <DialogContent className="sm:max-w-md">
                    <DialogHeader>
                        <DialogTitle>Duplicate day to...</DialogTitle>
                        <DialogDescription>
                            Copy all slots from a source day to one or more target days. Any slot
                            that would conflict will be skipped.
                        </DialogDescription>
                    </DialogHeader>

                    <div className="space-y-4 py-2">
                        {/* Source day */}
                        <div>
                            <label className="mb-1.5 block text-sm font-medium">Source day</label>
                            <Select
                                value={sourceDay ? String(sourceDay) : ""}
                                onValueChange={(v) => {
                                    setSourceDay(Number(v));
                                    setTargetDays([]);
                                }}
                            >
                                <SelectTrigger className="w-full">
                                    <SelectValue placeholder="Select day to copy from" />
                                </SelectTrigger>
                                <SelectContent>
                                    {populatedDays.map((d) => (
                                        <SelectItem key={d.value} value={String(d.value)}>
                                            {d.label} (
                                            {slots.filter((s) => s.day_of_week === d.value).length}{" "}
                                            slots)
                                        </SelectItem>
                                    ))}
                                </SelectContent>
                            </Select>
                        </div>

                        {/* Target days */}
                        {sourceDay && (
                            <div>
                                <label className="mb-1.5 block text-sm font-medium">
                                    Target days
                                </label>
                                <div className="flex flex-wrap gap-2">
                                    {availableTargetDays.map((d) => {
                                        const selected = targetDays.includes(d.value);
                                        return (
                                            <button
                                                key={d.value}
                                                type="button"
                                                onClick={() => handleToggleTargetDay(d.value)}
                                                className={`rounded-md px-3 py-1.5 text-xs font-medium transition-colors ${
                                                    selected
                                                        ? "bg-teal-500 text-white"
                                                        : "bg-secondary text-secondary-foreground hover:bg-teal-100"
                                                }`}
                                                aria-pressed={selected}
                                            >
                                                {d.short_label}
                                            </button>
                                        );
                                    })}
                                </div>
                                {targetDays.length === 0 && (
                                    <p className="text-muted-foreground mt-1 text-xs">
                                        Select at least one target day
                                    </p>
                                )}
                            </div>
                        )}
                    </div>

                    <DialogFooter>
                        <Button variant="outline" onClick={() => setDuplicateOpen(false)}>
                            Cancel
                        </Button>
                        <Button
                            onClick={handleDuplicate}
                            disabled={!sourceDay || targetDays.length === 0 || isDuplicating}
                            className="bg-teal-600 text-white hover:bg-teal-700"
                        >
                            {isDuplicating ? (
                                <>
                                    <Loader2 className="mr-1.5 size-3.5 animate-spin" />
                                    Duplicating...
                                </>
                            ) : (
                                "Duplicate"
                            )}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            {/* ── Copy from another class ────────────────────────────── */}
            <Dialog open={copyOpen} onOpenChange={setCopyOpen}>
                <DialogTrigger asChild>
                    <Button variant="ghost" size="sm" className="h-8 text-xs">
                        <ArrowRightFromLine className="mr-1 size-3.5" />
                        Copy from class
                    </Button>
                </DialogTrigger>
                <DialogContent className="sm:max-w-md">
                    <DialogHeader>
                        <DialogTitle>Copy timetable from another class</DialogTitle>
                        <DialogDescription>
                            Copy all slots from another class in the same academic year. Conflicts
                            will be skipped per-slot.
                        </DialogDescription>
                    </DialogHeader>

                    <div className="py-2">
                        <label className="mb-1.5 block text-sm font-medium">Source class</label>
                        <Input
                            type="text"
                            placeholder="Enter source class ID"
                            value={sourceClassId}
                            onChange={(e) => setSourceClassId(e.target.value)}
                        />
                        <p className="text-muted-foreground mt-1 text-xs">
                            Paste the class ID to copy from. A class selector will be available once
                            the API supports it.
                        </p>
                    </div>

                    <DialogFooter>
                        <Button variant="outline" onClick={() => setCopyOpen(false)}>
                            Cancel
                        </Button>
                        <Button
                            onClick={handleCopyFromClass}
                            disabled={!sourceClassId || isCopying}
                            className="bg-teal-600 text-white hover:bg-teal-700"
                        >
                            {isCopying ? (
                                <>
                                    <Loader2 className="mr-1.5 size-3.5 animate-spin" />
                                    Copying...
                                </>
                            ) : (
                                "Copy"
                            )}
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </div>
    );
}
