/**
 * SchoolsList — table listing schools from backend.
 * Uses StaticTable for display.
 */

"use client";

import Link from "next/link";
import { useMemo } from "react";
import { StaticTable } from "@/components/shared/static-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";
import { buttonVariants } from "@/components/ui/button";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { useSchoolsList } from "../hooks/use-schools-list";
import type { SchoolItem } from "@/lib/api/schools";
import { cn } from "@/lib/utils";

export function SchoolsTable() {
    const { data, isLoading, isError } = useSchoolsList();

    const columns = useMemo<DataTableColumn<SchoolItem>[]>(
        () => [
            {
                id: "name",
                header: "School Name",
                cell: (row) => row.name,
                width: "2fr",
            },
            {
                id: "country",
                header: "Country",
                cell: (row) => row.country_name ?? "—",
                width: "1fr",
            },
            {
                id: "education",
                header: "Education System",
                cell: (row) => row.education_system_name ?? "—",
                width: "1fr",
                align: "right",
            },
        ],
        []
    );

    if (isLoading) {
        return (
            <div className="space-y-4">
                <div className="flex items-center justify-between">
                    <h1 className="text-2xl font-semibold">Schools</h1>
                    <Link
                        href="/schools/new"
                        className={cn(buttonVariants(), "pointer-events-none opacity-50")}
                    >
                        New School
                    </Link>
                </div>
                <div className="text-muted-foreground h-[480px] rounded-md border p-4 text-sm">
                    Loading schools…
                </div>
            </div>
        );
    }

    if (isError || !data) {
        return (
            <div className="space-y-4">
                <div className="flex items-center justify-between">
                    <h1 className="text-2xl font-semibold">Schools</h1>
                    <Link href="/schools/new" className={buttonVariants()}>
                        New School
                    </Link>
                </div>
                <Alert variant="destructive">
                    <AlertTitle>Error</AlertTitle>
                    <AlertDescription>Failed to load schools. Please try again.</AlertDescription>
                </Alert>
            </div>
        );
    }

    const schools = data ?? [];

    return (
        <div className="space-y-4">
            <div className="flex items-center justify-between">
                <h1 className="text-2xl font-semibold">Schools</h1>
                <Link href="/schools/new" className={buttonVariants()}>
                    New School
                </Link>
            </div>

            <StaticTable<SchoolItem>
                columns={columns}
                data={schools}
                getRowId={(row) => row.id}
                height={480}
                rowHeight={44}
            />
        </div>
    );
}
