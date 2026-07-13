/**
 * ClassRoster — Displays the roster of students enrolled in a class.
 *
 * Shows a table with Student Name, Admission Number, and Unenroll action.
 */

"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Loader2, UserMinus } from "lucide-react";

import { getClassRoster, unenrollStudent, type RosterEntry } from "@/lib/api/classes";
import { getErrorMessage } from "@/lib/errors";
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
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

// ─── Hook ──────────────────────────────────────────────────────────────────

export function useClassRoster(classId: string, academicTermId?: string) {
    return useQuery({
        queryKey: ["class-roster", classId, academicTermId],
        queryFn: () => getClassRoster(classId, academicTermId),
        staleTime: 30_000,
    });
}

// ─── Roster Skeleton ───────────────────────────────────────────────────────

export function RosterSkeleton() {
    return (
        <div className="space-y-3">
            <Skeleton className="h-8 w-full" />
            <Skeleton className="h-8 w-full" />
            <Skeleton className="h-8 w-full" />
            <Skeleton className="h-8 w-3/4" />
        </div>
    );
}

// ─── Unenroll Button ───────────────────────────────────────────────────────

function UnenrollButton({
    classId,
    studentId,
    studentName,
}: {
    classId: string;
    studentId: string;
    studentName: string;
}) {
    const queryClient = useQueryClient();

    const unenrollMutation = useMutation({
        mutationFn: () => unenrollStudent(classId, studentId),
        onSuccess: () => {
            toast.success(`${studentName} successfully unenrolled.`);
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
                    {unenrollMutation.isPending ? (
                        <Loader2 className="h-4 w-4 animate-spin" />
                    ) : (
                        <UserMinus className="h-4 w-4" />
                    )}
                    <span className="sr-only">Unenroll {studentName}</span>
                </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
                <AlertDialogHeader>
                    <AlertDialogTitle>Unenroll Student</AlertDialogTitle>
                    <AlertDialogDescription>
                        Are you sure you want to unenroll <strong>{studentName}</strong> from this
                        class? Their enrollment record will be marked as suspended.
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

// ─── Roster Table ──────────────────────────────────────────────────────────

interface RosterTableProps {
    roster: RosterEntry[];
    classId: string;
}

export function RosterTable({ roster, classId }: RosterTableProps) {
    if (roster.length === 0) {
        return (
            <p className="text-muted-foreground py-8 text-center">
                No students enrolled in this class yet.
            </p>
        );
    }

    return (
        <Table>
            <TableHeader>
                <TableRow>
                    <TableHead>Student Name</TableHead>
                    <TableHead>Admission Number</TableHead>
                    <TableHead className="w-12 text-right">Action</TableHead>
                </TableRow>
            </TableHeader>
            <TableBody>
                {roster.map((student) => (
                    <TableRow key={student.id}>
                        <TableCell className="font-medium">{student.full_name}</TableCell>
                        <TableCell className="text-muted-foreground">
                            {student.admission_number || "\u2014"}
                        </TableCell>
                        <TableCell className="text-right">
                            <UnenrollButton
                                classId={classId}
                                studentId={student.id}
                                studentName={student.full_name}
                            />
                        </TableCell>
                    </TableRow>
                ))}
            </TableBody>
        </Table>
    );
}

// ─── ClassRoster (composed) ────────────────────────────────────────────────

export function ClassRoster({ classId, academicTermId }: ClassRosterProps) {
    const { data: roster, isLoading, isError } = useClassRoster(classId, academicTermId);

    if (isLoading) return <RosterSkeleton />;
    if (isError) {
        return (
            <p className="text-destructive py-8 text-center">
                Failed to load roster. Please try again.
            </p>
        );
    }

    return <RosterTable roster={roster ?? []} classId={classId} />;
}
