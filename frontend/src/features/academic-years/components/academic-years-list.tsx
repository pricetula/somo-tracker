"use client";

import { Badge } from "@/components/ui/badge";
import { DataTable } from "@/components/shared/data-table";
import { type DataTableColumn } from "@/components/shared/data-table/types";
import { listAcademicYears, deleteAcademicYear } from "@/lib/api/academic-terms";
import { type AcademicYear } from "@/lib/api/academic-terms";
import Link from "next/link";

const columns: DataTableColumn<AcademicYear>[] = [
    {
        id: "name",
        header: "Name",
        cell: (row) => (
            <Link
                href={`/academic-years/${row.id}`}
                className="hover:text-primary font-medium transition-colors"
            >
                {row.name}
            </Link>
        ),
    },
    {
        id: "start_date",
        header: "Start Date",
        width: "120px",
        cell: (row) => <span className="text-muted-foreground">{row.start_date}</span>,
    },
    {
        id: "end_date",
        header: "End Date",
        width: "120px",
        cell: (row) => <span className="text-muted-foreground">{row.end_date}</span>,
    },
    {
        id: "terms_count",
        header: "Terms",
        width: "80px",
        align: "right",
        cell: (row) => (
            <span className="text-muted-foreground tabular-nums">{row.terms?.length ?? 0}</span>
        ),
    },
    {
        id: "is_current",
        header: "Status",
        width: "100px",
        cell: (row) =>
            row.is_current ? (
                <Badge variant="default">Current</Badge>
            ) : (
                <span className="text-muted-foreground">Inactive</span>
            ),
    },
    {
        id: "actions",
        header: "",
        width: "180px",
        align: "right",
        cell: (row) => <ActionsCell year={row} />,
    },
];

import { ActionsCell } from "./actions-cell";

export function AcademicYearsList() {
    return (
        <DataTable
            isCheckable
            addHref="/academic-years/new"
            queryKey={["academic-years"]}
            queryFn={() => listAcademicYears()}
            columns={columns}
            getRowId={(row) => row.id}
            deleteFn={(id) => deleteAcademicYear(String(id))}
            emptyState="No academic years yet. Create your first academic year to get started."
            noResultsState="No academic years match your search."
        />
    );
}
