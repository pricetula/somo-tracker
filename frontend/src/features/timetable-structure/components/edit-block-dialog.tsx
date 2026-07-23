/**
 * EditBlockDialog — edit or delete a time block.
 * All changes apply to ALL weekdays (Mon–Fri) automatically.
 */

"use client";

import { useState } from "react";
import { Loader2, Pencil, Trash2, AlertTriangle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog";
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
} from "@/components/ui/alert-dialog";

import type { TimeBlock, UpdateTimeBlockPayload } from "@/lib/api/timetable-structure";

// ─── Props ─────────────────────────────────────────────────────────────────

interface EditBlockDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    block: TimeBlock;
    academicYearID: string;
    isUpdatePending: boolean;
    isDeletePending: boolean;
    onUpdate: (payload: UpdateTimeBlockPayload) => void;
    onDelete: () => void;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function EditBlockDialog({
    open,
    onOpenChange,
    block,
    academicYearID,
    isUpdatePending,
    isDeletePending,
    onUpdate,
    onDelete,
}: EditBlockDialogProps) {
    const [periodName, setPeriodName] = useState(block.period_name);
    const [startTime, setStartTime] = useState(block.start_time.slice(0, 5));
    const [endTime, setEndTime] = useState(block.end_time.slice(0, 5));
    const [isBreak, setIsBreak] = useState(block.is_break);
    const [shiftFollowing, setShiftFollowing] = useState(false);

    const [showDeleteAlert, setShowDeleteAlert] = useState(false);

    // Sync state when block changes
    const handleOpenChange = (open: boolean) => {
        if (open) {
            setPeriodName(block.period_name);
            setStartTime(block.start_time.slice(0, 5));
            setEndTime(block.end_time.slice(0, 5));
            setIsBreak(block.is_break);
            setShiftFollowing(false);
            setShowDeleteAlert(false);
        }
        onOpenChange(open);
    };

    const timeError = startTime >= endTime;

    const handleSave = (e: React.FormEvent) => {
        e.preventDefault();
        if (timeError) return;

        onUpdate({
            day_of_week: block.day_of_week,
            period_name: periodName.trim(),
            start_time: `${startTime}:00`,
            end_time: `${endTime}:00`,
            is_break: isBreak,
            academic_year_id: academicYearID,
            propagate: "all_days",
            shift_following: shiftFollowing,
        });
    };

    const confirmDelete = () => {
        setShowDeleteAlert(false);
        onDelete();
    };

    const hasChanges =
        periodName !== block.period_name ||
        startTime !== block.start_time.slice(0, 5) ||
        endTime !== block.end_time.slice(0, 5) ||
        isBreak !== block.is_break;

    const isPending = isUpdatePending || isDeletePending;

    return (
        <>
            <Dialog open={open} onOpenChange={handleOpenChange}>
                <DialogContent className="sm:max-w-md">
                    <DialogHeader>
                        <DialogTitle>Edit Period</DialogTitle>
                        <DialogDescription>
                            Changes apply to all weekdays (Mon–Fri).
                        </DialogDescription>
                    </DialogHeader>

                    <form onSubmit={handleSave} className="space-y-4">
                        {/* Period name */}
                        <div className="space-y-1.5">
                            <Label htmlFor="period-name">Period Name</Label>
                            <Input
                                id="period-name"
                                type="text"
                                value={periodName}
                                onChange={(e) => setPeriodName(e.target.value)}
                                placeholder="e.g. Lesson 1"
                            />
                        </div>

                        {/* Time range */}
                        <div className="flex items-end gap-3">
                            <div className="flex-1 space-y-1.5">
                                <Label htmlFor="start-time">Start Time</Label>
                                <Input
                                    id="start-time"
                                    type="time"
                                    value={startTime}
                                    onChange={(e) => setStartTime(e.target.value)}
                                    step={300}
                                />
                            </div>
                            <span className="text-muted-foreground pb-2">–</span>
                            <div className="flex-1 space-y-1.5">
                                <Label htmlFor="end-time">End Time</Label>
                                <Input
                                    id="end-time"
                                    type="time"
                                    value={endTime}
                                    onChange={(e) => setEndTime(e.target.value)}
                                    step={300}
                                />
                            </div>
                        </div>
                        {timeError && (
                            <p className="text-destructive text-xs">
                                End time must be after start time
                            </p>
                        )}

                        {/* Break toggle */}
                        <label className="text-muted-foreground flex cursor-pointer items-center gap-2">
                            <Checkbox
                                checked={isBreak}
                                onCheckedChange={(v) => setIsBreak(v === true)}
                            />
                            Break period (non-assignable)
                        </label>

                        {/* Shift option */}
                        <div className="bg-muted/20 space-y-2 rounded-lg border p-3">
                            <p className="text-foreground text-xs font-semibold">Options</p>
                            <label className="text-muted-foreground flex cursor-pointer items-start gap-2">
                                <Checkbox
                                    checked={shiftFollowing}
                                    onCheckedChange={(v) => setShiftFollowing(v === true)}
                                />
                                <span>
                                    <span className="font-medium">Shift following periods</span>
                                    <span className="block text-[11px]">
                                        Adjusts subsequent blocks on all days by the same time
                                        change
                                    </span>
                                </span>
                            </label>
                        </div>

                        <DialogFooter className="gap-2">
                            <Button
                                type="button"
                                variant="outline"
                                size="sm"
                                onClick={() => setShowDeleteAlert(true)}
                                disabled={isPending}
                                className="text-destructive hover:text-destructive border-destructive/30 hover:bg-destructive/10"
                            >
                                <Trash2 className="mr-1 h-3.5 w-3.5" />
                                Delete this period
                            </Button>
                            <div className="flex gap-2">
                                <Button
                                    type="button"
                                    variant="outline"
                                    onClick={() => onOpenChange(false)}
                                    disabled={isPending}
                                >
                                    Cancel
                                </Button>
                                <Button
                                    type="submit"
                                    disabled={isPending || !periodName.trim() || timeError}
                                >
                                    {isUpdatePending ? (
                                        <>
                                            <Loader2 className="mr-1.5 h-4 w-4 animate-spin" />
                                            Saving…
                                        </>
                                    ) : (
                                        <>
                                            <Pencil className="mr-1.5 h-4 w-4" />
                                            {hasChanges || shiftFollowing
                                                ? "Save Changes"
                                                : "No Changes"}
                                        </>
                                    )}
                                </Button>
                            </div>
                        </DialogFooter>
                    </form>
                </DialogContent>
            </Dialog>

            {/* Delete confirmation */}
            <AlertDialog
                open={showDeleteAlert}
                onOpenChange={(v) => !isPending && setShowDeleteAlert(v)}
            >
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle className="flex items-center gap-2">
                            <AlertTriangle className="text-destructive h-5 w-5" />
                            Delete &quot;{block.period_name}&quot; from all days?
                        </AlertDialogTitle>
                        <AlertDialogDescription>
                            This will remove the &quot;{block.period_name}&quot; period from Monday
                            through Friday. If any classes are assigned to these periods, the delete
                            will be blocked.
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel disabled={isPending}>Cancel</AlertDialogCancel>
                        <AlertDialogAction
                            onClick={confirmDelete}
                            disabled={isPending}
                            className="bg-destructive hover:bg-destructive/90"
                        >
                            {isDeletePending ? (
                                <>
                                    <Loader2 className="mr-1.5 h-4 w-4 animate-spin" />
                                    Deleting…
                                </>
                            ) : (
                                "Delete"
                            )}
                        </AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </>
    );
}
