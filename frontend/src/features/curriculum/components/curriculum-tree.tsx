/**
 * Curriculum Tree View — shows the full hierarchy of strands, sub-strands,
 * and performance indicators for a learning area.
 *
 * Three-tier expandable tree with CRUD actions at each level.
 */

"use client";

import * as React from "react";
import { ChevronDown, ChevronRight, GripVertical, Plus, Pencil, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";

import type {
    LearningAreaTree,
    StrandTree,
    SubStrandTree,
    PerformanceIndicator,
} from "@/lib/api/curriculum";
import {
    useDeleteStrand,
    useDeleteSubStrand,
    useDeletePerformanceIndicator,
} from "../hooks/use-curriculum";
import { CreateStrandDialog } from "./create-strand-dialog";
import { CreateSubStrandDialog } from "./create-sub-strand-dialog";
import { CreateIndicatorDialog } from "./create-indicator-dialog";
import { DeleteConfirmDialog } from "./delete-confirm-dialog";

// ─── Education Level Labels ───────────────────────────────────────────────

const EDUCATION_LEVEL_LABELS: Record<string, string> = {
    Early_Years: "Early Years",
    Upper_Primary: "Upper Primary",
    Junior_Secondary: "Junior Secondary",
    Senior_School: "Senior School",
};

function formatEducationLevel(level: string): string {
    return EDUCATION_LEVEL_LABELS[level] ?? level;
}

// ─── Props ─────────────────────────────────────────────────────────────────

interface CurriculumTreeProps {
    tree: LearningAreaTree | undefined;
    isLoading: boolean;
    isError: boolean;
}

// ─── Sub-Components ────────────────────────────────────────────────────────

function PerformanceIndicatorRow({
    indicator,
    learningAreaId,
    onEdit,
}: {
    indicator: PerformanceIndicator;
    learningAreaId: string;
    onEdit: (indicator: PerformanceIndicator) => void;
}) {
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
                description="Are you sure you want to delete this performance indicator? This action cannot be undone."
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

function SubStrandSection({
    subStrand,
    learningAreaId,
    onEditIndicator,
}: {
    subStrand: SubStrandTree;
    learningAreaId: string;
    onEditIndicator: (indicator: PerformanceIndicator) => void;
}) {
    const [expanded, setExpanded] = React.useState(false);
    const [createIndicatorOpen, setCreateIndicatorOpen] = React.useState(false);
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
                    <Button
                        variant="ghost"
                        size="icon-sm"
                        onClick={() => setCreateIndicatorOpen(true)}
                        aria-label="Add indicator"
                    >
                        <Plus className="size-3.5" />
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

            {/* Create Indicator Dialog */}
            <CreateIndicatorDialog
                open={createIndicatorOpen}
                onOpenChange={setCreateIndicatorOpen}
                subStrandId={subStrand.id}
            />

            {/* Delete Sub-Strand Dialog */}
            <DeleteConfirmDialog
                open={deleteOpen}
                onOpenChange={setDeleteOpen}
                title="Delete Sub-Strand"
                description="Are you sure you want to delete this sub-strand and all its performance indicators? This action cannot be undone."
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

function StrandSection({
    strand,
    learningAreaId,
    onEditIndicator,
    onCreateSubStrand,
}: {
    strand: StrandTree;
    learningAreaId: string;
    onEditIndicator: (indicator: PerformanceIndicator) => void;
    onCreateSubStrand: (strandId: string) => void;
}) {
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
                    <span className="text-sm font-semibold">{strand.name}</span>
                </button>
                <Badge className="text-xs font-normal">
                    {subStrandCount} sub-strand{subStrandCount !== 1 ? "s" : ""}
                </Badge>
                <div className="ml-auto flex items-center gap-1 opacity-0 transition-opacity group-hover:opacity-100">
                    <Button
                        variant="ghost"
                        size="icon-sm"
                        onClick={() => onCreateSubStrand(strand.id)}
                        aria-label="Add sub-strand"
                    >
                        <Plus className="size-3.5" />
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
                        <p className="text-muted-foreground py-2 pl-8 text-xs">
                            No sub-strands yet.
                        </p>
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
                description="Are you sure you want to delete this strand, all its sub-strands, and performance indicators? This action cannot be undone."
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

// ─── Loading Skeleton ──────────────────────────────────────────────────────

function TreeSkeleton() {
    return (
        <div className="space-y-4">
            <Skeleton className="h-8 w-64" />
            <Skeleton className="h-6 w-48" />
            {Array.from({ length: 3 }).map((_, i) => (
                <div key={i} className="space-y-2">
                    <Skeleton className="h-8 w-full" />
                    <Skeleton className="ml-6 h-6 w-3/4" />
                    <Skeleton className="ml-12 h-5 w-1/2" />
                </div>
            ))}
        </div>
    );
}

// ─── Main Component ────────────────────────────────────────────────────────

export function CurriculumTree({ tree, isLoading, isError }: CurriculumTreeProps) {
    const [createStrandOpen, setCreateStrandOpen] = React.useState(false);
    const [createSubStrandStrandId, setCreateSubStrandStrandId] = React.useState<string | null>(
        null
    );
    const [editIndicator, setEditIndicator] = React.useState<PerformanceIndicator | null>(null);

    if (isLoading) {
        return <TreeSkeleton />;
    }

    if (isError || !tree) {
        return (
            <div className="flex items-center justify-center py-16">
                <p className="text-destructive text-sm">
                    Failed to load curriculum tree. Please try again.
                </p>
            </div>
        );
    }

    const strandCount = tree.strands.length;

    return (
        <div>
            {/* Learning Area Header */}
            <div className="flex items-start justify-between gap-4">
                <div>
                    <h2 className="text-xl font-semibold">{tree.name}</h2>
                    <div className="text-muted-foreground mt-1 flex items-center gap-3 text-sm">
                        <span className="font-mono text-xs">{tree.code}</span>
                        <span>{formatEducationLevel(tree.education_level)}</span>
                        <span>
                            {strandCount} strand{strandCount !== 1 ? "s" : ""}
                        </span>
                    </div>
                </div>
                <Button variant="outline" size="sm" onClick={() => setCreateStrandOpen(true)}>
                    <Plus className="mr-1.5 size-3.5" />
                    Add Strand
                </Button>
            </div>

            {/* Divider */}
            <div className="bg-border/40 my-5 h-px" />

            {/* Strands */}
            {strandCount === 0 ? (
                <div className="flex flex-col items-center gap-2 py-16">
                    <p className="text-muted-foreground text-sm font-medium">No strands yet</p>
                    <p className="text-muted-foreground text-xs">
                        Add a strand to start building your curriculum.
                    </p>
                    <Button
                        variant="outline"
                        size="sm"
                        className="mt-2"
                        onClick={() => setCreateStrandOpen(true)}
                    >
                        <Plus className="mr-1.5 size-3.5" />
                        Add Strand
                    </Button>
                </div>
            ) : (
                <div className="space-y-1">
                    {tree.strands.map((strand) => (
                        <StrandSection
                            key={strand.id}
                            strand={strand}
                            learningAreaId={tree.id}
                            onEditIndicator={setEditIndicator}
                            onCreateSubStrand={(strandId) => setCreateSubStrandStrandId(strandId)}
                        />
                    ))}
                </div>
            )}

            {/* Create Strand Dialog */}
            <CreateStrandDialog
                open={createStrandOpen}
                onOpenChange={setCreateStrandOpen}
                learningAreaId={tree.id}
            />

            {/* Create Sub-Strand Dialog */}
            <CreateSubStrandDialog
                open={!!createSubStrandStrandId}
                onOpenChange={(open) => {
                    if (!open) setCreateSubStrandStrandId(null);
                }}
                strandId={createSubStrandStrandId ?? ""}
            />

            {/* Edit Indicator Dialog */}
            <CreateIndicatorDialog
                open={!!editIndicator}
                onOpenChange={(open) => {
                    if (!open) setEditIndicator(null);
                }}
                subStrandId={editIndicator?.sub_strand_id ?? ""}
                indicator={editIndicator}
            />
        </div>
    );
}
