/**
 * Blueprint Detail page.
 *
 * Shows blueprint metadata + linked indicators + available indicators to add.
 * Maps to GET /api/v1/assessment/blueprints/:id.
 */

"use client";

import * as React from "react";
import { useParams, useRouter } from "next/navigation";

import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { ArrowLeft, Link2, Trash2 } from "lucide-react";

import { useBlueprintDetail, useUnlinkIndicator, IndicatorLinker } from "@/features/assessment";

function typeLabel(type: string): string {
    return type.replace(/_/g, " ").replace(/\b\w/g, (c) => c.toUpperCase());
}

export default function BlueprintDetailPage() {
    const params = useParams();
    const router = useRouter();
    const id = params.id as string;

    const { data: detailData, isLoading, isError } = useBlueprintDetail(id);

    const unlinkIndicator = useUnlinkIndicator();
    const [linkerOpen, setLinkerOpen] = React.useState(false);

    const detail = detailData?.data;

    if (isLoading) {
        return (
            <div className="flex flex-col gap-4 px-6 pt-6 pb-8">
                <Skeleton className="h-8 w-64" />
                <Skeleton className="h-4 w-48" />
                <Skeleton className="mt-4 h-32 w-full" />
            </div>
        );
    }

    if (isError || !detail) {
        return (
            <div className="flex items-center justify-center py-16">
                <div className="text-center">
                    <p className="text-destructive text-sm font-medium">
                        Failed to load blueprint details.
                    </p>
                    <Button
                        variant="outline"
                        size="sm"
                        className="mt-4"
                        onClick={() => router.push("/assessment/blueprints")}
                    >
                        Back to Blueprints
                    </Button>
                </div>
            </div>
        );
    }

    const handleUnlink = async (indicatorId: string) => {
        if (window.confirm("Remove this indicator from the blueprint?")) {
            unlinkIndicator.mutate({ blueprintId: id, indicatorId });
        }
    };

    return (
        <div className="flex flex-1 flex-col px-6 pt-6 pb-8">
            {/* Back link */}
            <Button
                variant="ghost"
                size="sm"
                className="mb-4 w-fit"
                onClick={() => router.push("/assessment/blueprints")}
            >
                <ArrowLeft className="mr-1.5 size-4" />
                Back to Blueprints
            </Button>

            {/* Blueprint metadata */}
            <div className="mb-6">
                <h1 className="text-2xl font-semibold tracking-tight">{detail.title}</h1>
                <div className="mt-2 flex flex-wrap items-center gap-3">
                    <Badge variant="secondary">{typeLabel(detail.type)}</Badge>
                    <span className="text-muted-foreground text-sm">{detail.grade_level}</span>
                    <span className="text-muted-foreground text-sm">
                        Year {detail.academic_year}, Term {detail.term}
                    </span>
                </div>
            </div>

            {/* Linked Indicators section */}
            <div className="mb-6">
                <div className="mb-3 flex items-center justify-between">
                    <h2 className="text-lg font-medium">Linked Indicators</h2>
                    <Button variant="outline" size="sm" onClick={() => setLinkerOpen(true)}>
                        <Link2 className="mr-1.5 size-3.5" />
                        Link Indicators
                    </Button>
                </div>

                {detail.indicators.length === 0 ? (
                    <div className="bg-muted/30 flex items-center justify-center rounded-md px-4 py-8">
                        <div className="text-center">
                            <p className="text-muted-foreground text-sm font-medium">
                                No indicators linked yet
                            </p>
                            <p className="text-muted-foreground mt-1 text-xs">
                                Link performance indicators from the curriculum to define what this
                                assessment measures.
                            </p>
                        </div>
                    </div>
                ) : (
                    <div className="ring-foreground/10 rounded-lg ring-1">
                        <table className="w-full">
                            <thead>
                                <tr className="border-border/40 border-b">
                                    <th className="text-muted-foreground px-3 py-2 text-left text-xs font-medium tracking-wider uppercase">
                                        Indicator
                                    </th>
                                    <th className="w-16 px-3 py-2" />
                                </tr>
                            </thead>
                            <tbody>
                                {detail.indicators.map((ind) => (
                                    <tr
                                        key={ind.id}
                                        className="group border-border/40 hover:bg-muted/30 border-b transition-colors"
                                    >
                                        <td className="px-3 py-2.5 text-sm">{ind.description}</td>
                                        <td className="px-3 py-2.5">
                                            <Button
                                                variant="ghost"
                                                size="icon-sm"
                                                className="opacity-0 transition-opacity group-hover:opacity-100"
                                                onClick={() => handleUnlink(ind.id)}
                                                title="Remove indicator"
                                            >
                                                <Trash2 className="text-destructive size-3.5" />
                                                <span className="sr-only">Remove</span>
                                            </Button>
                                        </td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>
                )}
            </div>

            {/* Indicator Linker Dialog */}
            <IndicatorLinker
                open={linkerOpen}
                onOpenChange={setLinkerOpen}
                blueprintId={id}
                alreadyLinked={detail.indicators.map((i) => i.id)}
            />
        </div>
    );
}
