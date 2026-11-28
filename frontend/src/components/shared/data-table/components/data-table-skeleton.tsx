"use client";

/**
 * Rendered in place of body rows while an additional page is being fetched
 * (`isLoadingMore`). Provides visual feedback that more data is arriving.
 */

interface DataTableSkeletonProps {
  columnCount: number;
  rowCount?: number;
  enableRowSelection?: boolean;
  renderRowActions?: boolean;
}

export function DataTableSkeleton({
  columnCount,
  rowCount = 5,
  enableRowSelection = false,
  renderRowActions = false,
}: DataTableSkeletonProps) {
  return (
    <div className="flex flex-col gap-1.5">
      {Array.from({ length: rowCount }).map((_, rowIdx) => (
        <div
          key={rowIdx}
          className="grid border-b border-background/50 py-3"
          style={{
            gridTemplateColumns: "repeat(var(--col-count), 1fr)",
            minWidth: "max-content",
          }}
        >
          {enableRowSelection && (
            <div className="px-4 flex items-center">
              <div className="w-4 h-4 rounded bg-muted animate-pulse" />
            </div>
          )}
          {Array.from({ length: columnCount }).map((_, colIdx) => (
            <div
              key={colIdx}
              className="px-4 h-4 bg-muted animate-pulse rounded"
            />
          ))}
          {renderRowActions && (
            <div className="px-3 flex items-center justify-end">
              <div className="w-6 h-6 rounded-full bg-muted animate-pulse" />
            </div>
          )}
        </div>
      ))}
    </div>
  );
}
