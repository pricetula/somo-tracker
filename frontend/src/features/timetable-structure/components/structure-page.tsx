/**
 * StructurePage — main container for the Timetable Settings portal.
 *
 * Two states:
 *   1. No blocks → "Add Blueprint" button opens a dialog to define the
 *      full day's block sequence (saved for all 5 weekdays).
 *   2. Blocks exist → slot grid for assigning classes to periods.
 */

"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Plus, Calendar } from "lucide-react";
import { Button } from "@/components/ui/button";

import { useTimeBlockList, useBatchCreateTimeBlocks } from "../hooks/use-timetable-structure";
import { BlueprintDialog } from "./blueprint-dialog";
import { TimetableSlotGrid } from "./timetable-slot-grid";
import { TemplateMenu } from "./template-menu";
import { AcademicYearCombobox } from "@/features/academic-terms/components/academic-year-combobox";

import type { BatchCreateTimeBlockPayload } from "@/lib/api/timetable-structure";

// ─── Component ─────────────────────────────────────────────────────────────

export function StructurePage() {
    const router = useRouter();
    const [academicYearID, setAcademicYearID] = useState("");

    const { data, isLoading } = useTimeBlockList(academicYearID);
    const batchCreateMutation = useBatchCreateTimeBlocks();

    const [blueprintOpen, setBlueprintOpen] = useState(false);

    const blocks = data?.items ?? [];
    const hasBlocks = blocks.length > 0;

    const handleSaveBlueprint = (
        entries: { periodName: string; startTime: string; endTime: string; isBreak: boolean }[]
    ) => {
        const payload: BatchCreateTimeBlockPayload = { blocks: [] };
        for (const entry of entries) {
            for (let day = 1; day <= 5; day++) {
                payload.blocks.push({
                    day_of_week: day,
                    period_name: entry.periodName,
                    start_time: `${entry.startTime}:00`,
                    end_time: `${entry.endTime}:00`,
                    is_break: entry.isBreak,
                    academic_year_id: academicYearID,
                });
            }
        }
        batchCreateMutation.mutate(payload, {
            onSuccess: () => setBlueprintOpen(false),
        });
    };

    const handleApplyTemplate = (payload: BatchCreateTimeBlockPayload) => {
        batchCreateMutation.mutate(payload, {
            onSuccess: () => setBlueprintOpen(false),
        });
    };

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="flex items-center justify-between gap-2">
                <AcademicYearCombobox
                    value={academicYearID}
                    onChange={setAcademicYearID}
                    placeholder="Select academic year..."
                    className="w-48"
                    onCreateItem={() => router.push("/academic-years/new")}
                />
                {academicYearID && (
                    <div className="flex items-center gap-2">
                        <TemplateMenu
                            isPending={batchCreateMutation.isPending}
                            academicYearID={academicYearID}
                            onApplyTemplate={handleApplyTemplate}
                        />
                        <Button onClick={() => setBlueprintOpen(true)} size="sm">
                            <Plus className="mr-1.5 h-4 w-4" />
                            Add Blueprint
                        </Button>
                    </div>
                )}
            </div>

            {/* Body */}
            {!academicYearID ? (
                <div className="flex flex-col items-center justify-center py-16 text-center">
                    <Calendar className="text-muted-foreground/40 mb-4 h-12 w-12" />
                    <h3 className="text-foreground text-lg font-medium">Select an Academic Year</h3>
                    <p className="text-muted-foreground mt-1 max-w-sm">
                        Choose an academic year above to get started.
                    </p>
                </div>
            ) : !hasBlocks ? (
                <div className="flex flex-col items-center justify-center py-16 text-center">
                    <Calendar className="text-muted-foreground/40 mb-4 h-12 w-12" />
                    <h3 className="text-foreground text-lg font-medium">No timetable yet</h3>
                    <p className="text-muted-foreground mt-1 mb-4 max-w-sm">
                        Create a blueprint to define the periods for each school day.
                    </p>
                    <Button
                        onClick={() => setBlueprintOpen(true)}
                        size="lg"
                        disabled={batchCreateMutation.isPending}
                    >
                        {batchCreateMutation.isPending ? (
                            <span className="flex items-center gap-2">
                                <Calendar className="h-5 w-5 animate-pulse" />
                                Creating…
                            </span>
                        ) : (
                            <span className="flex items-center gap-2">
                                <Plus className="h-5 w-5" />
                                Add Blueprint
                            </span>
                        )}
                    </Button>
                </div>
            ) : (
                <TimetableSlotGrid
                    blocks={blocks}
                    academicYearID={academicYearID}
                    isLoading={isLoading}
                />
            )}

            {/* Blueprint dialog */}
            <BlueprintDialog
                open={blueprintOpen}
                onOpenChange={setBlueprintOpen}
                isPending={batchCreateMutation.isPending}
                onSave={handleSaveBlueprint}
            />
        </div>
    );
}
