/**
 * ClassRoster — Displays the roster of students enrolled in a class.
 *
 * Uses the shared DataTable component with paginated roster data.
 * Student names are links to the student profile page (intercepted as side sheet).
 */

"use client";

import Link from "next/link";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { UserMinus } from "lucide-react";

import { DataTable } from "@/components/shared/data-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";
import {
    getClassRoster,
    unenrollStudent,
    type RosterEntry,
    type RosterListResult,
} from "@/lib/api/classes";
import { getErrorMessage } from "@/lib/errors";
import { Button } from "@/components/ui/button";
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
    AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { toast } from "sonner";

// ─── Props ─────────────────────────────────────────────────────────────────

interface ClassRosterProps {
    classId: string;
    /** Optional academic term ID; if omitted the backend uses the current term. */
    academicTermId?: string;
}

// ─── Columns ───────────────────────────────────────────────────────────────

function UnenrollCell({ classId, student }: { classId: string; student: RosterEntry }) {
    const queryClient = useQueryClient();

    const unenrollMutation = useMutation({
        mutationFn: () => unenrollStudent(classId, student.id),
        onSuccess: () => {
            toast.success(`${student.full_name} successfully unenrolled.`);
            queryClient.invalidateQueries({ queryKey: ["class-roster", classId] });
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });

    return (
        <AlertDialog>
            <AlertDialogTrigger asChild>
                <Button
                    variant="ghost"
                    size="icon-sm"
                    disabled={unenrollMutation.isPending}
                    className="text-muted-foreground hover:text-destructive"
                >
                    <UserMinus className="h-4 w-4" />
                    <span className="sr-only">Unenroll {student.full_name}</span>
                </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
                <AlertDialogHeader>
                    <AlertDialogTitle>Unenroll Student</AlertDialogTitle>
                    <AlertDialogDescription>
                        Are you sure you want to unenroll <strong>{student.full_name}</strong> from
                        this class? Their enrollment record will be marked as suspended.
                    </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                    <AlertDialogCancel>Cancel</AlertDialogCancel>
                    <AlertDialogAction
                        onClick={(e) => {
                            e.preventDefault();
                            unenrollMutation.mutate();
                        }}
                        className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                    >
                        {unenrollMutation.isPending ? "Unenrolling..." : "Unenroll"}
                    </AlertDialogAction>
                </AlertDialogFooter>
            </AlertDialogContent>
        </AlertDialog>
    );
}

function buildColumns(classId: string): DataTableColumn<RosterEntry>[] {
    return [
        {
            id: "full_name",
            header: "Student Name",
            cell: (row) => (
                <Link
                    href={`/students/${row.id}`}
                    className="hover:text-primary font-medium transition-colors"
                >
                    {row.full_name}
                </Link>
            ),
        },
        {
            id: "admission_number",
            header: "Admission Number",
            width: "160px",
            cell: (row) => (
                <span className="text-muted-foreground">{row.admission_number || "\u2014"}</span>
            ),
        },
        {
            id: "actions",
            header: "",
            width: "48px",
            align: "right",
            cell: (row) => <UnenrollCell classId={classId} student={row} />,
        },
    ];
}

// ─── normalize ─────────────────────────────────────────────────────────────

function normalize(result: RosterListResult) {
    return {
        items: result.items,
        total: result.total,
        page: result.page,
        limit: result.limit,
    };
}

// ─── Wrapper query function ────────────────────────────────────────────────

/**
 * Wraps getClassRoster into the ListApiFn signature expected by DataTable.
 * The classId and academicTermId are baked in via a closure in the component.
 */
function createRosterQueryFn(classId: string, academicTermId?: string) {
    return (params: { page?: number; limit?: number; search?: string }) =>
        getClassRoster(classId, {
            academic_term_id: academicTermId,
            page: params.page,
            limit: params.limit,
            search: params.search,
        });
}

// ─── ClassRoster (DataTable) ───────────────────────────────────────────────

export function ClassRoster({ classId, academicTermId }: ClassRosterProps) {
    const columns = buildColumns(classId);
    const rosterQueryFn = createRosterQueryFn(classId, academicTermId);

    return (
        <DataTable
            addHref={`/classes/${classId}/enroll`}
            queryKey={["class-roster", classId, academicTermId]}
            queryFn={rosterQueryFn}
            columns={columns}
            getRowId={(row) => row.id}
            normalize={normalize}
            isSearchable
            searchPlaceholder="Search by student name or admission number..."
            pageSize={50}
            rowHeight={44}
            height={480}
            emptyState="No students enrolled in this class yet."
            noResultsState="No students match your search."
        />
    );
}
