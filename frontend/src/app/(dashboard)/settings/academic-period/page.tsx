"use client";

import { useMemo } from "react";
import Link from "next/link";
import { StaticTable } from "@/components/shared/static-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";
import { useAcademicPeriods } from "@/features/school/hooks/use-academic-periods";

type PeriodRow = { periodName: string; termName: string; startDate: string; endDate: string };

export default function AcademicPeriodPage() {
    const { data, isLoading, isError } = useAcademicPeriods();

    const columns = useMemo<DataTableColumn<PeriodRow>[]>(
        () => [
            {
                id: "period",
                header: "Period",
                cell: (row) => <span className="font-medium">{row.periodName}</span>,
                width: "160px",
            },
            {
                id: "term",
                header: "Term",
                cell: (row) => row.termName,
                width: "1fr",
            },
            {
                id: "startDate",
                header: "Start Date",
                cell: (row) => row.startDate,
                width: "120px",
            },
            {
                id: "endDate",
                header: "End Date",
                cell: (row) => row.endDate,
                width: "120px",
            },
        ],
        []
    );

    const flatData = useMemo(() => {
        if (!data) return [];
        const rows: PeriodRow[] = [];
        for (const period of data) {
            for (const term of period.terms) {
                rows.push({
                    periodName: period.name,
                    termName: term.name,
                    startDate: term.start_date,
                    endDate: term.end_date,
                });
            }
        }
        return rows;
    }, [data]);

    if (isError) {
        return (
            <div className="space-y-6 p-6">
                <div className="flex items-center justify-between">
                    <h1 className="text-2xl font-semibold">Academic Periods</h1>
                    <Link
                        href="/settings/academic-period/add"
                        className="bg-primary text-primary-foreground hover:bg-primary/90 inline-flex h-10 items-center justify-center rounded-md px-4 py-2 text-sm font-medium"
                    >
                        Add Period
                    </Link>
                </div>
                <div className="border-destructive bg-destructive/10 text-destructive rounded-lg border p-4">
                    Failed to load academic periods.
                </div>
            </div>
        );
    }

    return (
        <div className="space-y-6 p-6">
            <StaticTable<PeriodRow>
                columns={columns}
                data={flatData}
                getRowId={(_, index) => index}
                isLoading={isLoading}
                height={480}
                rowHeight={44}
                addHref="/settings/academic-period/add"
                emptyState={
                    <div className="text-muted-foreground py-12 text-center text-sm">
                        No academic periods found.
                    </div>
                }
            />
        </div>
    );
}
