"use client";

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
  if (!open) return null;

  return (
    <div
      className="fixed bottom-[10%] left-1/2 -translate-x-1/2 z-50 w-full max-w-md"
      onClick={(e) => e.stopPropagation()}
    >
      <div className="rounded-xl border border-background bg-popover p-4 shadow-lg ring-1 ring-foreground/10">
        <div className="flex items-center gap-2 text-sm font-medium">
          {selectedCount} selected
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
