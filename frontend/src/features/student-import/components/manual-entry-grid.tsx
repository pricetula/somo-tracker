/**
 * Pattern A: Virtualized manual entry grid.
 *
 * Uses TanStack Virtual to maintain 60fps with hundreds of rows.
 * Each row has editable fields: Name, Gender, DOB, UPI, KNEC, Parent, Class.
 */

"use client";

import * as React from "react";
import { X, Plus } from "lucide-react";
import { useVirtualizer } from "@tanstack/react-virtual";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { Input } from "@/components/ui/input";
import { ParentCombobox } from "./parent-combobox";
import { ClassCombobox } from "./class-combobox";
import type { ManualRow } from "../hooks/use-student-import";
import type { ParentsMap, ClassesMap } from "../types";

interface ManualEntryGridProps {
    rows: ManualRow[];
    parentsMap: ParentsMap;
    classesMap: ClassesMap;
    onAddRow: () => void;
    onRemoveRow: (rowIndex: number) => void;
    onUpdateRow: (rowIndex: number, field: keyof ManualRow, value: string) => void;
    onProceed: () => void;
}

export function ManualEntryGrid({
    rows,
    parentsMap,
    classesMap,
    onAddRow,
    onRemoveRow,
    onUpdateRow,
    onProceed,
}: ManualEntryGridProps) {
    const parentRef = React.useRef<HTMLDivElement>(null);

    // eslint-disable-next-line react-hooks/incompatible-library
    const virtualizer = useVirtualizer({
        count: rows.length,
        getScrollElement: () => parentRef.current,
        estimateSize: () => 52,
        overscan: 10,
    });

    const filledCount = rows.filter((r) => r.full_name.trim()).length;
    const canProceed = filledCount > 0;

    return (
        <div className="space-y-3">
            <h3 className="text-sm font-medium">Manual Entry</h3>

            {/* Column headers */}
            <div
                className="text-muted-foreground grid gap-2 px-1 text-xs font-medium"
                style={{
                    gridTemplateColumns: "1fr 80px 120px 1fr 1fr 1fr 1fr 28px",
                }}
            >
                <span>Full Name *</span>
                <span>Gender *</span>
                <span>Date of Birth</span>
                <span>UPI Number</span>
                <span>KNEC #</span>
                <span>Parent</span>
                <span>Class</span>
                <span />
            </div>

            {/* Virtualized rows */}
            <div ref={parentRef} className="max-h-120 overflow-auto">
                <div
                    style={{
                        height: `${virtualizer.getTotalSize()}px`,
                        width: "100%",
                        position: "relative",
                    }}
                >
                    {virtualizer.getVirtualItems().map((virtualItem) => {
                        const row = rows[virtualItem.index];
                        const isLastRow = rows.length <= 1;

                        return (
                            <div
                                key={row._rowIndex}
                                data-index={virtualItem.index}
                                ref={virtualizer.measureElement}
                                className="absolute top-0 left-0 w-full"
                                style={{
                                    transform: `translateY(${virtualItem.start}px)`,
                                }}
                            >
                                <div
                                    className="grid gap-2 px-1 py-1"
                                    style={{
                                        gridTemplateColumns: "1fr 80px 120px 1fr 1fr 1fr 1fr 28px",
                                    }}
                                >
                                    <Input
                                        placeholder="Full name"
                                        value={row.full_name}
                                        onChange={(e) =>
                                            onUpdateRow(row._rowIndex, "full_name", e.target.value)
                                        }
                                        className="h-9 text-sm"
                                    />
                                    <Select
                                        value={row.gender}
                                        onValueChange={(v) =>
                                            onUpdateRow(row._rowIndex, "gender", v)
                                        }
                                    >
                                        <SelectTrigger className="h-9 text-sm">
                                            <SelectValue placeholder="-" />
                                        </SelectTrigger>
                                        <SelectContent>
                                            <SelectItem value="M">M</SelectItem>
                                            <SelectItem value="F">F</SelectItem>
                                        </SelectContent>
                                    </Select>
                                    <Input
                                        placeholder="DD/MM/YYYY"
                                        value={row.date_of_birth}
                                        onChange={(e) =>
                                            onUpdateRow(
                                                row._rowIndex,
                                                "date_of_birth",
                                                e.target.value
                                            )
                                        }
                                        className="h-9 text-sm"
                                    />
                                    <Input
                                        placeholder="UPI"
                                        value={row.upi_number}
                                        onChange={(e) =>
                                            onUpdateRow(row._rowIndex, "upi_number", e.target.value)
                                        }
                                        className="h-9 text-sm"
                                    />
                                    <Input
                                        placeholder="KNEC"
                                        value={row.knec_assessment_number}
                                        onChange={(e) =>
                                            onUpdateRow(
                                                row._rowIndex,
                                                "knec_assessment_number",
                                                e.target.value
                                            )
                                        }
                                        className="h-9 text-sm"
                                    />
                                    <ParentCombobox
                                        value={row.parent_name}
                                        parentsMap={parentsMap}
                                        onChange={(v) =>
                                            onUpdateRow(row._rowIndex, "parent_name", v)
                                        }
                                    />
                                    <ClassCombobox
                                        value={row.class_name}
                                        classesMap={classesMap}
                                        onChange={(v) =>
                                            onUpdateRow(row._rowIndex, "class_name", v)
                                        }
                                    />
                                    <button
                                        onClick={() => onRemoveRow(row._rowIndex)}
                                        disabled={isLastRow}
                                        className="text-muted-foreground hover:text-foreground mt-1 flex size-7 items-center justify-center rounded-md disabled:opacity-30"
                                    >
                                        <X className="size-4" />
                                    </button>
                                </div>
                            </div>
                        );
                    })}
                </div>
            </div>

            {/* Actions bar */}
            <div className="flex items-center justify-between px-1 pt-2">
                <button
                    onClick={onAddRow}
                    className="text-muted-foreground hover:text-foreground flex items-center gap-1.5 text-xs font-medium"
                >
                    <Plus className="size-3.5" />
                    Add row
                </button>

                <div className="flex items-center gap-3">
                    <span className="text-muted-foreground text-xs">{filledCount} filled</span>
                    <button
                        onClick={onProceed}
                        disabled={!canProceed}
                        className="bg-primary text-primary-foreground hover:bg-primary/90 rounded-md px-4 py-1.5 text-sm font-medium disabled:opacity-50"
                    >
                        Validate &amp; Review
                    </button>
                </div>
            </div>
        </div>
    );
}
