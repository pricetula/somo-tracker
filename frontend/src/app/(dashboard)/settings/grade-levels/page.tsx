"use client";

import { useMemo } from "react";
import { StaticTable } from "@/components/shared/static-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { useGrades } from "@/features/grades";
import type { GradeLevel } from "@/features/grades";

export default function GradeLevelsPage() {
    const { data, isLoading, isError } = useGrades();

    const columns = useMemo<DataTableColumn<GradeLevel>[]>(
        () => [
            {
                id: "local_label",
                header: "Local Label",
                cell: (row) => row.local_label ?? "—",
                width: "1fr",
            },
            {
                id: "tier_stage",
                header: "Tier / Stage",
                cell: (row) => row.tier_stage ?? "—",
                width: "1fr",
            },
        ],
        []
    );

    if (isLoading) {
        return (
            <div className="space-y-4">
                <h1 className="text-2xl font-semibold">Grade Levels</h1>
                <div className="text-muted-foreground h-[480px] rounded-md border p-4 text-sm">
                    Loading grade levels…
                </div>
            </div>
        );
    }

    if (isError || !data) {
        return (
            <div className="space-y-4">
                <h1 className="text-2xl font-semibold">Grade Levels</h1>
                <Alert variant="destructive">
                    <AlertTitle>Error</AlertTitle>
                    <AlertDescription>
                        Failed to load grade levels. Please try again.
                    </AlertDescription>
                </Alert>
            </div>
        );
    }

    return (
        <div className="space-y-4">
            <h1 className="text-2xl font-semibold">Grade Levels</h1>
            <StaticTable<GradeLevel>
                columns={columns}
                data={data ?? []}
                getRowId={(row) => row.id}
                height={480}
                rowHeight={44}
            />
        </div>
    );
}
