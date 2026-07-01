/**
 * Indicator Linker — a dialog that lets users browse curriculum learning areas,
 * drill into strands → sub-strands, and select performance indicators to link
 * to a blueprint.
 *
 * Uses the real curriculum API (GET /api/v1/curriculum/learning-areas/:id/tree)
 * to present a selectable tree of performance indicators.
 */

"use client";

import * as React from "react";

import { useQuery } from "@tanstack/react-query";

import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert } from "@/components/ui/alert";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { Checkbox } from "@/components/ui/checkbox";
import { Search, ChevronRight, ChevronDown, Loader2 } from "lucide-react";

import { useLinkIndicators } from "../hooks/use-assessment";
import { listLearningAreas, getLearningAreaTree } from "@/lib/api/curriculum";
import type { StrandTree, SubStrandTree, PerformanceIndicator } from "@/lib/api/curriculum";
// ─── Props ─────────────────────────────────────────────────────────────────

interface IndicatorLinkerProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    blueprintId: string;
    alreadyLinked: string[]; // indicator IDs already linked
}

// ─── Sub-component: Indicator check row ────────────────────────────────────

function IndicatorCheckItem({
    indicator,
    checked,
    onToggle,
}: {
    indicator: PerformanceIndicator;
    checked: boolean;
    onToggle: (id: string) => void;
}) {
    return (
        <label className="hover:bg-muted/30 flex cursor-pointer items-start gap-2 rounded-sm px-2 py-1.5 transition-colors">
            <Checkbox
                checked={checked}
                onCheckedChange={() => onToggle(indicator.id)}
                className="mt-0.5"
            />
            <span className="text-sm leading-snug">{indicator.description}</span>
        </label>
    );
}

// ─── Sub-component: Sub-strand section ─────────────────────────────────────

function SubStrandSection({
    subStrand,
    selectedIds,
    onToggle,
}: {
    subStrand: SubStrandTree;
    selectedIds: Set<string>;
    onToggle: (id: string) => void;
}) {
    const [open, setOpen] = React.useState(false);
    const indicators = subStrand.performance_indicators;
    const checkedCount = indicators.filter((i) => selectedIds.has(i.id)).length;

    return (
        <Collapsible open={open} onOpenChange={setOpen}>
            <CollapsibleTrigger asChild>
                <button
                    type="button"
                    className="hover:bg-muted/30 flex w-full items-center gap-1.5 rounded-sm px-2 py-1.5 text-left text-sm font-medium transition-colors"
                >
                    {open ? (
                        <ChevronDown className="text-muted-foreground size-3.5 shrink-0" />
                    ) : (
                        <ChevronRight className="text-muted-foreground size-3.5 shrink-0" />
                    )}
                    <span className="truncate">{subStrand.name}</span>
                    {checkedCount > 0 && (
                        <Badge variant="secondary" className="ml-auto shrink-0 text-[10px]">
                            {checkedCount}
                        </Badge>
                    )}
                </button>
            </CollapsibleTrigger>
            <CollapsibleContent className="space-y-0.5 py-1 pl-5">
                {indicators.length === 0 ? (
                    <p className="text-muted-foreground px-2 text-xs">No performance indicators.</p>
                ) : (
                    indicators
                        .sort((a, b) => a.sequence_order - b.sequence_order)
                        .map((ind) => (
                            <IndicatorCheckItem
                                key={ind.id}
                                indicator={ind}
                                checked={selectedIds.has(ind.id)}
                                onToggle={onToggle}
                            />
                        ))
                )}
            </CollapsibleContent>
        </Collapsible>
    );
}

// ─── Sub-component: Strand section ─────────────────────────────────────────

function StrandSection({
    strand,
    selectedIds,
    onToggle,
}: {
    strand: StrandTree;
    selectedIds: Set<string>;
    onToggle: (id: string) => void;
}) {
    const [open, setOpen] = React.useState(false);

    // Count selected indicators across all sub-strands
    const totalSelected = strand.sub_strands.reduce(
        (acc, ss) => acc + ss.performance_indicators.filter((i) => selectedIds.has(i.id)).length,
        0
    );

    return (
        <Collapsible open={open} onOpenChange={setOpen}>
            <CollapsibleTrigger asChild>
                <button
                    type="button"
                    className="hover:bg-muted/40 flex w-full items-center gap-1.5 rounded-sm px-2 py-2 text-left text-sm font-semibold transition-colors"
                >
                    {open ? (
                        <ChevronDown className="text-muted-foreground size-4 shrink-0" />
                    ) : (
                        <ChevronRight className="text-muted-foreground size-4 shrink-0" />
                    )}
                    <span>{strand.name}</span>
                    {totalSelected > 0 && (
                        <Badge variant="secondary" className="ml-auto shrink-0 text-[10px]">
                            {totalSelected}
                        </Badge>
                    )}
                </button>
            </CollapsibleTrigger>
            <CollapsibleContent className="space-y-0.5 py-1 pl-4">
                {strand.sub_strands.map((ss) => (
                    <SubStrandSection
                        key={ss.id}
                        subStrand={ss}
                        selectedIds={selectedIds}
                        onToggle={onToggle}
                    />
                ))}
            </CollapsibleContent>
        </Collapsible>
    );
}

// ─── Sub-component: Learning Area Tree ─────────────────────────────────────

function LearningAreaBrowser({
    learningAreaId,
    selectedIds,
    onToggle,
}: {
    learningAreaId: string;
    selectedIds: Set<string>;
    onToggle: (id: string) => void;
}) {
    const {
        data: tree,
        isLoading,
        isError,
    } = useQuery({
        queryKey: ["learning-area-tree", learningAreaId],
        queryFn: () => getLearningAreaTree(learningAreaId),
        enabled: !!learningAreaId,
    });

    if (isLoading) {
        return (
            <div className="space-y-3 px-2 py-4">
                <Skeleton className="h-5 w-3/4" />
                <Skeleton className="h-4 w-1/2" />
                {Array.from({ length: 3 }).map((_, i) => (
                    <Skeleton key={i} className="h-8 w-full" />
                ))}
            </div>
        );
    }

    if (isError || !tree) {
        return (
            <p className="text-destructive px-2 py-4 text-sm">Failed to load curriculum tree.</p>
        );
    }

    return (
        <div className="py-2">
            {tree.strands.length === 0 ? (
                <p className="text-muted-foreground px-2 text-sm">
                    No strands in this learning area.
                </p>
            ) : (
                tree.strands.map((strand) => (
                    <StrandSection
                        key={strand.id}
                        strand={strand}
                        selectedIds={selectedIds}
                        onToggle={onToggle}
                    />
                ))
            )}
        </div>
    );
}

// ─── Main Component ────────────────────────────────────────────────────────

export function IndicatorLinker({
    open,
    onOpenChange,
    blueprintId,
    alreadyLinked,
}: IndicatorLinkerProps) {
    const [search, setSearch] = React.useState("");
    const [selectedIds, setSelectedIds] = React.useState<Set<string>>(new Set());
    const [expandedLearningArea, setExpandedLearningArea] = React.useState<string | null>(null);

    const linkIndicators = useLinkIndicators();

    // Fetch all learning areas
    const {
        data: learningAreasData,
        isLoading: areasLoading,
        isError: areasError,
    } = useQuery({
        queryKey: ["learning-areas"],
        queryFn: () => listLearningAreas(),
        enabled: open, // Only fetch when dialog opens
    });

    const learningAreas = learningAreasData?.learning_areas ?? [];

    // Build a filtered list of already-linked IDs for display
    const linkedSet = React.useMemo(() => new Set(alreadyLinked), [alreadyLinked]);

    const handleToggle = (indicatorId: string) => {
        setSelectedIds((prev) => {
            const next = new Set(prev);
            if (next.has(indicatorId)) {
                next.delete(indicatorId);
            } else {
                next.add(indicatorId);
            }
            return next;
        });
    };

    const handleLink = async () => {
        if (selectedIds.size === 0) return;
        try {
            await linkIndicators.mutateAsync({
                blueprintId,
                indicatorIds: Array.from(selectedIds),
            });
            onOpenChange(false);
        } catch {
            // Error handled by mutation onError → toast
        }
    };

    // Filter learning areas by search text
    const filteredAreas = search
        ? learningAreas.filter(
              (la) =>
                  la.name.toLowerCase().includes(search.toLowerCase()) ||
                  la.code.toLowerCase().includes(search.toLowerCase())
          )
        : learningAreas;

    // Count total unique selected (excluding already linked)
    const unlinkedSelected = Array.from(selectedIds).filter((id) => !linkedSet.has(id));

    return (
        <Dialog
            open={open}
            onOpenChange={(next) => {
                if (!next) {
                    setSearch("");
                    setSelectedIds(new Set());
                    setExpandedLearningArea(null);
                }
                onOpenChange(next);
            }}
        >
            <DialogContent className="flex max-h-[80vh] max-w-xl flex-col">
                <DialogHeader>
                    <DialogTitle>Link Indicators from Curriculum</DialogTitle>
                    <DialogDescription>
                        Browse learning areas and select performance indicators to link to this
                        blueprint. Already-linked indicators are shown below.
                    </DialogDescription>
                </DialogHeader>

                <div className="flex flex-1 flex-col gap-4 overflow-hidden">
                    {/* Search */}
                    <div className="relative">
                        <Search className="text-muted-foreground absolute top-2.5 left-2.5 size-4" />
                        <Input
                            placeholder="Search learning areas..."
                            value={search}
                            onChange={(e) => setSearch(e.target.value)}
                            className="pl-8"
                        />
                    </div>

                    {/* Already linked summary */}
                    {alreadyLinked.length > 0 && (
                        <p className="text-muted-foreground text-xs">
                            {alreadyLinked.length} indicator{alreadyLinked.length !== 1 ? "s" : ""}{" "}
                            already linked to this blueprint.
                        </p>
                    )}

                    {/* Selection summary */}
                    {unlinkedSelected.length > 0 && (
                        <div className="flex items-center gap-2">
                            <Badge variant="secondary" className="text-xs">
                                {unlinkedSelected.length} selected to link
                            </Badge>
                        </div>
                    )}

                    {/* Curriculum browser */}
                    <div className="flex-1 overflow-y-auto rounded-md border">
                        {areasLoading ? (
                            <div className="space-y-3 p-4">
                                {Array.from({ length: 4 }).map((_, i) => (
                                    <Skeleton key={i} className="h-8 w-full" />
                                ))}
                            </div>
                        ) : areasError ? (
                            <div className="p-4">
                                <Alert variant="destructive">
                                    Failed to load learning areas. Please try again.
                                </Alert>
                            </div>
                        ) : filteredAreas.length === 0 ? (
                            <div className="flex items-center justify-center p-8">
                                <p className="text-muted-foreground text-sm">
                                    {search
                                        ? "No learning areas match your search."
                                        : "No learning areas available for your school."}
                                </p>
                            </div>
                        ) : (
                            <div className="divide-y">
                                {filteredAreas.map((area) => (
                                    <div key={area.id} className="px-1 py-0.5">
                                        <button
                                            type="button"
                                            onClick={() =>
                                                setExpandedLearningArea(
                                                    expandedLearningArea === area.id
                                                        ? null
                                                        : area.id
                                                )
                                            }
                                            className="hover:bg-muted/30 flex w-full items-center gap-2 rounded-sm px-2 py-2 text-left transition-colors"
                                        >
                                            {expandedLearningArea === area.id ? (
                                                <ChevronDown className="text-muted-foreground size-4 shrink-0" />
                                            ) : (
                                                <ChevronRight className="text-muted-foreground size-4 shrink-0" />
                                            )}
                                            <span className="text-sm font-medium">{area.name}</span>
                                            <span className="text-muted-foreground ml-2 font-mono text-xs">
                                                {area.code}
                                            </span>
                                        </button>

                                        {expandedLearningArea === area.id && (
                                            <LearningAreaBrowser
                                                learningAreaId={area.id}
                                                selectedIds={selectedIds}
                                                onToggle={handleToggle}
                                            />
                                        )}
                                    </div>
                                ))}
                            </div>
                        )}
                    </div>

                    {/* Action buttons */}
                    <div className="flex items-center justify-end gap-3 border-t pt-3">
                        <Button
                            variant="ghost"
                            onClick={() => onOpenChange(false)}
                            disabled={linkIndicators.isPending}
                        >
                            Cancel
                        </Button>
                        <Button
                            onClick={handleLink}
                            disabled={unlinkedSelected.length === 0 || linkIndicators.isPending}
                        >
                            {linkIndicators.isPending ? (
                                <>
                                    <Loader2 className="mr-1.5 size-4 animate-spin" />
                                    Linking…
                                </>
                            ) : (
                                `Link Selected (${unlinkedSelected.length})`
                            )}
                        </Button>
                    </div>
                </div>
            </DialogContent>
        </Dialog>
    );
}
