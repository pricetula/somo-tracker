/**
 * EnrollStudentsPanel — Paginated searchable checklist for batch-enrolling students.
 *
 * Uses DataTable with infinite pagination. Academic year and term are resolved
 * server-side from the current active term.
 */

"use client";

import { useCallback, useMemo, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Loader2, Check } from "lucide-react";

import { DataTable } from "@/components/shared/data-table";
import { type DataTableColumn } from "@/components/shared/data-table/types";
import {
    batchEnrollStudents,
    getAvailableStudents,
    type AvailableStudent,
} from "@/lib/api/classes";
import { getErrorMessage } from "@/lib/errors";
import { Button } from "@/components/ui/button";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { toast } from "sonner";
import { AlertTriangle } from "lucide-react";
import Link from "next/link";

// ─── Props ─────────────────────────────────────────────────────────────────

interface EnrollStudentsPanelProps {
    classId: string;
    /** Called after successful enrollment to close the overlay. */
    onSuccess?: () => void;
}

// ─── Query-fn factory ──────────────────────────────────────────────────────

function createAvailableStudentsQueryFn(classId: string) {
    return (params: { page?: number; limit?: number; search?: string }) =>
        getAvailableStudents(classId, {
            page: params.page,
            limit: params.limit,
            search: params.search,
        });
}

// ─── Component ─────────────────────────────────────────────────────────────

export function EnrollStudentsPanel({ classId, onSuccess }: EnrollStudentsPanelProps) {
    const queryClient = useQueryClient();
    const [errorBanner, setErrorBanner] = useState<string | null>(null);

    const columns = useMemo((): DataTableColumn<AvailableStudent>[] => {
        return [
            {
                id: "full_name",
                header: "Student Name",
                cell: (row) => <span className="truncate font-medium">{row.full_name}</span>,
            },
            {
                id: "admission_number",
                header: "Admission No.",
                width: "160px",
                cell: (row) => (
                    <span className="text-muted-foreground truncate font-mono">
                        {row.admission_number ?? "—"}
                    </span>
                ),
            },
            {
                id: "upi_number",
                header: "UPI Number",
                width: "160px",
                cell: (row) => (
                    <span className="text-muted-foreground truncate font-mono">
                        {row.upi_number ?? "—"}
                    </span>
                ),
            },
            {
                id: "current_class",
                header: "Current Class",
                width: "200px",
                cell: (row) =>
                    row.current_class_id ? (
                        <Link
                            href={`/classes/${row.current_class_id}`}
                            className="text-muted-foreground truncate"
                        >
                            {row.current_class}
                        </Link>
                    ) : (
                        <span className="shrink-0 text-emerald-600">Unenrolled</span>
                    ),
            },
        ];
    }, []);

    const queryFn = useMemo(() => createAvailableStudentsQueryFn(classId), [classId]);

    const isRowCheckable = useCallback((row: AvailableStudent) => !row.current_class_id, []);

    // ── Batch enrollment mutation ────────────────────────────────────
    const enrollMutation = useMutation({
        mutationFn: (studentIds: string[]) => batchEnrollStudents(classId, studentIds),
        onSuccess: (data) => {
            toast.success(data.message ?? `${data.enrolled_count} students enrolled.`);
            queryClient.invalidateQueries({ queryKey: ["class-roster", classId] });
            queryClient.invalidateQueries({ queryKey: ["available-students", classId] });
            onSuccess?.();
        },
        onError: (err) => {
            setErrorBanner(getErrorMessage(err));
        },
    });

    const handleConfirmEnrollment = useCallback(
        (selectedIds: Set<string>) => {
            if (selectedIds.size === 0) return;
            setErrorBanner(null);
            enrollMutation.mutate(Array.from(selectedIds));
        },
        [enrollMutation]
    );

    return (
        <div className="flex flex-col gap-4">
            <p className="text-muted-foreground">
                Search and select students to enroll in this class. The current academic term will
                be used automatically. Students already enrolled in another class are disabled.
            </p>

            {/* Error banner */}
            {errorBanner && (
                <Alert variant="destructive">
                    <AlertTriangle className="h-4 w-4" />
                    <AlertDescription>{errorBanner}</AlertDescription>
                </Alert>
            )}

            {/* Paginated DataTable */}
            <DataTable
                isCheckable
                isRowCheckable={isRowCheckable}
                queryKey={["available-students", classId]}
                queryFn={queryFn}
                columns={columns}
                getRowId={(row) => row.id}
                isSearchable
                searchPlaceholder="Search by name or admission number..."
                pageSize={50}
                height={500}
                emptyState="All students are already enrolled in this class."
                noResultsState="No students match your search."
                renderToolBarComponents={(selectedIds) => (
                    <Button
                        size="sm"
                        onClick={() => handleConfirmEnrollment(selectedIds)}
                        disabled={selectedIds.size === 0 || enrollMutation.isPending}
                    >
                        {enrollMutation.isPending ? (
                            <>
                                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                                Enrolling…
                            </>
                        ) : (
                            <>
                                <Check className="mr-2 h-4 w-4" />
                                Confirm Enrollment ({selectedIds.size})
                            </>
                        )}
                    </Button>
                )}
            />
        </div>
    );
}
