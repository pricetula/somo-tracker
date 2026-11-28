"use client";

import { useEffect } from "react";
import { Trash2, X } from "lucide-react";
import { Button } from "@/components/ui/button";

interface DataTableSelectionSheetProps {
  open: boolean;
  onDelete?: () => void;
  onClearSelection: () => void;
  onClose: () => void;
}

export function DataTableSelectionSheet({
  open,
  onDelete,
  onClearSelection,
  onClose,
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

  return (
    <div
      className="absolute bottom-0 left-0 right-0 z-50 flex justify-center"
      style={{ pointerEvents: "none" }}
    >
      <div
        className="flex items-center justify-between gap-4 rounded-xl border border-background bg-popover p-2 px-6 shadow-lg ring-1 ring-foreground/10"
        style={{ pointerEvents: "auto" }}
      >
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
          onClick={() => {
            onClearSelection();
            onClose();
          }}
          aria-label="Clear selection"
        >
          <X className="size-5" />
        </Button>
      </div>
    </div>
  );
}
