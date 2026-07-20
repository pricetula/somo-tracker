/**
 * TimetableSlotGrid — weekly grid for assigning classes to time blocks.
 *
 * Hover a period name to edit/delete (changes apply to all weekdays).
 * Click cells to assign/remove class slots.
 */

"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Plus, Pencil } from "lucide-react";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import {
    useEnrichedSlotList,
    useCreateSlot,
    useDeleteSlot,
    useUpdateTimeBlock,
    useDeleteTimeBlocksByName,
} from "../hooks/use-timetable-structure";
import { AddSlotDialog } from "./add-slot-dialog";
import { EditBlockDialog } from "./edit-block-dialog";
import { ClassCombobox } from "@/features/classes/components/class-combobox";
import { useClassList } from "@/features/classes/hooks/use-classes";

import type {
    TimeBlock,
    CreateSlotPayload,
    UpdateTimeBlockPayload,
} from "@/lib/api/timetable-structure";
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
    const router = useRouter();
    const [selectedClassID, setSelectedClassID] = useState("");

    // Derive the effective class ID — defaults to the first class when data loads
    const { data: classData } = useClassList();
    const classItems = classData?.items ?? [];
    const resolvedClassID = selectedClassID || classItems[0]?.value || "";

    const { data: enrichedData, isLoading: slotsLoading } = useEnrichedSlotList(
        academicYearID,
        resolvedClassID ? { classId: resolvedClassID } : undefined
    );
    const createMutation = useCreateSlot();
    const deleteSlotMutation = useDeleteSlot();
    const updateBlockMutation = useUpdateTimeBlock();
    const deleteBlocksByNameMutation = useDeleteTimeBlocksByName();

    // Slot assignment dialog state
    const [assignOpen, setAssignOpen] = useState(false);
    const [assignStructureID, setAssignStructureID] = useState("");
    const [assignPeriod, setAssignPeriod] = useState("");
    const [assignDay, setAssignDay] = useState(1);
    const [assignTimeRange, setAssignTimeRange] = useState("");

    // Block edit dialog state
    const [editOpen, setEditOpen] = useState(false);
    const [editBlock, setEditBlock] = useState<TimeBlock | null>(null);

    const allSlots = enrichedData?.items ?? [];

    // Build a quick lookup: which structure blocks does the selected class occupy?
    const slotSet = new Set<string>();
    for (const s of allSlots) {
        slotSet.add(`${s.day_of_week}-${s.structure_id}`);
    }

    const dayGroups = groupByDay(blocks);

    // Rows from Monday's blocks (all days share the same structure after replication)
    const mondayBlocks = dayGroups.get(1) ?? [];

    // ── Slot assignment handlers ──

    const handleCellClick = (day: number, block: TimeBlock) => {
        if (block.is_break) return;
        const key = `${day}-${block.id}`;
        if (slotSet.has(key)) {
            const slot = allSlots.find((s) => s.day_of_week === day && s.structure_id === block.id);
            if (slot) {
                if (window.confirm(`Remove ${slot.class_name} from ${block.period_name}?`)) {
                    deleteSlotMutation.mutate(slot.id, {
                        onError: (err) => toast.error(getErrorMessage(err)),
                    });
                }
            }
            return;
        }
        if (!resolvedClassID) return;
        setAssignStructureID(block.id);
        setAssignPeriod(block.period_name);
        setAssignDay(day);
        setAssignTimeRange(`${fmt(block.start_time)} – ${fmt(block.end_time)}`);
        setAssignOpen(true);
    };

    const handleCreateSlot = (payload: CreateSlotPayload) => {
        createMutation.mutate(payload, {
            onSuccess: () => setAssignOpen(false),
            onError: (err) => {
                if (err instanceof ApiError) toast.error(err.message);
            },
        });
    };

    // ── Block edit/delete handlers (always all days) ──

    const handleEditClick = (block: TimeBlock) => {
        setEditBlock(block);
        setEditOpen(true);
    };

    const handleUpdateBlock = (payload: UpdateTimeBlockPayload) => {
        if (!editBlock) return;
        updateBlockMutation.mutate(
            { id: editBlock.id, ...payload },
            {
                onSuccess: () => setEditOpen(false),
                onError: (err) => toast.error(getErrorMessage(err)),
            }
        );
    };

    const handleDeleteBlock = () => {
        if (!editBlock) return;
        deleteBlocksByNameMutation.mutate(
            { periodName: editBlock.period_name, academicYearID },
            {
                onSuccess: () => setEditOpen(false),
                onError: (err) => toast.error(getErrorMessage(err)),
            }
        );
    };

    if (isLoading || slotsLoading) {
        return (
            <div className="space-y-3">
                <h3 className="text-foreground font-semibold">Class Assignments</h3>
                <Skeleton className="h-64 w-full rounded-lg" />
            </div>
        );
    }

    if (blocks.length === 0) return null;

    return (
        <div className="space-y-4">
            <div>
                <h3 className="text-foreground font-semibold">Class Assignments</h3>
                <p className="text-muted-foreground mt-0.5 text-xs">
                    Select a class below, then click any period cell to assign it. Hover a period
                    name to edit or delete (applies to all weekdays).
                </p>
            </div>

            {/* Class selector */}
            <div className="max-w-xs">
                <ClassCombobox
                    value={resolvedClassID}
                    onChange={(v) => setSelectedClassID(v as string)}
                    placeholder="Select a class to assign..."
                    onCreateItem={() => router.push("/classes/add")}
                />
            </div>

            {/* Grid */}
            <div className="border-border/40 max-h-[600px] overflow-auto rounded-lg border">
                <table className="w-full min-w-[500px]">
                    <thead>
                        <tr className="bg-muted/20">
                            <th className="bg-muted/20 text-muted-foreground border-border/40 sticky top-0 z-10 w-[170px] border-b px-3 py-2 text-left text-xs font-semibold">
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
                                <tr
                                    key={refBlock.id}
                                    className={cn(isBreak && "bg-muted/10", "group")}
                                >
                                    {/* Period column with inline edit button */}
                                    <td className="border-border/20 text-foreground relative border-b px-3 py-2.5 text-xs font-medium">
                                        <div className="flex items-start justify-between gap-1">
                                            <div className="min-w-0 flex-1">
                                                <span className="block truncate leading-tight">
                                                    {refBlock.period_name}
                                                </span>
                                                <span className="text-muted-foreground block text-[11px]">
                                                    {fmt(refBlock.start_time)} –{" "}
                                                    {fmt(refBlock.end_time)}
                                                </span>
                                            </div>
                                            {/* Edit button — visible on row hover */}
                                            <div className="flex shrink-0 items-center gap-0.5 opacity-0 transition-opacity group-hover:opacity-100">
                                                <button
                                                    type="button"
                                                    onClick={() => handleEditClick(refBlock)}
                                                    className="hover:bg-primary/10 text-muted-foreground hover:text-foreground rounded p-1 transition-colors"
                                                    aria-label="Edit period"
                                                    title="Edit period"
                                                >
                                                    <Pencil className="h-3.5 w-3.5" />
                                                </button>
                                            </div>
                                        </div>
                                    </td>

                                    {/* Day cells */}
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
                                                        {resolvedClassID
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

            {/* Slot assignment dialog */}
            <AddSlotDialog
                open={assignOpen}
                onOpenChange={setAssignOpen}
                structureID={assignStructureID}
                periodName={assignPeriod}
                dayName={DAY_NAMES[assignDay] ?? ""}
                timeRange={assignTimeRange}
                academicYearID={academicYearID}
                isPending={createMutation.isPending}
                classID={resolvedClassID}
                onSubmit={handleCreateSlot}
            />

            {/* Block edit dialog */}
            {editBlock && (
                <EditBlockDialog
                    open={editOpen}
                    onOpenChange={setEditOpen}
                    block={editBlock}
                    academicYearID={academicYearID}
                    isUpdatePending={updateBlockMutation.isPending}
                    isDeletePending={deleteBlocksByNameMutation.isPending}
                    onUpdate={handleUpdateBlock}
                    onDelete={handleDeleteBlock}
                />
            )}
        </div>
    );
}
