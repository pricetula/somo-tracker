"use client";

import { useState } from "react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { DataTable } from "@/components/shared/data-table";
import { type DataTableColumn } from "@/components/shared/data-table/types";
import { getErrorMessage } from "@/lib/errors";
import { useAcademicYearDetail } from "../hooks/use-academic-years";
import { listTerms } from "@/lib/api/academic-terms";
import { type AcademicTerm } from "@/lib/api/academic-terms";
import { TermForm } from "./term-form";

interface AcademicYearDetailProps {
    id: string;
}

import { TermActionsCell } from "./term-actions-cell";
import { TermStatusCell } from "./term-status-cell";

export function AcademicYearDetail({ id }: AcademicYearDetailProps) {
    const { data: year, isLoading, isError, error } = useAcademicYearDetail(id);

    // UI state
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
