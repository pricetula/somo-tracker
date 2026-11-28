"use client";

import { useEffect } from "react";
import { Trash2, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import type { DataTableBulkActionContext } from "../types";

interface DataTableSelectionSheetProps {
  open: boolean;
  selectedCount: number;
  selectedIds: string[];
  onDelete?: () => void;
  onClearSelection: () => void;
  onClose: () => void;
  renderBulkActions?: (context: DataTableBulkActionContext) => React.ReactNode;
  refresh?: () => void;
}

export function DataTableSelectionSheet({
  open,
  selectedCount,
  selectedIds,
  onDelete,
  onClearSelection,
  onClose,
  renderBulkActions,
  refresh,
}: DataTableSelectionSheetProps) {
  // Escape closes the sheet.
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        onClose();
      }
    };
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [onClose]);

  if (!open) return null;

  const clearSelection = () => {
    onClearSelection();
    onClose();
  };

  return (
    <div
      className="absolute bottom-0 left-0 right-0 z-50 p-3 flex"
      style={{ pointerEvents: "none" }}
    >
      <div
        className="flex flex-col gap-2 rounded-xl border border-background bg-popover p-3 shadow-lg ring-1 ring-foreground/10 w-full max-w-md mx-auto"
        style={{ pointerEvents: "auto" }}
      >
        <div className="flex items-center justify-between gap-2">
          {onDelete && (
            <Button
              variant="destructive"
              size="icon"
              className="size-8 p-0"
              onClick={onDelete}
              aria-label="Delete selected"
            >
              <Trash2 className="size-5" />
            </Button>
          )}
          <Button
            variant="outline"
            size="icon"
            className="size-8 p-0"
            onClick={clearSelection}
            aria-label="Clear selection"
          >
            <X className="size-5" />
          </Button>
        </div>

        {renderBulkActions && (
          <div
            className="flex items-center justify-end gap-2 w-full"
            role="group"
            aria-label="Bulk actions"
          >
            {renderBulkActions({
              selectedIds,
              selectedCount,
              clearSelection,
              refresh: refresh ?? onClose,
            })}
          </div>
        )}
      </div>
    </div>
  );
}
