/**
 * BlueprintDialog — define a full day's block sequence in one dialog.
 * Each block is created for all 5 weekdays (Mon–Fri) on save.
 */

"use client";

import { useState } from "react";
import { Plus, X, Loader2, GripVertical } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Checkbox } from "@/components/ui/checkbox";
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog";

// ─── Helpers ──────────────────────────────────────────────────────────────

function addMinutes(time: string, minutes: number): string {
    const [h, m] = time.split(":").map(Number);
    const total = h * 60 + m + minutes;
    const nh = Math.floor(total / 60);
    const nm = total % 60;
    return `${String(nh).padStart(2, "0")}:${String(nm).padStart(2, "0")}`;
}

// ─── Types ─────────────────────────────────────────────────────────────────

interface DraftBlock {
    id: string;
    periodName: string;
    startTime: string;
    endTime: string;
    isBreak: boolean;
}

let nextId = 1;
function newId(): string {
    return `draft-${nextId++}`;
}

function createDraft(
    startTime: string,
    endTime: string,
    periodName: string,
    isBreak = false
): DraftBlock {
    return { id: newId(), periodName, startTime, endTime, isBreak };
}

// ─── Props ─────────────────────────────────────────────────────────────────

interface BlueprintDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    isPending: boolean;
    onSave: (
        blocks: { periodName: string; startTime: string; endTime: string; isBreak: boolean }[]
    ) => void;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function BlueprintDialog({ open, onOpenChange, isPending, onSave }: BlueprintDialogProps) {
    const [drafts, setDrafts] = useState<DraftBlock[]>([
        createDraft("08:00", addMinutes("08:00", 30), "Lesson 1"),
    ]);

    const handleOpenChange = (open: boolean) => {
        if (open) {
            setDrafts([createDraft("08:00", addMinutes("08:00", 30), "Lesson 1")]);
        }
        onOpenChange(open);
    };

    function addBlock() {
        const last = drafts[drafts.length - 1];
        const nextStart = last ? last.endTime : "08:00";
        const nextNum = drafts.filter((d) => !d.isBreak).length + 1;
        const nextEnd = addMinutes(nextStart, 30);
        setDrafts([...drafts, createDraft(nextStart, nextEnd, `Lesson ${nextNum}`)]);
    }

    function updateBlock(id: string, patch: Partial<DraftBlock>) {
        setDrafts(drafts.map((d) => (d.id === id ? { ...d, ...patch } : d)));
    }

    function removeBlock(id: string) {
        setDrafts(drafts.filter((d) => d.id !== id));
    }

    function canSave() {
        return (
            drafts.length > 0 && drafts.every((d) => d.periodName.trim() && d.startTime < d.endTime)
        );
    }

    const handleSave = () => {
        onSave(
            drafts.map((d) => ({
                periodName: d.periodName.trim(),
                startTime: d.startTime,
                endTime: d.endTime,
                isBreak: d.isBreak,
            }))
        );
    };

    return (
        <Dialog open={open} onOpenChange={handleOpenChange}>
            <DialogContent className="flex max-h-[85vh] flex-col sm:max-w-lg">
                <DialogHeader>
                    <DialogTitle>Master Blueprint</DialogTitle>
                    <DialogDescription>
                        Define the daily block sequence. Each block is created for all weekdays
                        (Mon–Fri).
                    </DialogDescription>
                </DialogHeader>

                <div className="flex-1 space-y-2 overflow-y-auto pr-1">
                    {drafts.map((d, i) => {
                        const timeError = d.startTime >= d.endTime;

                        return (
                            <div
                                key={d.id}
                                className="border-border/40 space-y-2 rounded-lg border p-3"
                            >
                                <div className="flex items-center gap-2">
                                    <GripVertical className="text-muted-foreground/30 h-4 w-4 shrink-0" />
                                    <span className="text-muted-foreground w-5 text-xs font-medium">
                                        {i + 1}
                                    </span>
                                    <Input
                                        type="text"
                                        value={d.periodName}
                                        onChange={(e) =>
                                            updateBlock(d.id, { periodName: e.target.value })
                                        }
                                        className="h-8 flex-1 text-sm"
                                        placeholder="Period name"
                                    />
                                    <button
                                        type="button"
                                        onClick={() => removeBlock(d.id)}
                                        className="hover:bg-destructive/10 rounded p-1 transition-colors"
                                        aria-label="Remove block"
                                    >
                                        <X className="text-muted-foreground h-3.5 w-3.5" />
                                    </button>
                                </div>

                                <div className="flex items-center gap-3 pl-11">
                                    <div className="flex items-center gap-1.5">
                                        <Input
                                            type="time"
                                            value={d.startTime}
                                            onChange={(e) =>
                                                updateBlock(d.id, { startTime: e.target.value })
                                            }
                                            className="h-8 w-[120px] text-xs"
                                            step={300}
                                        />
                                        <span className="text-muted-foreground text-xs">–</span>
                                        <Input
                                            type="time"
                                            value={d.endTime}
                                            onChange={(e) =>
                                                updateBlock(d.id, { endTime: e.target.value })
                                            }
                                            className="h-8 w-[120px] text-xs"
                                            step={300}
                                        />
                                    </div>

                                    <label className="text-muted-foreground flex cursor-pointer items-center gap-1.5 text-xs">
                                        <Checkbox
                                            checked={d.isBreak}
                                            onCheckedChange={(v) =>
                                                updateBlock(d.id, { isBreak: v === true })
                                            }
                                        />
                                        Break
                                    </label>

                                    {timeError && (
                                        <span className="text-destructive text-xs">
                                            End must be after start
                                        </span>
                                    )}
                                </div>
                            </div>
                        );
                    })}

                    <button
                        type="button"
                        onClick={addBlock}
                        className="border-border/40 text-muted-foreground hover:bg-muted/30 flex w-full items-center gap-2 rounded-lg border border-dashed p-3 text-sm transition-colors"
                    >
                        <Plus className="h-4 w-4" />
                        Add block
                    </button>
                </div>

                <DialogFooter className="border-border/20 border-t pt-4">
                    <Button
                        type="button"
                        variant="outline"
                        onClick={() => onOpenChange(false)}
                        disabled={isPending}
                    >
                        Cancel
                    </Button>
                    <Button type="button" onClick={handleSave} disabled={isPending || !canSave()}>
                        {isPending ? (
                            <>
                                <Loader2 className="mr-1.5 h-4 w-4 animate-spin" />
                                Saving…
                            </>
                        ) : (
                            "Save Blueprint"
                        )}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}
