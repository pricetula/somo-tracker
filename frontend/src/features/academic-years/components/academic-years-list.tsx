/**
 * AcademicYearsList — lists all academic years with management actions.
 *
 * Renders a table of years with name, date range, current badge,
 * and actions (Set Current, Edit, Delete).
 */

"use client";

import Link from "next/link";

import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
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
import { getErrorMessage } from "@/lib/errors";
import {
    useAcademicYearsManage,
    useSetCurrentYear,
    useDeleteAcademicYear,
} from "../hooks/use-academic-years";

export function AcademicYearsList() {
    const { data, isLoading, isError, error } = useAcademicYearsManage();
    const setCurrentMutation = useSetCurrentYear();
    const deleteMutation = useDeleteAcademicYear();

    // ── Loading state ─────────────────────────────────────────────────────
    if (isLoading) {
        return (
            <div className="space-y-3">
                <Skeleton className="h-8 w-48" />
                <Skeleton className="h-10 w-full" />
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

    const years = data?.items ?? [];

    if (years.length === 0) {
        return (
            <div className="space-y-4">
                <p className="text-muted-foreground">
                    No academic years yet. Create your first academic year to get started.
                </p>
                <Button asChild>
                    <Link href="/academic-years/new">Add Academic Year</Link>
                </Button>
            </div>
        );
    }

    return (
        <div className="space-y-4">
            <div className="flex items-center justify-between">
                <p className="text-muted-foreground">
                    {years.length} year{years.length !== 1 ? "s" : ""}
                </p>
                <Button asChild>
                    <Link href="/academic-years/new">Add Academic Year</Link>
                </Button>
            </div>

            <Table>
                <TableHeader>
                    <TableRow>
                        <TableHead>Name</TableHead>
                        <TableHead>Start Date</TableHead>
                        <TableHead>End Date</TableHead>
                        <TableHead>Terms</TableHead>
                        <TableHead>Status</TableHead>
                        <TableHead className="text-right">Actions</TableHead>
                    </TableRow>
                </TableHeader>
                <TableBody>
                    {years.map((year) => (
                        <TableRow key={year.id}>
                            <TableCell className="font-medium">
                                <Link
                                    href={`/academic-years/${year.id}`}
                                    className="hover:text-primary transition-colors"
                                >
                                    {year.name}
                                </Link>
                            </TableCell>
                            <TableCell>{year.start_date}</TableCell>
                            <TableCell>{year.end_date}</TableCell>
                            <TableCell>{year.terms?.length ?? 0}</TableCell>
                            <TableCell>
                                {year.is_current ? (
                                    <Badge variant="default">Current</Badge>
                                ) : (
                                    <span className="text-muted-foreground">Inactive</span>
                                )}
                            </TableCell>
                            <TableCell className="text-right">
                                <div className="flex items-center justify-end gap-2">
                                    {!year.is_current && (
                                        <Button
                                            variant="outline"
                                            size="sm"
                                            onClick={() => setCurrentMutation.mutate(year.id)}
                                            disabled={setCurrentMutation.isPending}
                                        >
                                            Set Current
                                        </Button>
                                    )}
                                    <Button variant="outline" size="sm" asChild>
                                        <Link href={`/academic-years/${year.id}`}>Edit</Link>
                                    </Button>
                                    <AlertDialog>
                                        <AlertDialogTrigger asChild>
                                            <Button
                                                variant="outline"
                                                size="sm"
                                                className="text-destructive"
                                                disabled={deleteMutation.isPending}
                                            >
                                                Delete
                                            </Button>
                                        </AlertDialogTrigger>
                                        <AlertDialogContent>
                                            <AlertDialogHeader>
                                                <AlertDialogTitle>
                                                    Delete Academic Year
                                                </AlertDialogTitle>
                                                <AlertDialogDescription>
                                                    Are you sure you want to delete &ldquo;
                                                    {year.name}&rdquo;? This will also delete all
                                                    terms, assessments, and other linked records
                                                    within this year. This action cannot be undone.
                                                </AlertDialogDescription>
                                            </AlertDialogHeader>
                                            <AlertDialogFooter>
                                                <AlertDialogCancel>Cancel</AlertDialogCancel>
                                                <AlertDialogAction
                                                    onClick={() => deleteMutation.mutate(year.id)}
                                                >
                                                    Delete
                                                </AlertDialogAction>
                                            </AlertDialogFooter>
                                        </AlertDialogContent>
                                    </AlertDialog>
                                </div>
                            </TableCell>
                        </TableRow>
                    ))}
                </TableBody>
            </Table>
        </div>
    );
}
