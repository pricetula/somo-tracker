"use client";

import { useState, useMemo } from "react";
import Link from "next/link";
import { StaticTable } from "@/components/shared/static-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { buttonVariants } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { useStreams, useDeleteStreams } from "@/features/streams";
import type { Stream } from "@/features/streams";

export default function StreamsPage() {
    const { data, isLoading, isError } = useStreams();
    const deleteStreams = useDeleteStreams();
    const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());

    const columns = useMemo<DataTableColumn<Stream>[]>(
        () => [
            {
                id: "name",
                header: "Name",
                cell: (row) => (
                    <Link
                        href={`/settings/streams/${row.id}`}
                        className="hover:text-foreground underline underline-offset-2"
                    >
                        {row.name}
                    </Link>
                ),
                width: "1fr",
            },
            {
                id: "color",
                header: "Color",
                cell: (row) => (
                    <span className="inline-flex items-center gap-2">
                        {row.color ? (
                            <span
                                className="inline-block h-3 w-3 rounded-full border shadow-sm"
                                style={{ backgroundColor: row.color }}
                            />
                        ) : (
                            <span className="text-muted-foreground text-xs">—</span>
                        )}
                    </span>
                ),
                width: "60px",
            },
        ],
        []
    );

    const handleDelete = () => {
        if (selectedIds.size === 0) return;
        deleteStreams.mutate(Array.from(selectedIds));
        setSelectedIds(new Set());
    };

    if (isError) {
        return (
            <div className="space-y-4">
                <h1 className="text-2xl font-semibold">Streams</h1>
                <Alert variant="destructive">
                    <AlertTitle>Error</AlertTitle>
                    <AlertDescription>Failed to load streams. Please try again.</AlertDescription>
                </Alert>
            </div>
        );
    }

    return (
        <StaticTable<Stream>
            addHref="/settings/streams/add"
            columns={columns}
            data={data ?? []}
            getRowId={(row) => row.id}
            height={480}
            rowHeight={44}
            isCheckable
            isLoading={isLoading}
            selectedIds={selectedIds}
            onSelectionChange={setSelectedIds}
        />
    );
}
