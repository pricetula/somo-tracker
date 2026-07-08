"use client";

import * as React from "react";
import { GripVertical, Pencil, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";

import type { PerformanceIndicator } from "@/lib/api/curriculum";
import { useDeletePerformanceIndicator } from "../hooks/use-curriculum";
import { DeleteConfirmDialog } from "./delete-confirm-dialog";

// ─── Props ─────────────────────────────────────────────────────────────────

interface PerformanceIndicatorRowProps {
    indicator: PerformanceIndicator;
    learningAreaId: string;
    onEdit: (indicator: PerformanceIndicator) => void;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function PerformanceIndicatorRow({
    indicator,
    learningAreaId,
    onEdit,
}: PerformanceIndicatorRowProps) {
    const deleteMutation = useDeletePerformanceIndicator();
    const [deleteOpen, setDeleteOpen] = React.useState(false);

    return (
        <>
            <div className="group flex items-center gap-2 py-1.5 pl-12">
                <GripVertical className="text-muted-foreground/30 size-3.5 shrink-0" />
                <span className="text-muted-foreground text-xs tabular-nums">
                    {indicator.sequence_order}.
                </span>
                <p className="text-sm">{indicator.description}</p>
                <div className="ml-auto flex items-center gap-1 opacity-0 transition-opacity group-hover:opacity-100">
                    <Button
                        variant="ghost"
                        size="icon-sm"
                        onClick={() => onEdit(indicator)}
                        aria-label="Edit indicator"
                    >
                        <Pencil className="size-3.5" />
                    </Button>
                    <Button
                        variant="ghost"
                        size="icon-sm"
                        onClick={() => setDeleteOpen(true)}
                        aria-label="Delete indicator"
                    >
                        <Trash2 className="size-3.5" />
                    </Button>
                </div>
            </div>
            <DeleteConfirmDialog
                open={deleteOpen}
                onOpenChange={setDeleteOpen}
                title="Delete Performance Indicator"
                description="This action cannot be undone. The performance indicator will be permanently removed."
                onConfirm={() => {
                    deleteMutation.mutate(
                        { id: indicator.id, learningAreaId },
                        { onSuccess: () => setDeleteOpen(false) }
                    );
                }}
                isPending={deleteMutation.isPending}
            />
        </>
    );
}
