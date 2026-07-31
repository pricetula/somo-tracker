"use client";

import { XCircle } from "lucide-react";
import { toast } from "sonner";
import { useQueryClient } from "@tanstack/react-query";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { DataTable } from "@/components/shared/data-table";
import { type DataTableColumn } from "@/components/shared/data-table/types";
import { RowActions } from "@/components/shared/data-table/row-actions";
import { type RowAction } from "@/components/shared/data-table/row-actions";
import { listJobs, cancelImportJob, type ImportJob } from "@/lib/api/imports";
import { getErrorMessage } from "@/lib/errors";
import Link from "next/link";

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
const CANCELLABLE_STATUSES = ["pending", "processing"];
function createColumns(
    queryClient: ReturnType<typeof useQueryClient>
): DataTableColumn<ImportJob>[] {
    return [
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
            cell: (row) => (
                <span className="text-emerald-600 tabular-nums">{row.success_count}</span>
            ),
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
            width: "120px",
            align: "right",
            cell: (row) => {
                const cancellable = CANCELLABLE_STATUSES.includes(row.status);
                const rowActions: RowAction[] = cancellable
                    ? [
                          {
                              label: "Cancel",
                              icon: XCircle,
                              destructive: true,
                              confirmTitle: "Cancel Import Job",
                              confirmDescription: `Are you sure you want to cancel this ${row.job_type.replace(/_/g, " ").toLowerCase()} import?`,
                              onClick: async () => {
                                  try {
                                      await cancelImportJob(row.id);
                                      queryClient.invalidateQueries({ queryKey: ["import-jobs"] });
                                      toast.success("Import job cancelled.");
                                  } catch (err) {
                                      toast.error(getErrorMessage(err));
                                  }
                              },
                          },
                      ]
                    : [];

                return (
                    <div className="flex items-center justify-end gap-1">
                        <Button variant="outline" size="sm" asChild>
                            <Link href={`/imports/${row.id}`}>View</Link>
                        </Button>
                        {cancellable && (
                            <RowActions rowId={row.id} label="import job" actions={rowActions} />
                        )}
                    </div>
                );
            },
        },
    ];
}

import { ActiveJobBanner } from "./active-job-banner";

export function ImportJobsList() {
    const queryClient = useQueryClient();
    const cols = createColumns(queryClient);

    return (
        <div className="space-y-4">
            <ActiveJobBanner />

            <DataTable
                isCheckable
                queryKey={["import-jobs", "list"]}
                queryFn={(params) => listJobs(params)}
                columns={cols}
                getRowId={(row) => row.id}
                emptyState="No import jobs yet. Import students or staff to see job history here."
                noResultsState="No import jobs match your search."
                pageSize={20}
            />
        </div>
    );
}
