/**
 * AcademicYearsList — lists all academic years with management actions.
 *
 * Uses the shared DataTable component with per-row Set Current, Edit, Delete actions.
 */

"use client";

import Link from "next/link";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { DataTable } from "@/components/shared/data-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";
import { RowActions } from "@/components/shared/data-table/row-actions";
import { listAcademicYears, setCurrentYear, deleteAcademicYear } from "@/lib/api/academic-terms";
import type { AcademicYear } from "@/lib/api/academic-terms";
import { getErrorMessage } from "@/lib/errors";

// ─── Actions cell ─────────────────────────────────────────────────────────

function ActionsCell({ year }: { year: AcademicYear }) {
    const queryClient = useQueryClient();

    const setCurrentMutation = useMutation({
        mutationFn: () => setCurrentYear(year.id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["academic-years"] });
            toast.success(`${year.name} set as current year.`);
        },
        onError: (err) => toast.error(getErrorMessage(err)),
    });

    const deleteMutation = useMutation({
        mutationFn: () => deleteAcademicYear(year.id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["academic-years"] });
            toast.success("Academic year deleted.");
        },
        onError: (err) => toast.error(getErrorMessage(err)),
    });

    return (
        <div className="flex items-center justify-end gap-2">
            {!year.is_current && (
                <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setCurrentMutation.mutate()}
                    disabled={setCurrentMutation.isPending}
                >
                    Set Current
                </Button>
            )}
            <Button variant="outline" size="sm" asChild>
                <Link href={`/academic-years/${year.id}`}>Edit</Link>
            </Button>
            <RowActions
                rowId={year.id}
                label={year.name}
                onDelete={() => deleteMutation.mutate()}
                disabled={deleteMutation.isPending}
            />
        </div>
    );
}

// ─── Columns ──────────────────────────────────────────────────────────────

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

// ─── Component ────────────────────────────────────────────────────────────

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
