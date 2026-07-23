"use client";

import { Skeleton } from "@/components/ui/skeleton";

interface SkeletonRowsProps {
    rowHeight: number;
    gridTemplateColumns: string;
    isCheckable: boolean;
}

/**
 * Renders 4 skeleton rows that match the table's grid layout, borders,
 * and cell structure — as siblings to the virtualized rows region.
 */
export function SkeletonRows({ rowHeight, gridTemplateColumns, isCheckable }: SkeletonRowsProps) {
    const columnCount = gridTemplateColumns.split(" ").length;
    const contentColumnCount = isCheckable ? columnCount - 1 : columnCount;

    return (
        <>
            {Array.from({ length: 4 }, (_, i) => (
                <div
                    key={i}
                    className="hover:bg-muted/30 grid w-full border-b text-xs/relaxed transition-colors"
                    style={{
                        height: rowHeight,
                        gridTemplateColumns,
                    }}
                >
                    {isCheckable && (
                        <div className="flex items-center justify-center">
                            <Skeleton className="size-4 shrink-0 rounded-[4px]" />
                        </div>
                    )}
                    {Array.from({ length: contentColumnCount }, (_, j) => (
                        <div key={j} className="flex items-center truncate border-l px-3">
                            <Skeleton className="h-3.5 w-full" />
                        </div>
                    ))}
                </div>
            ))}
        </>
    );
}
