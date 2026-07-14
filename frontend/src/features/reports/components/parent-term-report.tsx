/**
 * ParentTermReport — read-only compiled term report view for parents.
 *
 * Shows attendance percentage, approved behavior notes, and competency summary
 * as sections on one page. No destructive or write actions.
 */

"use client";

import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table";

import { useTermReport } from "../hooks/use-reports";

interface ParentTermReportProps {
    termId: string;
    studentId: string;
}

export function ParentTermReport({ termId, studentId }: ParentTermReportProps) {
    const { data, isLoading, isError } = useTermReport(termId, studentId);

    if (isLoading) {
        return (
            <div className="space-y-6">
                <Skeleton className="h-24 w-full rounded-lg" />
                <Skeleton className="h-8 w-48" />
                <Skeleton className="h-32 w-full rounded-lg" />
                <Skeleton className="h-8 w-48" />
                <Skeleton className="h-32 w-full rounded-lg" />
            </div>
        );
    }

    if (isError) {
        return (
            <div className="border-destructive/50 text-destructive rounded-md border p-4">
                Failed to load term report.
            </div>
        );
    }

    if (!data) {
        return (
            <div className="text-muted-foreground flex items-center justify-center py-16">
                <p>No report available for this term.</p>
            </div>
        );
    }

    return (
        <div className="space-y-8">
            <h1 className="text-2xl font-bold">Term Report</h1>

            {/* Attendance Section */}
            <section className="space-y-3">
                <h2 className="text-lg font-semibold">Attendance</h2>
                <div className="rounded-lg border p-6">
                    <div className="flex items-baseline gap-2">
                        <span className="text-4xl font-bold">
                            {data.attendance.attendance_percentage.toFixed(1)}%
                        </span>
                        <span className="text-muted-foreground">attendance</span>
                    </div>
                    <div className="text-muted-foreground mt-3 flex gap-6 text-sm">
                        <span>{data.attendance.absences_count} absences</span>
                        <span>{data.attendance.late_count} late</span>
                    </div>
                </div>
            </section>

            {/* Behavior Notes Section */}
            <section className="space-y-3">
                <h2 className="text-lg font-semibold">Behavior Notes</h2>
                {data.behavior_notes.length === 0 ? (
                    <p className="text-muted-foreground text-sm">
                        No behavior notes for this term.
                    </p>
                ) : (
                    <Table>
                        <TableHeader>
                            <TableRow>
                                <TableHead>Date</TableHead>
                                <TableHead>Category</TableHead>
                                <TableHead>Subject</TableHead>
                                <TableHead>Description</TableHead>
                            </TableRow>
                        </TableHeader>
                        <TableBody>
                            {data.behavior_notes.map((note, idx) => (
                                <TableRow key={idx}>
                                    <TableCell>{note.date}</TableCell>
                                    <TableCell>
                                        <Badge variant="outline">{note.category_name}</Badge>
                                    </TableCell>
                                    <TableCell>{note.subject}</TableCell>
                                    <TableCell className="text-sm">{note.description}</TableCell>
                                </TableRow>
                            ))}
                        </TableBody>
                    </Table>
                )}
            </section>

            {/* Competency Summary Section */}
            <section className="space-y-3">
                <h2 className="text-lg font-semibold">Competency Summary</h2>
                {data.competency_summary.length === 0 ? (
                    <p className="text-muted-foreground text-sm">
                        No competency data for this term.
                    </p>
                ) : (
                    <Table>
                        <TableHeader>
                            <TableRow>
                                <TableHead>Learning Area</TableHead>
                                <TableHead className="w-24">Level</TableHead>
                                <TableHead>Narrative</TableHead>
                            </TableRow>
                        </TableHeader>
                        <TableBody>
                            {data.competency_summary.map((comp, idx) => (
                                <TableRow key={idx}>
                                    <TableCell className="font-medium">
                                        {comp.learning_area_name}
                                    </TableCell>
                                    <TableCell>
                                        <Badge>{comp.final_level}</Badge>
                                    </TableCell>
                                    <TableCell className="text-muted-foreground text-sm">
                                        {comp.teacher_narrative_summary ?? "—"}
                                    </TableCell>
                                </TableRow>
                            ))}
                        </TableBody>
                    </Table>
                )}
            </section>

            {/* Status footer */}
            <p className="text-muted-foreground text-xs">
                Generated {new Date(data.generated_at).toLocaleDateString()}
                {data.published_at
                    ? ` · Published ${new Date(data.published_at).toLocaleDateString()}`
                    : ""}
            </p>
        </div>
    );
}
