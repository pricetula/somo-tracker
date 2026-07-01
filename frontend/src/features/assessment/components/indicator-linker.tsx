/**
 * Indicator Linker — a dialog that lets users browse curriculum learning areas,
 * drill into strands → sub-strands, and select performance indicators to link
 * to a blueprint.
 */

"use client";

import * as React from "react";

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
import { Search, Loader2 } from "lucide-react";

import { useLinkIndicators } from "../hooks/use-assessment";

// ─── Props ─────────────────────────────────────────────────────────────────

interface IndicatorLinkerProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    blueprintId: string;
    alreadyLinked: string[]; // indicator IDs already linked
}

// ─── Component ─────────────────────────────────────────────────────────────

export function IndicatorLinker({
    open,
    onOpenChange,
    blueprintId,
    alreadyLinked,
}: IndicatorLinkerProps) {
    const [search, setSearch] = React.useState("");
    const [selectedIds, setSelectedIds] = React.useState<Set<string>>(new Set());

    const linkIndicators = useLinkIndicators();

    const handleLink = async () => {
        if (selectedIds.size === 0) return;
        try {
            await linkIndicators.mutateAsync({
                blueprintId,
                indicatorIds: Array.from(selectedIds),
            });
            onOpenChange(false);
        } catch {
            // Error handled by mutation onError
        }
    };

    const availableForSelection = Array.from(selectedIds).filter(
        (id) => !alreadyLinked.includes(id)
    );

    return (
        <Dialog
            open={open}
            onOpenChange={(next) => {
                if (!next) {
                    // Reset state when dialog closes
                    setSearch("");
                    setSelectedIds(new Set());
                }
                onOpenChange(next);
            }}
        >
            <DialogContent className="max-w-lg">
                <DialogHeader>
                    <DialogTitle>Link Indicators</DialogTitle>
                    <DialogDescription>
                        Select performance indicators from the curriculum to link to this blueprint.
                    </DialogDescription>
                </DialogHeader>

                <div className="space-y-4">
                    {/* Search */}
                    <div className="relative">
                        <Search className="text-muted-foreground absolute top-2.5 left-2.5 size-4" />
                        <Input
                            placeholder="Search indicators..."
                            value={search}
                            onChange={(e) => setSearch(e.target.value)}
                            className="pl-8"
                        />
                    </div>

                    {/* Selection summary */}
                    {selectedIds.size > 0 && (
                        <div className="text-muted-foreground flex items-center gap-2 text-sm">
                            <Badge variant="secondary">{selectedIds.size} selected</Badge>
                            {availableForSelection.length < selectedIds.size && (
                                <span className="text-xs">
                                    ({selectedIds.size - availableForSelection.length} already
                                    linked)
                                </span>
                            )}
                        </div>
                    )}

                    {/* Curriculum browser placeholder — simplified */}
                    <div className="bg-muted/30 flex min-h-[240px] items-center justify-center rounded-md px-4 py-8">
                        <div className="text-center">
                            <p className="text-muted-foreground text-sm font-medium">
                                Curriculum Browser
                            </p>
                            <p className="text-muted-foreground mt-1 max-w-xs text-xs">
                                Select indicators from the curriculum tree to link to this
                                blueprint. Expand learning areas to browse strands, sub-strands, and
                                performance indicators.
                            </p>
                            <p className="text-muted-foreground mt-3 text-xs">
                                (Full curriculum tree integration will be available in a subsequent
                                update. For now, indicator IDs can be passed via the API.)
                            </p>
                        </div>
                    </div>

                    {/* Action buttons */}
                    <div className="flex items-center justify-end gap-3">
                        <Button
                            variant="ghost"
                            onClick={() => onOpenChange(false)}
                            disabled={linkIndicators.isPending}
                        >
                            Cancel
                        </Button>
                        <Button
                            onClick={handleLink}
                            disabled={selectedIds.size === 0 || linkIndicators.isPending}
                        >
                            {linkIndicators.isPending ? (
                                <>
                                    <Loader2 className="mr-1.5 size-4 animate-spin" />
                                    Linking…
                                </>
                            ) : (
                                `Link (${selectedIds.size})`
                            )}
                        </Button>
                    </div>
                </div>
            </DialogContent>
        </Dialog>
    );
}
