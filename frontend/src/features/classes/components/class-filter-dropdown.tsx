/**
 * ClassFilterDropdown — Linear-style filter control.
 *
 * Clicking the filter icon opens a dropdown containing filter groups.
 * Each group with options (e.g. Grades) opens a submenu for multi-select.
 * Active filters are shown as badges below the button.
 */

"use client";

import * as React from "react";
import { Filter, Check, X } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuSub,
    DropdownMenuSubContent,
    DropdownMenuSubTrigger,
    DropdownMenuSeparator,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import type { Grade } from "@/features/classes/types";

// ─── Types ─────────────────────────────────────────────────────────────────

interface FilterState {
    gradeIds: string[];
    isActive: boolean | null; // null = no filter, true = active, false = inactive
}

interface ClassFilterDropdownProps {
    grades: Grade[];
    filters: FilterState;
    onFiltersChange: (filters: FilterState) => void;
}

// ─── Helpers ───────────────────────────────────────────────────────────────

function getActiveFilterCount(filters: FilterState): number {
    let count = 0;
    if (filters.gradeIds.length > 0) count++;
    if (filters.isActive !== null) count++;
    return count;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function ClassFilterDropdown({
    grades,
    filters,
    onFiltersChange,
}: ClassFilterDropdownProps) {
    const activeCount = getActiveFilterCount(filters);

    function toggleGrade(gradeId: string) {
        const next = filters.gradeIds.includes(gradeId)
            ? filters.gradeIds.filter((id) => id !== gradeId)
            : [...filters.gradeIds, gradeId];
        onFiltersChange({ ...filters, gradeIds: next });
    }

    function setActiveFilter(value: boolean | null) {
        onFiltersChange({ ...filters, isActive: value });
    }

    function clearAll() {
        onFiltersChange({ gradeIds: [], isActive: null });
    }

    return (
        <DropdownMenu>
            <DropdownMenuTrigger asChild>
                <Button variant="ghost" size="icon-sm" aria-label="Filter" className="relative">
                    <Filter className="size-4" />
                    {activeCount > 0 && (
                        <span className="bg-primary text-primary-foreground absolute -top-1 -right-1 flex h-4 min-w-4 items-center justify-center rounded-full px-1 text-[10px] leading-none font-bold">
                            {activeCount}
                        </span>
                    )}
                </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="start" className="w-48">
                {/* Header with clear */}
                <div className="flex items-center justify-between px-2 py-1">
                    <span className="text-muted-foreground text-xs font-medium">Filters</span>
                    {activeCount > 0 && (
                        <button
                            onClick={clearAll}
                            className="text-muted-foreground hover:text-foreground flex items-center gap-1 text-[11px] transition-colors"
                        >
                            <X className="size-3" />
                            Clear
                        </button>
                    )}
                </div>

                <DropdownMenuSeparator />

                {/* ── Grade Filter (Submenu with multi-select) ────────── */}
                <DropdownMenuSub>
                    <DropdownMenuSubTrigger>
                        <span className="flex items-center gap-2">
                            <span>Grade</span>
                        </span>
                        {filters.gradeIds.length > 0 && (
                            <Badge
                                variant="secondary"
                                className="mr-1 ml-auto h-4 min-w-4 rounded-full px-1 text-[10px]"
                            >
                                {filters.gradeIds.length}
                            </Badge>
                        )}
                    </DropdownMenuSubTrigger>
                    <DropdownMenuSubContent className="w-56">
                        {grades.length === 0 ? (
                            <div className="text-muted-foreground px-2 py-4 text-center text-xs">
                                No grades available
                            </div>
                        ) : (
                            grades.map((grade) => {
                                const selected = filters.gradeIds.includes(grade.id);
                                return (
                                    <DropdownMenuItem
                                        key={grade.id}
                                        onSelect={(e) => {
                                            e.preventDefault();
                                            toggleGrade(grade.id);
                                        }}
                                        className="flex items-center gap-2"
                                    >
                                        <span
                                            className={`flex h-4 w-4 items-center justify-center rounded-sm border transition-colors ${
                                                selected
                                                    ? "bg-primary border-primary"
                                                    : "border-muted-foreground/30"
                                            }`}
                                        >
                                            {selected && (
                                                <Check className="text-primary-foreground size-3" />
                                            )}
                                        </span>
                                        <span className="flex-1">{grade.name}</span>
                                    </DropdownMenuItem>
                                );
                            })
                        )}
                    </DropdownMenuSubContent>
                </DropdownMenuSub>

                {/* ── Status Filter ───────────────────────────────────── */}
                <DropdownMenuItem
                    onSelect={(e) => {
                        e.preventDefault();
                        setActiveFilter(filters.isActive === true ? null : true);
                    }}
                    className="flex items-center gap-2"
                >
                    <span
                        className={`flex h-4 w-4 items-center justify-center rounded-sm border transition-colors ${
                            filters.isActive === true
                                ? "bg-primary border-primary"
                                : "border-muted-foreground/30"
                        }`}
                    >
                        {filters.isActive === true && (
                            <Check className="text-primary-foreground size-3" />
                        )}
                    </span>
                    <span>Active</span>
                </DropdownMenuItem>
                <DropdownMenuItem
                    onSelect={(e) => {
                        e.preventDefault();
                        setActiveFilter(filters.isActive === false ? null : false);
                    }}
                    className="flex items-center gap-2"
                >
                    <span
                        className={`flex h-4 w-4 items-center justify-center rounded-sm border transition-colors ${
                            filters.isActive === false
                                ? "bg-primary border-primary"
                                : "border-muted-foreground/30"
                        }`}
                    >
                        {filters.isActive === false && (
                            <Check className="text-primary-foreground size-3" />
                        )}
                    </span>
                    <span>Inactive</span>
                </DropdownMenuItem>
            </DropdownMenuContent>
        </DropdownMenu>
    );
}
