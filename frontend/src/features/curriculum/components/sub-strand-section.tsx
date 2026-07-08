"use client";

import * as React from "react";
import Link from "next/link";
import { ChevronDown, ChevronRight, Plus, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";

import type { SubStrandTree, PerformanceIndicator } from "@/lib/api/curriculum";
import { useDeleteSubStrand } from "../hooks/use-curriculum";
import { PerformanceIndicatorRow } from "./performance-indicator-row";
import { DeleteConfirmDialog } from "./delete-confirm-dialog";

// ─── Props ─────────────────────────────────────────────────────────────────

interface SubStrandSectionProps {
    subStrand: SubStrandTree;
    learningAreaId: string;
    onEditIndicator: (indicator: PerformanceIndicator) => void;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function SubStrandSection({
    subStrand,
    learningAreaId,
    onEditIndicator,
}: SubStrandSectionProps) {
    const [expanded, setExpanded] = React.useState(false);
    const deleteMutation = useDeleteSubStrand();
    const [deleteOpen, setDeleteOpen] = React.useState(false);

    const indicatorCount = subStrand.performance_indicators.length;

    return (
        <div>
            {/* Sub-Strand Header */}
            <div className="hover:bg-muted/30 group flex items-center gap-2 rounded-sm py-2 pr-2 pl-8 transition-colors">
                <button
                    type="button"
                    onClick={() => setExpanded(!expanded)}
                    className="flex items-center gap-1.5"
                >
                    {expanded ? (
                        <ChevronDown className="text-muted-foreground size-4" />
                    ) : (
                        <ChevronRight className="text-muted-foreground size-4" />
                    )}
                    <span className="text-sm font-medium">{subStrand.name}</span>
                </button>
                <Badge variant="secondary" className="text-xs font-normal">
                    {indicatorCount} indicator{indicatorCount !== 1 ? "s" : ""}
                </Badge>
                <div className="ml-auto flex items-center gap-1 opacity-0 transition-opacity group-hover:opacity-100">
                    <Button variant="ghost" size="icon-sm" aria-label="Add indicator" asChild>
                        <Link href={`/curriculum/new-indicator?subStrandId=${subStrand.id}`}>
                            <Plus className="size-3.5" />
                        </Link>
                    </Button>
                    <Button
                        variant="ghost"
                        size="icon-sm"
                        onClick={() => setDeleteOpen(true)}
                        aria-label="Delete sub-strand"
                    >
                        <Trash2 className="size-3.5" />
                    </Button>
                </div>
            </div>

            {/* Performance Indicators */}
            {expanded && (
                <div>
                    {indicatorCount === 0 ? (
                        <p className="text-muted-foreground py-2 pl-12 text-xs">
                            No performance indicators yet.
                        </p>
                    ) : (
                        subStrand.performance_indicators
                            .sort((a, b) => a.sequence_order - b.sequence_order)
                            .map((indicator) => (
                                <PerformanceIndicatorRow
                                    key={indicator.id}
                                    indicator={indicator}
                                    learningAreaId={learningAreaId}
                                    onEdit={onEditIndicator}
                                />
                            ))
                    )}
                </div>
            )}

            {/* Delete Sub-Strand Dialog */}
            <DeleteConfirmDialog
                open={deleteOpen}
                onOpenChange={setDeleteOpen}
                title="Delete Sub-Strand"
                description="This action cannot be undone. The sub-strand and all its performance indicators will be permanently removed."
                onConfirm={() => {
                    deleteMutation.mutate(
                        { id: subStrand.id, learningAreaId },
                        { onSuccess: () => setDeleteOpen(false) }
                    );
                }}
                isPending={deleteMutation.isPending}
            />
        </div>
    );
}
