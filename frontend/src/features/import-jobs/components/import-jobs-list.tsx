/**
 * ImportJobsList — list of import jobs with active job banner.
 */

"use client";

import { useState } from "react";
import Link from "next/link";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table";
import { getErrorMessage } from "@/lib/errors";
import { useImportJobs, useActiveImportJob } from "../hooks/use-import-jobs";
import type { ImportJobStatus } from "../types";

// ─── Status Badge helper ──────────────────────────────────────────────────

function statusBadge(status: ImportJobStatus) {
    const variants: Record<ImportJobStatus, "default" | "secondary" | "destructive" | "outline"> = {
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

// ─── Component ────────────────────────────────────────────────────────────

export function ImportJobsList() {
    const [page, setPage] = useState(1);
    const limit = 20;

    const { data, isLoading, isError, error } = useImportJobs({ page, limit });
    const { data: activeData } = useActiveImportJob();

    const activeJob = activeData?.active ? activeData.job : null;

    // ── Loading ──────────────────────────────────────────────────────────
    if (isLoading) {
        return (
            <div className="space-y-4">
                <Skeleton className="h-8 w-48" />
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-10 w-full" />
            </div>
        );
    }

    // ── Error ────────────────────────────────────────────────────────────
    if (isError) {
        return (
            <Alert variant="destructive">
                <AlertDescription>{getErrorMessage(error)}</AlertDescription>
            </Alert>
        );
    }

    const jobs = data?.data ?? [];
    const total = data?.total ?? 0;
    const totalPages = Math.max(1, Math.ceil(total / limit));

    return (
        <div className="space-y-6">
            {/* ── Active Job Banner ──────────────────────────────────────── */}
            {activeJob && (
                <div className="bg-muted/30 rounded-md px-4 py-3">
                    <div className="flex items-center justify-between">
                        <div className="space-y-1">
                            <p className="text-foreground text-sm font-medium">Active Import Job</p>
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
            )}

            {/* ── Empty State ────────────────────────────────────────────── */}
            {jobs.length === 0 && !activeJob && (
                <p className="text-muted-foreground">
                    No import jobs yet. Import students or staff to see job history here.
                </p>
            )}

            {/* ── Jobs Table ─────────────────────────────────────────────── */}
            {jobs.length > 0 && (
                <Table>
                    <TableHeader>
                        <TableRow>
                            <TableHead>Type</TableHead>
                            <TableHead>Status</TableHead>
                            <TableHead>Records</TableHead>
                            <TableHead>Success</TableHead>
                            <TableHead>Failed</TableHead>
                            <TableHead>Started</TableHead>
                            <TableHead className="text-right">Actions</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {jobs.map((job) => (
                            <TableRow key={job.id}>
                                <TableCell className="font-medium">
                                    {job.job_type.replace(/_/g, " ")}
                                </TableCell>
                                <TableCell>{statusBadge(job.status)}</TableCell>
                                <TableCell>{job.total_records}</TableCell>
                                <TableCell className="text-emerald-600">
                                    {job.success_count}
                                </TableCell>
                                <TableCell className="text-destructive">
                                    {job.failed_count > 0 ? job.failed_count : "—"}
                                </TableCell>
                                <TableCell className="text-muted-foreground text-sm">
                                    {job.created_at
                                        ? new Date(job.created_at).toLocaleDateString()
                                        : "—"}
                                </TableCell>
                                <TableCell className="text-right">
                                    <Button variant="outline" size="sm" asChild>
                                        <Link href={`/imports/${job.id}`}>View</Link>
                                    </Button>
                                </TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
            )}

            {/* ── Pagination ─────────────────────────────────────────────── */}
            {totalPages > 1 && (
                <div className="text-muted-foreground flex items-center justify-between text-sm">
                    <p>
                        Page {page} of {totalPages} ({total} total)
                    </p>
                    <div className="flex gap-2">
                        <Button
                            variant="outline"
                            size="sm"
                            disabled={page <= 1}
                            onClick={() => setPage((p) => Math.max(1, p - 1))}
                        >
                            Previous
                        </Button>
                        <Button
                            variant="outline"
                            size="sm"
                            disabled={page >= totalPages}
                            onClick={() => setPage((p) => p + 1)}
                        >
                            Next
                        </Button>
                    </div>
                </div>
            )}
        </div>
    );
}
