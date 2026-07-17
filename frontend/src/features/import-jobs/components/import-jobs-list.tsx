/**
 * ImportJobsList — list of import jobs using the shared DataTable component.
 *
 * Uses DataTable with built-in pagination and search.
 */

"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { DataTable } from "@/components/shared/data-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";
import { listJobs, getActiveImportJob, type ImportJob } from "@/lib/api/imports";

// ─── Status Badge helper ──────────────────────────────────────────────────

function statusBadge(status: string) {
    const variants: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
        pending: "secondary",
        processing: "default",
        completed: "default",
        completed_with_errors: "destructive",
        failed: "destructive",
        cancelling: "secondary",
        cancelled: "outline",
    };
    return <Badge variant={variants[status] ?? "outline"}>{status.replace(/_/g, " ")}</Badge>;
}

// ─── Columns ──────────────────────────────────────────────────────────────

const columns: DataTableColumn<ImportJob>[] = [
    {
        id: "job_type",
        header: "Type",
        cell: (row) => <span className="font-medium">{row.job_type.replace(/_/g, " ")}</span>,
    },
    {
        id: "status",
        header: "Status",
        width: "130px",
        cell: (row) => statusBadge(row.status),
    },
    {
        id: "total_records",
        header: "Records",
        width: "80px",
        align: "right",
        cell: (row) => (
            <span className="text-muted-foreground tabular-nums">{row.total_records}</span>
        ),
    },
    {
        id: "success_count",
        header: "Success",
        width: "80px",
        align: "right",
        cell: (row) => <span className="text-emerald-600 tabular-nums">{row.success_count}</span>,
    },
    {
        id: "failed_count",
        header: "Failed",
        width: "80px",
        align: "right",
        cell: (row) =>
            row.failed_count > 0 ? (
                <span className="text-destructive tabular-nums">{row.failed_count}</span>
            ) : (
                <span className="text-muted-foreground">—</span>
            ),
    },
    {
        id: "created_at",
        header: "Started",
        width: "110px",
        cell: (row) => (
            <span className="text-muted-foreground text-xs">
                {row.created_at ? new Date(row.created_at).toLocaleDateString() : "—"}
            </span>
        ),
    },
    {
        id: "actions",
        header: "",
        width: "80px",
        align: "right",
        cell: (row) => (
            <Button variant="outline" size="sm" asChild>
                <Link href={`/imports/${row.id}`}>View</Link>
            </Button>
        ),
    },
];

// ─── Active Job Banner ────────────────────────────────────────────────────

function ActiveJobBanner() {
    const { data: activeData } = useQuery({
        queryKey: ["import-jobs", "active"],
        queryFn: () => getActiveImportJob(),
        staleTime: 15 * 1000,
        refetchInterval: (query) => {
            const d = query.state.data;
            if (!d?.active || !d.job) return false;
            const terminalStatuses = ["completed", "completed_with_errors", "failed", "cancelled"];
            if (terminalStatuses.includes(d.job.status)) return false;
            return 5_000;
        },
    });

    const activeJob = activeData?.active ? activeData.job : null;
    if (!activeJob) return null;

    return (
        <div className="bg-muted/30 mb-4 rounded-md px-4 py-3">
            <div className="flex items-center justify-between">
                <div className="space-y-1">
                    <p className="text-foreground font-medium">Active Import Job</p>
                    <p className="text-muted-foreground text-xs">
                        {activeJob.job_type.replace(/_/g, " ")} &mdash;{" "}
                        {activeJob.success_count + activeJob.failed_count} of{" "}
                        {activeJob.total_records} processed
                    </p>
                </div>
                <Button variant="outline" size="sm" asChild>
                    <Link href={`/imports/${activeJob.id}`}>View Details</Link>
                </Button>
            </div>
        </div>
    );
}

// ─── Component ────────────────────────────────────────────────────────────

export function ImportJobsList() {
    return (
        <div className="space-y-4">
            <ActiveJobBanner />

            <DataTable
                queryKey={["import-jobs", "list"]}
                queryFn={(params) => listJobs(params)}
                columns={columns}
                getRowId={(row) => row.id}
                emptyState="No import jobs yet. Import students or staff to see job history here."
                noResultsState="No import jobs match your search."
                pageSize={20}
            />
        </div>
    );
}
