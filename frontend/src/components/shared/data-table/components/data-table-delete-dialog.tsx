"use client";

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";

interface DataTableDeleteDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  selectedCount: number;
  ids: string[];
  onConfirm: (ids: string[]) => void;
  deleting: boolean;
}

export function DataTableDeleteDialog({
  open,
  onOpenChange,
  selectedCount,
  ids,
  onConfirm,
  deleting,
}: DataTableDeleteDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Delete {selectedCount} record(s)?</DialogTitle>
          <DialogDescription>
            This action cannot be undone. Deletion is performed by the backend
            and any records that cannot be deleted will remain selected.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            variant="destructive"
            disabled={deleting || ids.length === 0}
            onClick={() => onConfirm(ids)}
          >
            {deleting ? "Deleting…" : "Delete"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
