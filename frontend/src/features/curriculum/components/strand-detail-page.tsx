/**
 * Strand detail page — feature container.
 * Shows strand info + sub-strands DataTable.
 */

"use client";

import * as React from "react";
import Link from "next/link";
import { BookMarked } from "lucide-react";
import { useQuery } from "@tanstack/react-query";

import { DataTable } from "@/components/shared/data-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";

import { listSubStrands, getStrand, type SubStrand } from "@/lib/api/curriculum";

interface StrandDetailProps {
    learningAreaId: string;
    strandId: string;
}

export function StrandDetailPage({ learningAreaId, strandId }: StrandDetailProps) {
    const { data: strand } = useQuery({
        queryKey: ["curriculum", "strand", strandId],
        queryFn: () => getStrand(strandId),
        enabled: !!strandId,
    });

    const columns = React.useMemo<DataTableColumn<SubStrand>[]>(
        () => [
            {
                id: "name",
                header: "Sub-Strand",
                cell: (row) => (
                    <Link
                        href={`/curriculum/${learningAreaId}/strands/${strandId}/sub-strands/${row.id}`}
                        className="font-medium hover:underline"
                    >
                        {row.name}
                    </Link>
                ),
            },
        ],
        [learningAreaId, strandId]
    );

    return (
        <>
            <div className="space-y-2">
                <nav className="text-muted-foreground font-mono text-xs">
                    <Link href="/curriculum" className="hover:text-foreground">
                        Curriculum
                    </Link>
                    <span className="mx-1">/</span>
                    <Link href={`/curriculum/${learningAreaId}`} className="hover:text-foreground">
                        {strand?.learning_area_id ?? "…"}
                    </Link>
                    <span className="mx-1">/</span>
                    <span>{strand?.name ?? "Loading…"}</span>
                </nav>
                <h1 className="flex items-center gap-2 text-2xl font-semibold tracking-tight">
                    <BookMarked className="text-primary size-6" />
                    {strand?.name ?? "Loading…"}
                </h1>
            </div>

            <section className="space-y-3">
                <h2 className="text-base font-medium">Sub-Strands</h2>
                <DataTable
                    queryKey={["curriculum", "sub-strands", strandId]}
                    queryFn={(params: { page?: number; limit?: number; search?: string }) =>
                        listSubStrands({
                            strand_id: strandId,
                            search: params.search,
                            page: params.page,
                            limit: params.limit,
                        })
                    }
                    columns={columns}
                    getRowId={(row) => row.id}
                    isSearchable
                    searchPlaceholder="Search sub-strands..."
                    emptyState="No sub-strands yet."
                    noResultsState="No sub-strands match your search."
                    renderToolBarComponents={() => null}
                />
            </section>
        </>
    );
}
