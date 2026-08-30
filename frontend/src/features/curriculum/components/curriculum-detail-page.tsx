/**
 * Learning area detail page — feature container.
 * Shows learning area info + strands DataTable.
 */

"use client";

import * as React from "react";
import Link from "next/link";
import { BookOpen } from "lucide-react";

import { DataTable } from "@/components/shared/data-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";

import { listStrands, type Strand } from "@/lib/api/curriculum";
import { useLearningAreaTree } from "../hooks/use-curriculum";

interface CurriculumDetailProps {
    learningAreaId: string;
}

export function CurriculumDetailPage({ learningAreaId }: CurriculumDetailProps) {
    const { data: tree } = useLearningAreaTree(learningAreaId, {
        enabled: !!learningAreaId,
    });

    const columns = React.useMemo<DataTableColumn<Strand>[]>(
        () => [
            {
                id: "name",
                header: "Strand",
                cell: (row) => (
                    <Link
                        href={`/curriculum/${learningAreaId}/strands/${row.id}`}
                        className="font-medium hover:underline"
                    >
                        {row.name}
                    </Link>
                ),
            },
        ],
        [learningAreaId]
    );

    return (
        <>
            <div className="space-y-2">
                <nav className="text-muted-foreground font-mono text-xs">
                    <Link href="/curriculum" className="hover:text-foreground">
                        Curriculum
                    </Link>
                    <span className="mx-1">/</span>
                    <span>{tree?.name ?? "Loading…"}</span>
                </nav>
                <h1 className="flex items-center gap-2 text-2xl font-semibold tracking-tight">
                    <BookOpen className="text-primary size-6" />
                    {tree?.name ?? "Loading…"}
                </h1>
                {tree && (
                    <p className="text-muted-foreground flex items-center gap-2 text-sm">
                        <span className="font-mono text-xs">{tree.code}</span>
                        <span>·</span>
                        <span>{tree.education_level}</span>
                        <span>·</span>
                        <span>{tree.grade_level}</span>
                    </p>
                )}
            </div>

            <section className="space-y-3">
                <h2 className="text-base font-medium">Strands</h2>
                <DataTable
                    queryKey={["curriculum", "strands", learningAreaId]}
                    queryFn={(params: { page?: number; limit?: number; search?: string }) =>
                        listStrands({
                            learning_area_id: learningAreaId,
                            search: params.search,
                            page: params.page,
                            limit: params.limit,
                        })
                    }
                    columns={columns}
                    getRowId={(row) => row.id}
                    isSearchable
                    searchPlaceholder="Search strands..."
                    emptyState="No strands yet."
                    noResultsState="No strands match your search."
                    renderToolBarComponents={() => null}
                />
            </section>
        </>
    );
}
