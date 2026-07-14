/**
 * AdminTermReportManager — admin view for generating and managing term reports.
 *
 * Shows a list of students with their report status (DRAFT/PUBLISHED/none),
 * and allows generating or publishing reports.
 */

"use client";

import { Loader2, FileText } from "lucide-react";
import { Skeleton } from "@/components/ui/skeleton";
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";

import {
    useTermReportList,
    useGenerateClassReports,
    usePublishTermReport,
} from "../hooks/use-reports";

interface AdminTermReportManagerProps {
    termId: string;
    classId?: string;
}

export function AdminTermReportManager({ termId, classId }: AdminTermReportManagerProps) {
    const { data, isLoading, isError } = useTermReportList(termId, classId);
    const generateClass = useGenerateClassReports();
    const publishReport = usePublishTermReport();

    if (isLoading) {
        return (
            <div className="space-y-3">
                <Skeleton className="h-8 w-64" />
                <Skeleton className="h-8 w-full" />
                <Skeleton className="h-8 w-full" />
            </div>
        );
    }

    if (isError) {
        return (
            <div className="border-destructive/50 text-destructive rounded-md border p-4">
                Failed to load term reports.
            </div>
        );
    }

    const reports = data?.items ?? [];

    const handleGenerateClass = () => {
        if (!classId) return;
        generateClass.mutate({ termId, classId });
    };

    const handlePublish = (reportId: string) => {
        publishReport.mutate(reportId);
    };

    return (
        <div className="space-y-4">
            <div className="flex items-center justify-between">
                <h2 className="text-xl font-semibold">Term Reports</h2>
                {classId && (
                    <Button
                        variant="outline"
                        onClick={handleGenerateClass}
                        disabled={generateClass.isPending}
                    >
                        {generateClass.isPending ? (
                            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                        ) : (
                            <FileText className="mr-2 h-4 w-4" />
                        )}
                        Generate All
                    </Button>
                )}
            </div>

            {reports.length === 0 ? (
                <div className="text-muted-foreground flex items-center justify-center py-16">
                    <p>
                        No reports generated yet. Click &ldquo;Generate All&rdquo; to create them.
                    </p>
                </div>
            ) : (
                <Table>
                    <TableHeader>
                        <TableRow>
                            <TableHead>Student</TableHead>
                            <TableHead className="w-28">Status</TableHead>
                            <TableHead className="w-28">Generated</TableHead>
                            <TableHead className="w-24" />
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {reports.map((report) => (
                            <TableRow key={report.id}>
                                <TableCell className="font-medium">{report.student_id}</TableCell>
                                <TableCell>
                                    <Badge
                                        variant={
                                            report.status === "PUBLISHED" ? "default" : "secondary"
                                        }
                                    >
                                        {report.status}
                                    </Badge>
                                </TableCell>
                                <TableCell className="text-muted-foreground text-sm">
                                    {new Date(report.generated_at).toLocaleDateString()}
                                </TableCell>
                                <TableCell>
                                    {report.status === "DRAFT" && (
                                        <Button
                                            size="sm"
                                            variant="outline"
                                            onClick={() => handlePublish(report.id)}
                                            disabled={publishReport.isPending}
                                        >
                                            Publish
                                        </Button>
                                    )}
                                </TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
            )}
        </div>
    );
}
