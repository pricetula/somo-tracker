/**
 * TemplateMenu — dropdown menu for applying standard day structure templates.
 *
 * Provides pre-configured time block matrices (e.g. "Standard Friday Matrix",
 * "Early Years Schedule") that can be applied atomically via the batch endpoint.
 */

"use client";

import { ChevronDown, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

import type { BatchCreateTimeBlockPayload } from "@/lib/api/timetable-structure";

// ─── Template definitions ─────────────────────────────────────────────────

export interface DayTemplate {
    name: string;
    description: string;
    getPayload: (academicYearID: string) => BatchCreateTimeBlockPayload;
}

const STANDARD_MONDAY_FRIDAY: DayTemplate = {
    name: "Standard Monday–Friday Matrix",
    description: "8 blocks: Assembly + 6 lessons + 2 breaks per day",
    getPayload: (academicYearID: string) => generateStandardMatrix(academicYearID),
};

const EARLY_YEARS_SCHEDULE: DayTemplate = {
    name: "Early Years Schedule",
    description: "6 blocks: 4 lessons + 2 breaks, shorter periods",
    getPayload: (academicYearID: string) => generateEarlyYearsSchedule(academicYearID),
};

const SENIOR_SCHOOL_SCHEDULE: DayTemplate = {
    name: "Senior School Schedule",
    description: "10 blocks: 8 lessons + 2 breaks, 40-min periods",
    getPayload: (academicYearID: string) => generateSeniorSchedule(academicYearID),
};

export const DAY_TEMPLATES: DayTemplate[] = [
    STANDARD_MONDAY_FRIDAY,
    EARLY_YEARS_SCHEDULE,
    SENIOR_SCHOOL_SCHEDULE,
];

// ─── Template generators ──────────────────────────────────────────────────

interface BlockDef {
    periodName: string;
    isBreak: boolean;
    start: string; // "HH:MM"
    end: string; // "HH:MM"
}

const STANDARD_BLOCKS: BlockDef[] = [
    { periodName: "Assembly", isBreak: false, start: "07:45", end: "08:15" },
    { periodName: "Lesson 1", isBreak: false, start: "08:15", end: "08:55" },
    { periodName: "Lesson 2", isBreak: false, start: "08:55", end: "09:35" },
    { periodName: "Morning Break", isBreak: true, start: "09:35", end: "09:50" },
    { periodName: "Lesson 3", isBreak: false, start: "09:50", end: "10:30" },
    { periodName: "Lesson 4", isBreak: false, start: "10:30", end: "11:10" },
    { periodName: "Lesson 5", isBreak: false, start: "11:10", end: "11:50" },
    { periodName: "Lunch Break", isBreak: true, start: "11:50", end: "12:05" },
    { periodName: "Lesson 6", isBreak: false, start: "12:05", end: "12:45" },
    { periodName: "Lesson 7", isBreak: false, start: "12:45", end: "13:25" },
    { periodName: "Extra-Curricular", isBreak: false, start: "14:00", end: "15:00" },
];

const EARLY_YEARS_BLOCKS: BlockDef[] = [
    { periodName: "Lesson 1", isBreak: false, start: "08:00", end: "08:30" },
    { periodName: "Lesson 2", isBreak: false, start: "08:30", end: "09:00" },
    { periodName: "Break Time", isBreak: true, start: "09:00", end: "09:20" },
    { periodName: "Lesson 3", isBreak: false, start: "09:20", end: "09:50" },
    { periodName: "Lesson 4", isBreak: false, start: "09:50", end: "10:20" },
    { periodName: "Recess", isBreak: true, start: "10:20", end: "10:35" },
];

const SENIOR_BLOCKS: BlockDef[] = [
    { periodName: "Lesson 1", isBreak: false, start: "08:00", end: "08:40" },
    { periodName: "Lesson 2", isBreak: false, start: "08:40", end: "09:20" },
    { periodName: "Lesson 3", isBreak: false, start: "09:20", end: "10:00" },
    { periodName: "Morning Break", isBreak: true, start: "10:00", end: "10:20" },
    { periodName: "Lesson 4", isBreak: false, start: "10:20", end: "11:00" },
    { periodName: "Lesson 5", isBreak: false, start: "11:00", end: "11:40" },
    { periodName: "Lesson 6", isBreak: false, start: "11:40", end: "12:20" },
    { periodName: "Lesson 7", isBreak: false, start: "12:20", end: "13:00" },
    { periodName: "Lunch Break", isBreak: true, start: "13:00", end: "13:20" },
    { periodName: "Lesson 8", isBreak: false, start: "13:20", end: "14:00" },
];

function blocksToPayload(blocks: BlockDef[], academicYearID: string): BatchCreateTimeBlockPayload {
    // Apply blocks to all 5 weekdays
    const allBlocks = [];
    for (let day = 1; day <= 5; day++) {
        for (const b of blocks) {
            allBlocks.push({
                day_of_week: day,
                period_name: b.periodName,
                start_time: `${b.start}:00`,
                end_time: `${b.end}:00`,
                is_break: b.isBreak,
                academic_year_id: academicYearID,
            });
        }
    }
    return { blocks: allBlocks };
}

function generateStandardMatrix(academicYearID: string): BatchCreateTimeBlockPayload {
    return blocksToPayload(STANDARD_BLOCKS, academicYearID);
}

function generateEarlyYearsSchedule(academicYearID: string): BatchCreateTimeBlockPayload {
    return blocksToPayload(EARLY_YEARS_BLOCKS, academicYearID);
}

function generateSeniorSchedule(academicYearID: string): BatchCreateTimeBlockPayload {
    return blocksToPayload(SENIOR_BLOCKS, academicYearID);
}

// ─── Props ─────────────────────────────────────────────────────────────────

interface TemplateMenuProps {
    isPending: boolean;
    academicYearID: string;
    onApplyTemplate: (payload: BatchCreateTimeBlockPayload) => void;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function TemplateMenu({ isPending, academicYearID, onApplyTemplate }: TemplateMenuProps) {
    return (
        <DropdownMenu>
            <DropdownMenuTrigger asChild>
                <Button variant="outline" size="sm" disabled={isPending}>
                    {isPending ? (
                        <>
                            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                            Applying…
                        </>
                    ) : (
                        <>
                            Apply Standard Template
                            <ChevronDown className="ml-2 h-4 w-4" />
                        </>
                    )}
                </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-64">
                {DAY_TEMPLATES.map((tpl) => (
                    <DropdownMenuItem
                        key={tpl.name}
                        onClick={() => onApplyTemplate(tpl.getPayload(academicYearID))}
                        disabled={isPending}
                        className="flex flex-col items-start gap-0.5"
                    >
                        <span className="font-medium">{tpl.name}</span>
                        <span className="text-muted-foreground text-xs">{tpl.description}</span>
                    </DropdownMenuItem>
                ))}
            </DropdownMenuContent>
        </DropdownMenu>
    );
}
