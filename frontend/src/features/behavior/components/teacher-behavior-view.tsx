/**
 * TeacherBehaviorView — shows a teacher's own submitted behavior notes.
 *
 * Displays a list of notes with their review status, an empty state when
 * there are no notes, and a CTA linking to the attendance page where
 * teachers can log new behavior notes.
 */

"use client";

import Link from "next/link";
import { ClipboardList, Plus, AlertTriangle } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { useTeacherNotes } from "../hooks/use-behavior";

export function TeacherBehaviorView() {
    const { data, isLoading, isError } = useTeacherNotes();

    if (isLoading) {
        return (
            <div className="space-y-4">
                {Array.from({ length: 3 }).map((_, i) => (
                    <Skeleton key={i} className="h-28 w-full rounded-lg" />
                ))}
            </div>
        );
    }

    if (isError) {
        return (
            <div className="border-destructive/50 text-destructive rounded-md border p-4">
                Failed to load your behavior notes. Please try again later.
            </div>
        );
    }

    const notes = data?.notes ?? [];

    if (notes.length === 0) {
        return (
            <div className="text-muted-foreground flex flex-col items-center gap-4 py-16">
                <ClipboardList className="h-10 w-10" />
                <div className="text-center">
                    <p className="font-medium">No behavior notes yet</p>
                    <p className="mt-1 max-w-sm text-sm">
                        You haven&apos;t submitted any behavior notes. Log notes while taking
                        attendance in your classes.
                    </p>
                </div>
                <Button asChild>
                    <Link href="/attendance">
                        <Plus className="mr-2 h-4 w-4" />
                        Go to Attendance
                    </Link>
                </Button>
            </div>
        );
    }

    return (
        <div className="space-y-4">
            <div className="flex items-center justify-between">
                <p className="text-muted-foreground text-sm">
                    Notes you have submitted. They appear here once reviewed by an admin.
                </p>
                <Button variant="outline" size="sm" asChild>
                    <Link href="/attendance">
                        <Plus className="mr-2 h-4 w-4" />
                        New Note
                    </Link>
                </Button>
            </div>

            {notes.map((note) => (
                <div
                    key={note.id}
                    className={`rounded-lg border p-4 ${note.is_urgent ? "border-l-4 border-l-red-500" : ""}`}
                >
                    <div className="flex items-start justify-between gap-4">
                        <div className="min-w-0 space-y-1.5">
                            <div className="flex items-center gap-2">
                                <span className="font-medium">{note.student_full_name}</span>
                                <Badge variant="outline" className="text-[10px]">
                                    {note.class_name}
                                </Badge>
                            </div>
                            <p className="text-muted-foreground line-clamp-2 text-sm">
                                {note.description}
                            </p>
                            <div className="flex items-center gap-2">
                                <Badge variant="secondary" className="text-[10px]">
                                    {note.category_name}
                                </Badge>
                                <StatusBadge status={note.status} />
                                {note.is_urgent && (
                                    <Badge variant="destructive" className="gap-1 text-[10px]">
                                        <AlertTriangle className="h-3 w-3" />
                                        Urgent
                                    </Badge>
                                )}
                                <span className="text-muted-foreground text-xs">
                                    {formatDate(note.date)}
                                </span>
                            </div>
                        </div>
                    </div>
                </div>
            ))}
        </div>
    );
}

function StatusBadge({ status }: { status: string }) {
    switch (status) {
        case "PENDING_REVIEW":
            return (
                <Badge variant="outline" className="text-amber-600">
                    Pending Review
                </Badge>
            );
        case "APPROVED":
            return (
                <Badge className="bg-green-100 text-green-700 hover:bg-green-100">Approved</Badge>
            );
        case "REJECTED":
            return <Badge variant="destructive">Rejected</Badge>;
        case "INCLUDED_IN_REPORT":
            return <Badge className="bg-sky-100 text-sky-700 hover:bg-sky-100">In Report</Badge>;
        default:
            return <Badge variant="outline">{status}</Badge>;
    }
}

function formatDate(dateStr: string): string {
    try {
        const date = new Date(dateStr);
        return date.toLocaleDateString(undefined, {
            month: "short",
            day: "numeric",
            year: "numeric",
        });
    } catch {
        return dateStr;
    }
}
