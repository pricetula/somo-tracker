"use client";

import { useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Badge } from "@/components/ui/badge";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { ScoreLevelBadge } from "./score-level-badge";
import type { StudentRow, InterventionRow } from "./types";

function EmptyState({ tab: _tab }: { tab: string }) {
    void _tab;
    return (
        <div className="text-muted-foreground py-8 text-center text-xs">
            No students in this category for the current term
        </div>
    );
}

function DeltaSpan({ delta, show }: { delta: number; show: boolean }) {
    if (!show) return null;
    const isPositive = delta > 0;
    return (
        <span className={`ml-auto text-xs ${isPositive ? "text-emerald-600" : "text-red-600"}`}>
            {isPositive ? "↑" : "↓"} {Math.abs(delta).toFixed(1)}
        </span>
    );
}

function StudentFlexRow({
    row,
    showDelta,
    deltaAlways,
}: {
    row: StudentRow;
    showDelta: boolean;
    deltaAlways?: boolean;
}) {
    return (
        <div className="border-border/50 flex items-center gap-2 border-b py-2 last:border-0">
            <span className="text-muted-foreground w-6 text-center font-mono text-xs">
                {row.rank}
            </span>
            <span className="flex-1 text-xs font-medium">{row.name}</span>
            <span className="bg-muted text-muted-foreground rounded px-1.5 py-0.5 text-[10px]">
                {row.class}
            </span>
            <ScoreLevelBadge level={row.scoreLevel} />
            <DeltaSpan delta={row.delta} show={showDelta || (deltaAlways ?? false)} />
        </div>
    );
}

function InterventionFlexRow({ row }: { row: InterventionRow }) {
    const [open, setOpen] = useState(false);

    return (
        <Collapsible
            open={open}
            onOpenChange={setOpen}
            className="border-border/50 border-b py-2 last:border-0"
        >
            <CollapsibleTrigger asChild>
                <div className="flex cursor-pointer items-center gap-2 text-xs">
                    <span className="flex-1 font-medium">{row.name}</span>
                    <span className="bg-muted text-muted-foreground rounded px-1.5 py-0.5 text-[10px]">
                        {row.class}
                    </span>
                    <Badge variant="destructive" className="text-[10px]">
                        {row.beAreaCount}
                    </Badge>
                    <span className="text-muted-foreground text-[10px]">▼</span>
                </div>
            </CollapsibleTrigger>
            <CollapsibleContent>
                <div className="mt-1.5 ml-6 flex flex-wrap gap-1">
                    {row.areas.map((a) => (
                        <span
                            key={a}
                            className="bg-muted text-muted-foreground rounded px-1.5 py-0.5 text-[10px]"
                        >
                            {a}
                        </span>
                    ))}
                </div>
            </CollapsibleContent>
        </Collapsible>
    );
}

type TabData = {
    type: "student";
    label: string;
    rows: StudentRow[];
    showDelta: boolean;
};

type InterventionTabData = {
    type: "intervention";
    label: string;
    rows: InterventionRow[];
};

export function PerformanceMatrix({
    topPerformers,
    mostImproved,
    regressed,
    interventionList,
}: {
    topPerformers: StudentRow[];
    mostImproved: StudentRow[];
    regressed: StudentRow[];
    interventionList: InterventionRow[];
}) {
    const studentTabs: TabData[] = [
        { type: "student", label: "Top performers", rows: topPerformers, showDelta: false },
        { type: "student", label: "Most improved", rows: mostImproved, showDelta: true },
        { type: "student", label: "Regressed", rows: regressed, showDelta: true },
    ];

    const interventionTab: InterventionTabData = {
        type: "intervention",
        label: "Intervention",
        rows: interventionList,
    };

    return (
        <Card>
            <CardHeader>
                <CardTitle>CBC Student Performance Matrix</CardTitle>
            </CardHeader>
            <CardContent>
                <Tabs defaultValue="top-performers">
                    <TabsList className="mb-3 w-full justify-start gap-0 overflow-x-auto">
                        {studentTabs.map((t) => (
                            <TabsTrigger
                                key={t.label}
                                value={t.label.toLowerCase().replace(/\s+/g, "-")}
                                className="text-xs"
                            >
                                {t.label} ({t.rows.length})
                            </TabsTrigger>
                        ))}
                        <TabsTrigger value="intervention" className="text-xs">
                            {interventionTab.label} ({interventionTab.rows.length})
                        </TabsTrigger>
                    </TabsList>

                    {studentTabs.map((t) => (
                        <TabsContent
                            key={t.label}
                            value={t.label.toLowerCase().replace(/\s+/g, "-")}
                        >
                            {t.rows.length === 0 ? (
                                <EmptyState tab={t.label} />
                            ) : (
                                <div className="flex flex-col">
                                    {t.rows.map((r) => (
                                        <StudentFlexRow
                                            key={`${r.name}-${r.rank}`}
                                            row={r}
                                            showDelta={t.showDelta}
                                        />
                                    ))}
                                </div>
                            )}
                        </TabsContent>
                    ))}

                    <TabsContent value="intervention">
                        {interventionTab.rows.length === 0 ? (
                            <EmptyState tab="Intervention" />
                        ) : (
                            <div className="flex flex-col">
                                {interventionTab.rows.map((r) => (
                                    <InterventionFlexRow key={r.name} row={r} />
                                ))}
                            </div>
                        )}
                    </TabsContent>
                </Tabs>
            </CardContent>
        </Card>
    );
}
