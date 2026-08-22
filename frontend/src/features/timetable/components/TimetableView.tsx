"use client";

import { useState, useCallback } from "react";
import { Calendar, Plus, Settings, Lock } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
} from "@/components/ui/dialog";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { TimetableGrid } from "./TimetableGrid";
import { EmptyStateStructure } from "./EmptyStateStructure";
import { EmptyStateAllocation } from "./EmptyStateAllocation";
import { EmptyStateFiltered } from "./EmptyStateFiltered";
import { StructureBuilder } from "./StructureBuilder";
import { SlotAssignmentForm } from "./SlotAssignmentForm";
import { SlotConflictBanner } from "./SlotConflictBanner";
import { useTimetableGrid } from "../hooks/use-timetable-grid";
import {
    useTimetableClasses,
    useTimetableTeachers,
    useTimetableLearningAreas,
    useTimetableRooms,
} from "../hooks/use-timetable-data";
import { useCreateSlot, useUpdateSlot, useDeleteSlot, useBatchCreateTimeBlocks } from "../hooks";
import type { CreateSlotPayload, UpdateSlotPayload } from "../types";
import type {
    EnrichedSlot,
    CreateSlotPayload as APICreateSlotPayload,
    UpdateSlotPayload as APIUpdateSlotPayload,
} from "@/lib/api/timetable-structure";

interface TimetableViewProps {
    academicYearId: string;
    isReadOnly?: boolean;
}

export function TimetableView({ academicYearId, isReadOnly = false }: TimetableViewProps) {
    const [showStructureBuilder, setShowStructureBuilder] = useState(false);
    const [slotForm, setSlotForm] = useState<{
        open: boolean;
        editingSlot?: EnrichedSlot;
        structureId?: string;
    }>({ open: false });
    const [conflicts, setConflicts] = useState<
        Array<{
            type: "class" | "teacher" | "room";
            message: string;
            structureId: string;
            conflictingSlotId: string;
        }>
    >([]);
    const [selectedDate, setSelectedDate] = useState<string | undefined>(undefined);

    const { gridView, isLoading, error, timeBlocks, slots } = useTimetableGrid({
        academicYearId,
        filters: selectedDate ? { date: selectedDate } : {},
    });

    // Fetch reference data for slot assignment form
    const { data: classes = [] } = useTimetableClasses();
    const { data: teachers = [] } = useTimetableTeachers();
    const { data: learningAreas = [] } = useTimetableLearningAreas();
    const { data: rooms = [] } = useTimetableRooms(slots);

    const createSlotMutation = useCreateSlot();
    const updateSlotMutation = useUpdateSlot();
    const deleteSlotMutation = useDeleteSlot();
    const batchCreateBlocksMutation = useBatchCreateTimeBlocks();

    const handleCreateStructure = useCallback(
        async (
            periods: Array<{
                dayOfWeek: number;
                periodName: string;
                startTime: string;
                endTime: string;
                isBreak: boolean;
            }>
        ) => {
            const payload = {
                blocks: periods.map((p) => ({
                    day_of_week: p.dayOfWeek,
                    period_name: p.periodName,
                    start_time: p.startTime,
                    end_time: p.endTime,
                    is_break: p.isBreak,
                })),
            };
            await batchCreateBlocksMutation.mutateAsync(payload);
            setShowStructureBuilder(false);
        },
        [batchCreateBlocksMutation]
    );

    const handleCellAdd = useCallback(
        (structureId: string) => {
            if (!isReadOnly) {
                setSlotForm({ open: true, structureId });
            }
        },
        [isReadOnly]
    );

    const handleCellEdit = useCallback(
        (slotId: string) => {
            if (!isReadOnly) {
                const slot = slots.find((s) => s.id === slotId);
                if (slot) {
                    setSlotForm({ open: true, editingSlot: slot, structureId: slot.structure_id });
                }
            }
        },
        [slots, isReadOnly]
    );

    const handleCellDelete = useCallback(
        async (slotId: string) => {
            if (!isReadOnly && confirm("Delete this lesson assignment?")) {
                await deleteSlotMutation.mutateAsync(slotId);
            }
        },
        [deleteSlotMutation, isReadOnly]
    );

    const handleSlotSubmit = useCallback(
        async (payload: CreateSlotPayload | UpdateSlotPayload) => {
            if (slotForm.editingSlot) {
                const updatePayload = payload as UpdateSlotPayload;
                const apiPayload: APIUpdateSlotPayload = {
                    learning_area_id: updatePayload.learningAreaId ?? null,
                    teacher_id: updatePayload.teacherId ?? null,
                    room_identifier: updatePayload.roomIdentifier ?? null,
                };
                await updateSlotMutation.mutateAsync({
                    id: slotForm.editingSlot.id,
                    payload: apiPayload,
                });
            } else {
                const createPayload = payload as CreateSlotPayload;
                const apiPayload: APICreateSlotPayload = {
                    structure_id: createPayload.structureId,
                    class_id: createPayload.classId,
                    learning_area_id: createPayload.learningAreaId,
                    teacher_id: createPayload.teacherId,
                    room_identifier: createPayload.roomIdentifier ?? null,
                };
                await createSlotMutation.mutateAsync(apiPayload);
            }
            setSlotForm({ open: false });
        },
        [slotForm.editingSlot, createSlotMutation, updateSlotMutation]
    );

    const handleDismissConflict = useCallback((structureId: string) => {
        setConflicts((prev) => prev.filter((c) => c.structureId !== structureId));
    }, []);

    const buildFilterLabel = () => {
        const parts: string[] = [];
        return parts.join(", ") || "current filter";
    };

    const isSubmitting = createSlotMutation.isPending || updateSlotMutation.isPending;

    if (isLoading) {
        return (
            <div className="flex h-[60vh] items-center justify-center">
                <div className="animate-pulse space-y-4">
                    <div className="bg-muted h-8 w-1/4 rounded" />
                    <div className="bg-muted h-64 rounded" />
                </div>
            </div>
        );
    }

    if (error) {
        return (
            <div className="flex h-[60vh] items-center justify-center">
                <div className="text-center">
                    <p className="text-destructive">Failed to load timetable</p>
                    <p className="text-muted-foreground text-sm">{error.message}</p>
                </div>
            </div>
        );
    }

    return (
        <div className="space-y-4">
            {/* Header with actions */}
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-4">
                    <h1 className="text-2xl font-semibold">Timetable</h1>
                    {selectedDate && (
                        <span className="text-muted-foreground text-sm">
                            Showing attendance status for{" "}
                            {new Date(selectedDate).toLocaleDateString()}
                        </span>
                    )}
                    {isReadOnly && (
                        <span className="text-muted-foreground bg-muted inline-flex items-center gap-1 rounded px-2 py-1 text-xs">
                            <Lock className="h-3 w-3" />
                            View only
                        </span>
                    )}
                </div>
                <div className="flex items-center gap-2">
                    <Popover>
                        <PopoverTrigger>
                            <Button variant="outline" size="icon">
                                <Calendar className="h-4 w-4" />
                                <span className="sr-only">Select date for attendance status</span>
                            </Button>
                        </PopoverTrigger>
                        <PopoverContent className="w-auto p-2" align="end">
                            <input
                                type="date"
                                value={selectedDate ?? ""}
                                onChange={(e) => setSelectedDate(e.target.value || undefined)}
                                className="rounded-md border p-2"
                            />
                        </PopoverContent>
                    </Popover>
                    {!isReadOnly && (
                        <Button variant="outline" onClick={() => setShowStructureBuilder(true)}>
                            <Settings className="mr-2 h-4 w-4" />
                            Manage Structure
                        </Button>
                    )}
                    {!isReadOnly && timeBlocks.length > 0 && slots.length === 0 && (
                        <Button
                            onClick={() =>
                                setSlotForm({ open: true, structureId: timeBlocks[0].id })
                            }
                        >
                            <Plus className="mr-2 h-4 w-4" />
                            Assign Lesson
                        </Button>
                    )}
                </div>
            </div>

            {/* Conflict banner */}
            <SlotConflictBanner conflicts={conflicts} onDismiss={handleDismissConflict} />

            {/* Empty states or grid */}
            <div className="bg-background rounded-md border">
                {gridView.emptyState === "no_structure" && (
                    <EmptyStateStructure
                        onCreateStructure={
                            isReadOnly ? undefined : () => setShowStructureBuilder(true)
                        }
                    />
                )}
                {gridView.emptyState === "no_slots" && (
                    <EmptyStateAllocation
                        onAssignLesson={
                            isReadOnly
                                ? undefined
                                : () => setSlotForm({ open: true, structureId: timeBlocks[0]?.id })
                        }
                        onManageStructure={
                            isReadOnly ? undefined : () => setShowStructureBuilder(true)
                        }
                    />
                )}
                {gridView.emptyState === "filtered_empty" && (
                    <EmptyStateFiltered
                        filterLabel={buildFilterLabel()}
                        onClearFilters={() => setSelectedDate(undefined)}
                        onAddLesson={
                            isReadOnly
                                ? undefined
                                : () => setSlotForm({ open: true, structureId: timeBlocks[0]?.id })
                        }
                    />
                )}
                {gridView.emptyState === "none" && (
                    <TimetableGrid
                        gridView={gridView}
                        onCellAdd={isReadOnly ? undefined : handleCellAdd}
                        onCellEdit={isReadOnly ? undefined : handleCellEdit}
                        onCellDelete={isReadOnly ? undefined : handleCellDelete}
                        isReadOnly={isReadOnly}
                    />
                )}
            </div>

            {/* Structure Builder Dialog (admin only) */}
            {!isReadOnly && (
                <Dialog open={showStructureBuilder} onOpenChange={setShowStructureBuilder}>
                    <DialogContent className="max-h-[90vh] max-w-3xl">
                        <DialogHeader>
                            <DialogTitle>Timetable Structure Builder</DialogTitle>
                            <DialogDescription>
                                Define periods for Monday, then replicate to other days.
                            </DialogDescription>
                        </DialogHeader>
                        <StructureBuilder
                            onComplete={handleCreateStructure}
                            onCancel={() => setShowStructureBuilder(false)}
                        />
                    </DialogContent>
                </Dialog>
            )}

            {/* Slot Assignment Form (admin only) */}
            {!isReadOnly && (
                <SlotAssignmentForm
                    open={slotForm.open}
                    onClose={() => setSlotForm({ open: false })}
                    onSubmit={handleSlotSubmit}
                    structureId={slotForm.structureId ?? ""}
                    editingSlot={slotForm.editingSlot}
                    classes={classes}
                    learningAreas={learningAreas}
                    teachers={teachers}
                    rooms={rooms}
                    isLoading={isSubmitting}
                />
            )}
        </div>
    );
}
