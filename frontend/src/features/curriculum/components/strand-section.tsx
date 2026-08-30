"use client";

import * as React from "react";
import Link from "next/link";
import { ChevronDown, ChevronRight, Plus, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";

import type { StrandTree, PerformanceIndicator } from "@/lib/api/curriculum";
import { useDeleteStrand } from "../hooks/use-curriculum";
import { SubStrandSection } from "./sub-strand-section";
import { DeleteConfirmDialog } from "./delete-confirm-dialog";

// ─── Props ─────────────────────────────────────────────────────────────────

interface StrandSectionProps {
    strand: StrandTree;
    learningAreaId: string;
    onEditIndicator: (indicator: PerformanceIndicator) => void;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function StrandSection({ strand, learningAreaId, onEditIndicator }: StrandSectionProps) {
    const [expanded, setExpanded] = React.useState(false);
    const deleteMutation = useDeleteStrand();
    const [deleteOpen, setDeleteOpen] = React.useState(false);

    const subStrandCount = strand.sub_strands.length;

    return (
        <div className="space-y-0.5">
            {/* Strand Header */}
            <div className="hover:bg-muted/30 group flex items-center gap-2 rounded-sm py-2.5 pr-2 transition-colors">
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
                    <span className="font-semibold">{strand.name}</span>
                </button>
                <Badge className="font-normal">
                    {subStrandCount} sub-strand{subStrandCount !== 1 ? "s" : ""}
                </Badge>
                <div className="ml-auto flex items-center gap-1 opacity-0 transition-opacity group-hover:opacity-100">
                    <Button variant="ghost" size="icon-sm" aria-label="Add sub-strand" asChild>
                        <Link href={`/curriculum/new-sub-strand?strandId=${strand.id}`}>
                            <Plus className="size-3.5" />
                        </Link>
                    </Button>
                    <Button
                        variant="ghost"
                        size="icon-sm"
                        onClick={() => setDeleteOpen(true)}
                        aria-label="Delete strand"
                    >
                        <Trash2 className="size-3.5" />
                    </Button>
                </div>
            </div>

            {/* Sub-Strands */}
            {expanded && (
                <div>
                    {subStrandCount === 0 ? (
                        <p className="text-muted-foreground py-2 pl-8">No sub-strands yet.</p>
                    ) : (
                        strand.sub_strands.map((subStrand) => (
                            <SubStrandSection
                                key={subStrand.id}
                                subStrand={subStrand}
                                learningAreaId={learningAreaId}
                                onEditIndicator={onEditIndicator}
                            />
                        ))
                    )}
                </div>
            )}

            {/* Delete Strand Dialog */}
            <DeleteConfirmDialog
                open={deleteOpen}
                onOpenChange={setDeleteOpen}
                title="Delete Strand"
                description="This action cannot be undone. The strand, all its sub-strands, and performance indicators will be permanently removed."
                onConfirm={() => {
                    deleteMutation.mutate(
                        { id: strand.id, learningAreaId },
                        { onSuccess: () => setDeleteOpen(false) }
                    );
                }}
                isPending={deleteMutation.isPending}
            />
        </div>
    );
}
