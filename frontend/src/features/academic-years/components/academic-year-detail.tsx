/**
 * AcademicYearDetail — shows a single academic year with its terms.
 *
 * Includes actions: Set Current, Edit Year (inline form toggle),
 * Delete Year, Add Term (dialog), Edit Term (dialog).
 * Uses the shared DataTable component for the terms listing.
 */

"use client";

import { useState } from "react";
import { toast } from "sonner";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
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
import { DataTable } from "@/components/shared/data-table";
import type { DataTableColumn } from "@/components/shared/data-table/types";
import { getErrorMessage } from "@/lib/errors";
import {
    useAcademicYearDetail,
    useSetCurrentYear,
    useDeleteAcademicYear,
} from "../hooks/use-academic-years";
import { listTerms } from "@/lib/api/academic-terms";
import type { AcademicTerm } from "@/lib/api/academic-terms";
import { AcademicYearForm } from "./academic-year-form";
import { TermForm } from "./term-form";

// ─── Actions cell for terms ───────────────────────────────────────────────

function TermActionsCell({
    term,
    onEdit,
}: {
    term: AcademicTerm;
    onEdit: (term: AcademicTerm) => void;
}) {
    return (
        <div className="flex items-center justify-end">
            <Button variant="outline" size="sm" onClick={() => onEdit(term)}>
                Edit
            </Button>
        </div>
    );
}

// ─── Term status cell ─────────────────────────────────────────────────────

function TermStatusCell({ term }: { term: AcademicTerm }) {
    return term.is_current ? (
        <Badge variant="default">Current</Badge>
    ) : (
        <span className="text-muted-foreground">Scheduled</span>
    );
}

// ─── Props ────────────────────────────────────────────────────────────────

interface AcademicYearDetailProps {
    id: string;
}

// ─── Component ────────────────────────────────────────────────────────────

export function AcademicYearDetail({ id }: AcademicYearDetailProps) {
    const { data: year, isLoading, isError, error } = useAcademicYearDetail(id);
    const setCurrentMutation = useSetCurrentYear();
    const deleteMutation = useDeleteAcademicYear();

    // UI state
    const [showEditForm, setShowEditForm] = useState(false);
    const [termDialogOpen, setTermDialogOpen] = useState(false);
    const [editingTerm, setEditingTerm] = useState<AcademicTerm | null>(null);

    // ── Handlers ───────────────────────────────────────────────────────
    const handleEditTerm = (term: AcademicTerm) => {
        setEditingTerm(term);
        setTermDialogOpen(true);
    };

    const handleAddTerm = () => {
        setEditingTerm(null);
        setTermDialogOpen(true);
    };

    const handleTermDialogClose = () => {
        setTermDialogOpen(false);
        setEditingTerm(null);
    };

    const handleDelete = async () => {
        try {
            await deleteMutation.mutateAsync(id);
            toast.success("Academic year deleted.");
        } catch (err) {
            toast.error(getErrorMessage(err));
        }
    };

    // ── Term columns ───────────────────────────────────────────────────
    const termColumns: DataTableColumn<AcademicTerm>[] = [
        {
            id: "term_number",
            header: "#",
            width: "50px",
            cell: (row) => <span className="tabular-nums">{row.term_number}</span>,
        },
        {
            id: "name",
            header: "Name",
            cell: (row) => <span className="font-medium">{row.name}</span>,
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
            id: "status",
            header: "Status",
            width: "100px",
            cell: (row) => <TermStatusCell term={row} />,
        },
        {
            id: "actions",
            header: "",
            width: "80px",
            align: "right",
            cell: (row) => <TermActionsCell term={row} onEdit={handleEditTerm} />,
        },
    ];

    // ── Term query fn ──────────────────────────────────────────────────
    const termQueryFn = () => listTerms({ academic_year_id: id });

    // ── Loading state ─────────────────────────────────────────────────────
    if (isLoading) {
        return (
            <div className="space-y-4">
                <Skeleton className="h-8 w-64" />
                <Skeleton className="h-6 w-48" />
                <Skeleton className="h-10 w-full" />
            </div>
        );
    }

    // ── Error state ──────────────────────────────────────────────────────
    if (isError) {
        return (
            <Alert variant="destructive">
                <AlertDescription>{getErrorMessage(error)}</AlertDescription>
            </Alert>
        );
    }

    if (!year) {
        return (
            <Alert>
                <AlertDescription>Academic year not found.</AlertDescription>
            </Alert>
        );
    }

    // ── Render ────────────────────────────────────────────────────────────

    return (
        <div className="space-y-6">
            {/* ── Year Header ─────────────────────────────────────────── */}
            <div className="space-y-2">
                <div className="flex items-center gap-3">
                    <h1 className="text-foreground text-2xl font-semibold">{year.name}</h1>
                    {year.is_current && <Badge variant="default">Current</Badge>}
                </div>
                <p className="text-muted-foreground">
                    {year.start_date} &mdash; {year.end_date}
                </p>
            </div>

            {/* ── Actions ──────────────────────────────────────────────── */}
            <div className="flex flex-wrap gap-2">
                {!year.is_current && (
                    <Button
                        variant="outline"
                        size="sm"
                        onClick={() => setCurrentMutation.mutate(id)}
                        disabled={setCurrentMutation.isPending}
                    >
                        Set as Current Year
                    </Button>
                )}

                <Button variant="outline" size="sm" onClick={() => setShowEditForm(!showEditForm)}>
                    {showEditForm ? "Cancel" : "Edit Year"}
                </Button>

                <AlertDialog>
                    <AlertDialogTrigger asChild>
                        <Button variant="outline" size="sm" className="text-destructive">
                            Delete Year
                        </Button>
                    </AlertDialogTrigger>
                    <AlertDialogContent>
                        <AlertDialogHeader>
                            <AlertDialogTitle>Delete Academic Year</AlertDialogTitle>
                            <AlertDialogDescription>
                                Are you sure you want to delete &ldquo;{year.name}&rdquo;? This will
                                also delete all terms, assessments, and other linked records within
                                this year. This action cannot be undone.
                            </AlertDialogDescription>
                        </AlertDialogHeader>
                        <AlertDialogFooter>
                            <AlertDialogCancel>Cancel</AlertDialogCancel>
                            <AlertDialogAction onClick={handleDelete}>Delete</AlertDialogAction>
                        </AlertDialogFooter>
                    </AlertDialogContent>
                </AlertDialog>
            </div>

            {/* ── Edit Form (collapsible) ──────────────────────────────── */}
            {showEditForm && (
                <div className="rounded-md border p-4">
                    <h2 className="text-foreground mb-3 font-medium">Edit Academic Year</h2>
                    <AcademicYearForm year={year} />
                </div>
            )}

            {/* ── Terms Section ────────────────────────────────────────── */}
            <div className="space-y-3">
                <div className="flex items-center justify-between">
                    <h2 className="text-foreground text-lg font-medium">Terms</h2>
                    <Button size="sm" onClick={handleAddTerm}>
                        Add Term
                    </Button>
                </div>

                <DataTable
                    queryKey={["academic-terms", id]}
                    queryFn={termQueryFn}
                    columns={termColumns}
                    getRowId={(row) => row.id}
                    emptyState="No terms defined for this academic year."
                    noResultsState="No terms match your search."
                    height={250}
                    pageSize={10}
                />
            </div>

            {/* ── Term Dialog ──────────────────────────────────────────── */}
            <Dialog open={termDialogOpen} onOpenChange={setTermDialogOpen}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>{editingTerm ? "Edit Term" : "Create Term"}</DialogTitle>
                    </DialogHeader>
                    <TermForm
                        academicYearId={id}
                        term={editingTerm ?? undefined}
                        onSuccess={handleTermDialogClose}
                    />
                </DialogContent>
            </Dialog>
        </div>
    );
}
