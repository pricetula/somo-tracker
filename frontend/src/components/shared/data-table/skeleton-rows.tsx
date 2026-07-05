"use client";

import { Skeleton } from "@/components/ui/skeleton";

interface SkeletonRowsProps {
    rowHeight: number;
    height: number;
    columnCount: number;
    isCheckable: boolean;
}

export function SkeletonRows({ rowHeight, height, columnCount, isCheckable }: SkeletonRowsProps) {
    const count = Math.max(4, Math.ceil(height / rowHeight));
    return (
        <div style={{ height }} className="flex flex-col">
            {Array.from({ length: count }, (_, i) => (
                <div key={i} className="flex items-center gap-2 px-3" style={{ height: rowHeight }}>
                    {isCheckable && <Skeleton className="size-4 shrink-0 rounded-[4px]" />}
                    {Array.from({ length: columnCount }, (_, j) => (
                        <Skeleton
                            key={j}
                            className="h-3.5 flex-1"
                            style={{
                                maxWidth: `${40 + Math.random() * 40}%`,
                            }}
                        />
                    ))}
                </div>
            ))}
        </div>
    );
}
