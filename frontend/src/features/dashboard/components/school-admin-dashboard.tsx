"use client";

import * as React from "react";

import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { Badge } from "@/components/ui/badge";
import { submitStudentImport, getImportJob } from "@/lib/api/imports";
import { getErrorMessage } from "@/lib/errors";
import type { ImportJobStatus, ImportRow } from "@/lib/api/imports";
import { DataTable } from "@/components/shared/data-table";
import { listStudents, type Student } from "@/lib/api/students";

// ─── Status badge variant mapping ────────────────────────────────────────

const STATUS_VARIANT: Record<ImportJobStatus, "default" | "secondary" | "destructive" | "outline"> =
    {
        pending: "secondary",
        processing: "default",
        completed: "default",
        completed_with_errors: "outline",
        failed: "destructive",
        cancelled: "outline",
    };

// ─── Test data generator ─────────────────────────────────────────────────

const FIRST_NAMES = [
    "Alice",
    "Bob",
    "Charlie",
    "Diana",
    "Edward",
    "Fiona",
    "George",
    "Hannah",
    "Isaac",
    "Julia",
    "Kevin",
    "Laura",
    "Michael",
    "Nina",
    "Oscar",
    "Penelope",
    "Quinn",
    "Rachel",
    "Samuel",
    "Tina",
    "Uma",
    "Victor",
    "Wendy",
    "Xavier",
    "Yvonne",
    "Zachary",
    "Amelia",
    "Benjamin",
    "Catherine",
    "Daniel",
];

const LAST_NAMES = [
    "Smith",
    "Jones",
    "Williams",
    "Brown",
    "Taylor",
    "Davies",
    "Wilson",
    "Evans",
    "Thomas",
    "Roberts",
    "Johnson",
    "Walker",
    "Wright",
    "Thompson",
    "Robinson",
    "White",
    "Hughes",
    "Edwards",
    "Green",
    "Hall",
    "Wood",
    "Harris",
    "Martin",
    "Jackson",
    "Clarke",
    "Lewis",
    "Lee",
    "King",
    "Baker",
    "Scott",
];

function pick<T>(arr: T[]): T {
    return arr[Math.floor(Math.random() * arr.length)];
}

function generateTestStudents(count: number): ImportRow[] {
    return Array.from({ length: count }, (_, i) => {
        const firstName = pick(FIRST_NAMES);
        const lastName = pick(LAST_NAMES);
        return {
            full_name: `${firstName} ${lastName}`,
            gender: i % 2 === 0 ? "M" : "F",
            date_of_birth: `${2014 + (i % 6)}-${String((i % 12) + 1).padStart(2, "0")}-${String((i % 28) + 1).padStart(2, "0")}`,
            admission_number: `TST${String(i + 1).padStart(6, "0")}`,
            upi_number: `UPI${String(i + 1).padStart(14, "0")}`,
            grade_level: "",
            stream_name: "",
        };
    });
}

// ─── Component ───────────────────────────────────────────────────────────

export function SchoolAdminDashboardPage() {
    const [importState, setImportState] = React.useState<{
        jobId: string;
        status: ImportJobStatus;
        totalRecords: number;
        processedRecords: number;
        successCount: number;
        failedCount: number;
        totalChunks: number;
        processedChunks: number;
    } | null>(null);

    const [importing, setImporting] = React.useState(false);
    const [error, setError] = React.useState<string | null>(null);

    const pollRef = React.useRef<ReturnType<typeof setInterval> | null>(null);

    // Cleanup polling on unmount
    React.useEffect(() => {
        return () => {
            if (pollRef.current) clearInterval(pollRef.current);
        };
    }, []);

    const stopPolling = React.useCallback(() => {
        if (pollRef.current) {
            clearInterval(pollRef.current);
            pollRef.current = null;
        }
    }, []);

    const startPolling = React.useCallback(
        (jobId: string) => {
            stopPolling();

            pollRef.current = setInterval(async () => {
                try {
                    const job = await getImportJob(jobId);
                    setImportState({
                        jobId: job.id,
                        status: job.status,
                        totalRecords: job.total_records,
                        processedRecords: job.processed_records,
                        successCount: job.success_count,
                        failedCount: job.failed_count,
                        totalChunks: job.total_chunks,
                        processedChunks: job.processed_chunks,
                    });

                    // Terminal states → stop polling
                    if (
                        job.status === "completed" ||
                        job.status === "completed_with_errors" ||
                        job.status === "failed" ||
                        job.status === "cancelled"
                    ) {
                        stopPolling();
                        setImporting(false);
                    }
                } catch {
                    // Polling errors are non-fatal — stale state is acceptable
                }
            }, 1500);
        },
        [stopPolling]
    );

    const handleGenerateAndImport = async () => {
        setError(null);
        setImporting(true);
        setImportState(null);

        try {
            const rows = generateTestStudents(5000);

            const response = await submitStudentImport({
                academic_term_id: "ef8cd053-ac7b-496b-bd56-af2f2a820c5a",
                rows,
            });

            setImportState({
                jobId: response.job_id,
                status: response.status,
                totalRecords: response.total_records,
                processedRecords: 0,
                successCount: 0,
                failedCount: 0,
                totalChunks: response.total_chunks,
                processedChunks: 0,
            });

            startPolling(response.job_id);
        } catch (err) {
            setImporting(false);
            setError(getErrorMessage(err));
        }
    };

    const isTerminal = importState
        ? ["completed", "completed_with_errors", "failed", "cancelled"].includes(importState.status)
        : false;

    const progress = importState
        ? Math.round((importState.processedRecords / importState.totalRecords) * 100)
        : 0;

    return (
        <article className="space-y-6">
            <div>
                <h1 className="text-2xl font-semibold tracking-tight">School Admin Dashboard</h1>
                <p className="text-muted-foreground mt-1 text-sm">
                    Welcome to SomoTracker. Manage your school, members, and settings.
                </p>
            </div>

            <DataTable
                queryKey={["students"]}
                queryFn={listStudents}
                columns={[
                    {
                        id: "full_name",
                        header: "Name",
                        cell: (row: Student) => (
                            <span className="text-sm font-medium">{row.full_name}</span>
                        ),
                    },
                    {
                        id: "gender",
                        header: "Gender",
                        width: "80px",
                        cell: (row: Student) => (
                            <span className="text-muted-foreground text-xs">{row.gender}</span>
                        ),
                    },
                    {
                        id: "class_name",
                        header: "Class",
                        cell: (row: Student) => (
                            <span className="text-muted-foreground text-xs">
                                {row.class_name ?? "—"}
                            </span>
                        ),
                    },
                    {
                        id: "is_active",
                        header: "Status",
                        width: "100px",
                        cell: (row: Student) => (
                            <span
                                className={
                                    row.is_active
                                        ? "text-xs text-emerald-600"
                                        : "text-muted-foreground text-xs"
                                }
                            >
                                {row.is_active ? "Active" : "Inactive"}
                            </span>
                        ),
                    },
                ]}
                getRowId={(row) => row.id}
                isSearchable
                searchPlaceholder="Search students..."
                rowHeight={48}
                height={300}
                emptyState="No students yet."
                noResultsState="No students match your search."
            />

            {/* ── Bulk Student Import ── */}
            <section className="space-y-4">
                <h2 className="text-lg font-medium">Bulk Student Import</h2>
                <p className="text-muted-foreground text-sm">
                    Generate 5,000 test student records with randomised names and import them in a
                    single batch. Progress is polled from the server every 1.5&thinsp;s.
                </p>

                <Button onClick={handleGenerateAndImport} disabled={importing} size="sm">
                    {importing ? "Importing…" : "Generate 5,000 Test Students & Import"}
                </Button>

                {error && <p className="text-destructive text-sm">{error}</p>}

                {/* ── Progress Panel ── */}
                {importState && (
                    <div className="space-y-3 pt-1">
                        {/* Status badge */}
                        <div className="flex items-center gap-2">
                            <span className="text-sm font-medium">Status:</span>
                            <Badge variant={STATUS_VARIANT[importState.status]}>
                                {importState.status}
                            </Badge>
                        </div>

                        {/* Progress bar */}
                        <div className="space-y-1">
                            <div className="flex justify-between text-sm">
                                <span className="text-muted-foreground">
                                    {importState.processedRecords} / {importState.totalRecords}{" "}
                                    records
                                </span>
                                <span className="text-muted-foreground">{progress}%</span>
                            </div>
                            <Progress value={progress} className="h-2" />
                        </div>

                        {/* Success / failure counts */}
                        <div className="flex gap-4 text-sm">
                            <span className="text-emerald-600">
                                ✓ {importState.successCount} succeeded
                            </span>
                            {importState.failedCount > 0 && (
                                <span className="text-destructive">
                                    ✗ {importState.failedCount} failed
                                </span>
                            )}
                        </div>

                        {/* Chunk progress (during processing) */}
                        {!isTerminal && (
                            <p className="text-muted-foreground animate-pulse text-xs">
                                Processing chunk {importState.processedChunks} of{" "}
                                {importState.totalChunks}
                            </p>
                        )}

                        {isTerminal && (
                            <p className="text-muted-foreground text-xs">Import finished.</p>
                        )}
                    </div>
                )}
            </section>
        </article>
    );
}
