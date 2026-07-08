/**
 * Curriculum Tree View — shows the full hierarchy of strands, sub-strands,
 * and performance indicators for a learning area.
 *
 * Three-tier expandable tree with CRUD actions at each level.
 */

"use client";

import * as React from "react";
import Link from "next/link";
import { Plus } from "lucide-react";

import { Button } from "@/components/ui/button";

import type { LearningAreaTree, PerformanceIndicator } from "@/lib/api/curriculum";
import { formatEducationLevel } from "@/lib/curriculum-filters";
import { StrandSection } from "./strand-section";
import { TreeSkeleton } from "./tree-skeleton";
import { CreateIndicatorDialog } from "./create-indicator-dialog";

// ─── Props ─────────────────────────────────────────────────────────────────

interface CurriculumTreeProps {
    tree: LearningAreaTree | undefined;
    isLoading: boolean;
    isError: boolean;
}

// ─── Main Component ────────────────────────────────────────────────────────

export function CurriculumTree({ tree, isLoading, isError }: CurriculumTreeProps) {
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
                <Button variant="outline" size="sm" asChild>
                    <Link href={`/curriculum/new-strand?learningAreaId=${tree.id}`}>
                        <Plus className="mr-1.5 size-3.5" />
                        Add Strand
                    </Link>
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
                    <Button variant="outline" size="sm" className="mt-2" asChild>
                        <Link href={`/curriculum/new-strand?learningAreaId=${tree.id}`}>
                            <Plus className="mr-1.5 size-3.5" />
                            Add Strand
                        </Link>
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
                        />
                    ))}
                </div>
            )}

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
