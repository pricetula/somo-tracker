"use client";

import { useCallback, useEffect } from "react";
import { Button } from "@/components/ui/button";

interface DataTableSelectionSheetProps {
  open: boolean;
  selectedCount: number;
  onClearSelection: () => void;
  onClose: () => void;
  children?: React.ReactNode;
}

export function DataTableSelectionSheet({
  open,
  selectedCount,
  onClearSelection,
  onClose,
  children,
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

  const handleBackdropClick = useCallback(
    (e: React.MouseEvent) => {
      if (e.target === e.currentTarget) {
        onClose();
      }
    },
    [onClose],
  );

  if (!open) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-end justify-center p-4"
      onClick={handleBackdropClick}
    >
      <div
        className="fixed inset-0 bg-background/80 backdrop-blur-sm"
        aria-hidden="true"
      />
      <div
        className="rounded-xl border border-background bg-popover p-4 shadow-lg ring-1 ring-foreground/10 w-full max-w-md"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between gap-2 font-medium">
          <span>{selectedCount} selected</span>
          <Button
            variant="outline"
            size="icon"
            className="size-6 p-1"
            onClick={onClose}
            aria-label="Close selection"
          >
            ×
          </Button>
        </div>
        <div className="mt-4 flex flex-col gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => {
              onClearSelection();
              onClose();
            }}
          >
            Clear selection
          </Button>
          {children}
        </div>
      </div>
    </div>
  );
}
