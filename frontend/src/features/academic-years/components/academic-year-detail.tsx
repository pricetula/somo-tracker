/**
 * AcademicYearDetail — shows a single academic year with its terms.
 *
 * Includes actions: Set Current, Edit Year (inline form toggle),
 * Delete Year, Add Term (dialog), Edit Term (dialog).
 */

"use client";

import { useState } from "react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogTrigger,
} from "@/components/ui/dialog";
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
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table";
import { getErrorMessage } from "@/lib/errors";
import {
    useAcademicYearDetail,
    useSetCurrentYear,
    useDeleteAcademicYear,
} from "../hooks/use-academic-years";
import { AcademicYearForm } from "./academic-year-form";
import { TermForm } from "./term-form";
import type { AcademicTerm } from "../types";

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

    const terms: AcademicTerm[] = (year.terms ?? []) as AcademicTerm[];

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
                        onClick={() => setCurrentMutation.mutate(year.id)}
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
                            <AlertDialogAction onClick={() => deleteMutation.mutate(year.id)}>
                                Delete
                            </AlertDialogAction>
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
                    <Dialog
                        open={termDialogOpen && !editingTerm}
                        onOpenChange={(open) => {
                            setTermDialogOpen(open);
                            if (!open) setEditingTerm(null);
                        }}
                    >
                        <DialogTrigger asChild>
                            <Button
                                size="sm"
                                onClick={() => {
                                    setEditingTerm(null);
                                    setTermDialogOpen(true);
                                }}
                            >
                                Add Term
                            </Button>
                        </DialogTrigger>
                        <DialogContent>
                            <DialogHeader>
                                <DialogTitle>Create Term</DialogTitle>
                            </DialogHeader>
                            <TermForm
                                academicYearId={year.id}
                                onSuccess={() => setTermDialogOpen(false)}
                            />
                        </DialogContent>
                    </Dialog>
                </div>

                {terms.length === 0 ? (
                    <p className="text-muted-foreground">
                        No terms defined for this academic year.
                    </p>
                ) : (
                    <Table>
                        <TableHeader>
                            <TableRow>
                                <TableHead>#</TableHead>
                                <TableHead>Name</TableHead>
                                <TableHead>Start Date</TableHead>
                                <TableHead>End Date</TableHead>
                                <TableHead>Status</TableHead>
                                <TableHead className="text-right">Actions</TableHead>
                            </TableRow>
                        </TableHeader>
                        <TableBody>
                            {terms.map((term) => (
                                <TableRow key={term.id}>
                                    <TableCell>{term.term_number}</TableCell>
                                    <TableCell className="font-medium">{term.name}</TableCell>
                                    <TableCell>{term.start_date}</TableCell>
                                    <TableCell>{term.end_date}</TableCell>
                                    <TableCell>
                                        {term.is_current ? (
                                            <Badge variant="default">Current</Badge>
                                        ) : (
                                            <span className="text-muted-foreground">Scheduled</span>
                                        )}
                                    </TableCell>
                                    <TableCell className="text-right">
                                        <Dialog
                                            open={termDialogOpen && editingTerm?.id === term.id}
                                            onOpenChange={(open) => {
                                                if (!open) {
                                                    setTermDialogOpen(false);
                                                    setEditingTerm(null);
                                                }
                                            }}
                                        >
                                            <DialogTrigger asChild>
                                                <Button
                                                    variant="outline"
                                                    size="sm"
                                                    onClick={() => {
                                                        setEditingTerm(term);
                                                        setTermDialogOpen(true);
                                                    }}
                                                >
                                                    Edit
                                                </Button>
                                            </DialogTrigger>
                                            <DialogContent>
                                                <DialogHeader>
                                                    <DialogTitle>Edit Term</DialogTitle>
                                                </DialogHeader>
                                                <TermForm
                                                    academicYearId={year.id}
                                                    term={term}
                                                    onSuccess={() => {
                                                        setTermDialogOpen(false);
                                                        setEditingTerm(null);
                                                    }}
                                                />
                                            </DialogContent>
                                        </Dialog>
                                    </TableCell>
                                </TableRow>
                            ))}
                        </TableBody>
                    </Table>
                )}
            </div>
        </div>
    );
}
