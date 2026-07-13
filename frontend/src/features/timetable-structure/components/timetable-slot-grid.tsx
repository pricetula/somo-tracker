/**
 * TimetableSlotGrid — weekly grid for assigning classes to time blocks.
 *
 * Pick a class from the selector above the grid, then click cells to
 * assign that class (plus optional teacher/area/room) to any period.
 * Click an already-assigned cell to remove the assignment.
 */

"use client";

import { useState } from "react";
import { Plus } from "lucide-react";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import {
    useEnrichedSlotList,
    useCreateSlot,
    useDeleteSlot,
} from "../hooks/use-timetable-structure";
import { AddSlotDialog } from "./add-slot-dialog";
import { ClassCombobox } from "@/features/classes/components/class-combobox";

import type { TimeBlock, CreateSlotPayload } from "@/lib/api/timetable-structure";
import { getDayName, DAY_NAMES } from "@/lib/api/timetable-structure";
import { ApiError } from "@/lib/api/client";
import { toast } from "sonner";
import { getErrorMessage } from "@/lib/errors";

// ─── Props ─────────────────────────────────────────────────────────────────

interface TimetableSlotGridProps {
    blocks: TimeBlock[];
    academicYearID: string;
    isLoading: boolean;
}

// ─── Helpers ──────────────────────────────────────────────────────────────

function fmt(time: string): string {
    return time.slice(0, 5);
}

function groupByDay(blocks: TimeBlock[]): Map<number, TimeBlock[]> {
    const groups = new Map<number, TimeBlock[]>();
    for (let d = 1; d <= 5; d++) groups.set(d, []);
    for (const b of blocks) {
        groups.get(b.day_of_week)?.push(b);
    }
    for (const [, list] of groups) {
        list.sort((a, b) => a.start_time.localeCompare(b.start_time));
    }
    return groups;
}

const DAYS = [1, 2, 3, 4, 5];

// ─── Component ─────────────────────────────────────────────────────────────

export function TimetableSlotGrid({ blocks, academicYearID, isLoading }: TimetableSlotGridProps) {
    const [selectedClassID, setSelectedClassID] = useState("");

    const viewBy = selectedClassID ? { mode: "class" as const, id: selectedClassID } : undefined;
    const { data: enrichedData, isLoading: slotsLoading } = useEnrichedSlotList(
        academicYearID,
        viewBy
    );
    const createMutation = useCreateSlot();
    const deleteMutation = useDeleteSlot();

    const [dialogOpen, setDialogOpen] = useState(false);
    const [dialogStructureID, setDialogStructureID] = useState("");
    const [dialogPeriod, setDialogPeriod] = useState("");
    const [dialogDay, setDialogDay] = useState(1);
    const [dialogTimeRange, setDialogTimeRange] = useState("");

    const allSlots = enrichedData?.items ?? [];

    // Build a quick lookup: which structure blocks does the selected class occupy?
    const slotSet = new Set<string>();
    for (const s of allSlots) {
        slotSet.add(`${s.day_of_week}-${s.structure_id}`);
    }

    const dayGroups = groupByDay(blocks);

    // Rows from Monday's blocks (all days share the same structure after replication)
    const mondayBlocks = dayGroups.get(1) ?? [];

    const handleCellClick = (day: number, block: TimeBlock) => {
        if (block.is_break) return;
        const key = `${day}-${block.id}`;
        if (slotSet.has(key)) {
            const slot = allSlots.find((s) => s.day_of_week === day && s.structure_id === block.id);
            if (slot) {
                if (window.confirm(`Remove ${slot.class_name} from ${block.period_name}?`)) {
                    deleteMutation.mutate(slot.id, {
                        onError: (err) => toast.error(getErrorMessage(err)),
                    });
                }
            }
            return;
        }
        if (!selectedClassID) return;
        setDialogStructureID(block.id);
        setDialogPeriod(block.period_name);
        setDialogDay(day);
        setDialogTimeRange(`${fmt(block.start_time)} – ${fmt(block.end_time)}`);
        setDialogOpen(true);
    };

    const handleCreateSlot = (payload: CreateSlotPayload) => {
        createMutation.mutate(payload, {
            onSuccess: () => setDialogOpen(false),
            onError: (err) => {
                if (err instanceof ApiError) toast.error(err.message);
            },
        });
    };

    if (isLoading || slotsLoading) {
        return (
            <div className="space-y-3">
                <h3 className="text-foreground text-sm font-semibold">Class Assignments</h3>
                <Skeleton className="h-64 w-full rounded-lg" />
            </div>
        );
    }

    if (blocks.length === 0) return null;

    return (
        <div className="space-y-4">
            <div>
                <h3 className="text-foreground text-sm font-semibold">Class Assignments</h3>
                <p className="text-muted-foreground mt-0.5 text-xs">
                    Select a class below, then click any period cell to assign it.
                </p>
            </div>

            {/* Class selector */}
            <div className="max-w-xs">
                <ClassCombobox
                    value={selectedClassID}
                    onChange={(v) => setSelectedClassID(v as string)}
                    placeholder="Select a class to assign..."
                />
            </div>

            {/* Grid */}
            <div className="border-border/40 max-h-[600px] overflow-auto rounded-lg border">
                <table className="w-full min-w-[500px] text-sm">
                    <thead>
                        <tr className="bg-muted/20">
                            <th className="bg-muted/20 text-muted-foreground border-border/40 sticky top-0 z-10 w-[160px] border-b px-3 py-2 text-left text-xs font-semibold">
                                Period
                            </th>
                            {DAYS.map((day) => (
                                <th
                                    key={day}
                                    className="bg-muted/20 text-muted-foreground border-border/40 border-border/40 sticky top-0 z-10 border-b border-l px-3 py-2 text-left text-xs font-semibold"
                                >
                                    {getDayName(day).slice(0, 3)}
                                </th>
                            ))}
                        </tr>
                    </thead>
                    <tbody>
                        {mondayBlocks.map((refBlock, rowIdx) => {
                            const isBreak = refBlock.is_break;
                            return (
                                <tr key={refBlock.id} className={cn(isBreak && "bg-muted/10")}>
                                    <td className="border-border/20 text-foreground border-b px-3 py-2.5 text-xs font-medium">
                                        <span className="block leading-tight">
                                            {refBlock.period_name}
                                        </span>
                                        <span className="text-muted-foreground block text-[11px]">
                                            {fmt(refBlock.start_time)} – {fmt(refBlock.end_time)}
                                        </span>
                                    </td>
                                    {DAYS.map((day) => {
                                        const dayBlocks = dayGroups.get(day) ?? [];
                                        const block = dayBlocks[rowIdx];
                                        const key = block ? `${day}-${block.id}` : "";
                                        const hasSlot = block ? slotSet.has(key) : false;
                                        const slot = hasSlot
                                            ? allSlots.find(
                                                  (s) =>
                                                      s.day_of_week === day &&
                                                      s.structure_id === block!.id
                                              )
                                            : null;

                                        return (
                                            <td
                                                key={key || `${day}-${rowIdx}`}
                                                className={cn(
                                                    "border-border/20 border-border/20 border-b border-l px-3 py-2.5",
                                                    !isBreak &&
                                                        "hover:bg-primary/5 cursor-pointer transition-colors",
                                                    hasSlot && "hover:bg-destructive/5"
                                                )}
                                                onClick={() =>
                                                    block && !isBreak && handleCellClick(day, block)
                                                }
                                            >
                                                {isBreak ? (
                                                    <span className="text-muted-foreground text-[11px] italic">
                                                        Break
                                                    </span>
                                                ) : hasSlot && slot ? (
                                                    <div className="text-xs">
                                                        <span className="text-foreground block truncate font-medium">
                                                            {slot.class_name}
                                                        </span>
                                                        {slot.teacher_name && (
                                                            <span className="text-muted-foreground block truncate text-[11px]">
                                                                {slot.teacher_name}
                                                            </span>
                                                        )}
                                                        {slot.learning_area_name && (
                                                            <span className="text-muted-foreground block truncate text-[11px]">
                                                                {slot.learning_area_name}
                                                            </span>
                                                        )}
                                                    </div>
                                                ) : (
                                                    <span className="text-muted-foreground/40 flex items-center gap-1 text-xs">
                                                        <Plus className="h-3 w-3" />
                                                        {selectedClassID
                                                            ? "Assign"
                                                            : "Select class"}
                                                    </span>
                                                )}
                                            </td>
                                        );
                                    })}
                                </tr>
                            );
                        })}
                    </tbody>
                </table>
            </div>

            <AddSlotDialog
                open={dialogOpen}
                onOpenChange={setDialogOpen}
                structureID={dialogStructureID}
                periodName={dialogPeriod}
                dayName={DAY_NAMES[dialogDay] ?? ""}
                timeRange={dialogTimeRange}
                academicYearID={academicYearID}
                isPending={createMutation.isPending}
                classID={selectedClassID}
                onSubmit={handleCreateSlot}
            />
        </div>
    );
}
