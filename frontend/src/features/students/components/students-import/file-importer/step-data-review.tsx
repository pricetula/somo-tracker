"use client";

import { AlertCircle, ChevronLeft, ChevronRight } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { type StagedStudentRecord } from "./types";
import { validateAndDetectDuplicates } from "./utils/validation-utils";
import {
    getStagedRecords,
    getStagedRecordsPaginated,
    getStagedCountByStatus,
    updateStagedRecord,
} from "./db";
import { checkDuplicates } from "@/lib/api/imports";
import * as React from "react";

const PAGE_SIZE = 25;
interface StepDataReviewProps {
    onProceed: () => void;
    onBack: () => void;
    schoolId: string;
}

import { EditableRow } from "./editable-row";

export function StepDataReview({ onProceed, onBack, schoolId }: StepDataReviewProps) {
    const [records, setRecords] = React.useState<StagedStudentRecord[]>([]);
    const [total, setTotal] = React.useState(0);
    const [page, setPage] = React.useState(1);
    const [filter, setFilter] = React.useState<"all" | "valid" | "error">("all");
    const [counts, setCounts] = React.useState({
        total: 0,
        valid: 0,
        error: 0,
        duplicate: 0,
        submitted: 0,
    });
    // Debounced saves
    const debounceTimers = React.useRef<Map<number, ReturnType<typeof setTimeout>>>(new Map());

    // ── Data loading ───────────────────────────────────────────────────
    React.useEffect(() => {
        Promise.all([
            getStagedRecordsPaginated(schoolId, page, PAGE_SIZE, filter),
            getStagedCountByStatus(schoolId),
        ]).then(([paginated, statusCounts]) => {
            setRecords(paginated.records);
            setTotal(paginated.total);
            setCounts(statusCounts);
        });
    }, [page, filter, schoolId]);

    const handleSaveRecord = React.useCallback(
        (updated: StagedStudentRecord) => {
            const recordId = updated.id;
            if (recordId === undefined) return;

            // Debounce: cancel any pending save for this record
            const existing = debounceTimers.current.get(recordId);
            if (existing) clearTimeout(existing);

            const timer = setTimeout(async () => {
                try {
                    // Re-validate the record
                    const allRecords = await (await import("./db")).getStagedRecords(schoolId);

                    // Need to re-run dup check against all records
                    const otherRecords = allRecords.filter((r) => r.id !== recordId);
                    const validated = validateAndDetectDuplicates([updated, ...otherRecords]);
                    const finalRecord = validated.find((r) => r.id === recordId) ?? updated;

                    await updateStagedRecord(finalRecord);
                    debounceTimers.current.delete(recordId);

                    // Reload data after edit
                    const [paginated, statusCounts] = await Promise.all([
                        getStagedRecordsPaginated(schoolId, page, PAGE_SIZE, filter),
                        getStagedCountByStatus(schoolId),
                    ]);
                    setRecords(paginated.records);
                    setTotal(paginated.total);
                    setCounts(statusCounts);
                } catch (err) {
                    console.error("Failed to save record:", err);
                }
            }, 300);

            debounceTimers.current.set(recordId, timer);

            // Optimistically update local state
            setRecords((prev) => prev.map((r) => (r.id === recordId ? updated : r)));
        },
        [page, filter, schoolId]
    );

    // Cleanup debounce timers on unmount
    React.useEffect(() => {
        const timers = debounceTimers.current;
        return () => {
            for (const timer of timers.values()) {
                clearTimeout(timer);
            }
            timers.clear();
        };
    }, []);

    // ── Against-existing-records check (once on entering review step) ──
    const existingCheckDone = React.useRef(false);
    React.useEffect(() => {
        if (existingCheckDone.current) return;

        (async () => {
            try {
                const allRecords = await getStagedRecords(schoolId);
                const admNumbers = allRecords
                    .map((r) => r.payload.admission_number)
                    .filter(Boolean) as string[];
                const upiNumbers = allRecords
                    .map((r) => r.payload.upi_number)
                    .filter(Boolean) as string[];
                const knecNumbers = allRecords
                    .map((r) => r.payload.knec_assessment_number)
                    .filter(Boolean) as string[];

                if (
                    admNumbers.length === 0 &&
                    upiNumbers.length === 0 &&
                    knecNumbers.length === 0
                ) {
                    existingCheckDone.current = true;
                    return;
                }

                const result = await checkDuplicates({
                    admission_numbers: admNumbers,
                    upi_numbers: upiNumbers,
                    knec_assessment_numbers: knecNumbers,
                });

                const existingAdmSet = new Set(
                    result.existing_admission_numbers.map((v) => v.toLowerCase())
                );
                const existingUPISet = new Set(
                    result.existing_upi_numbers.map((v) => v.toLowerCase())
                );
                const existingKnecSet = new Set(
                    result.existing_knec_assessment_numbers.map((v) => v.toLowerCase())
                );

                for (const record of allRecords) {
                    let hasConflict = false;
                    const newErrors: string[] = [];

                    if (
                        record.payload.admission_number &&
                        existingAdmSet.has(record.payload.admission_number.toLowerCase())
                    ) {
                        newErrors.push(
                            `Admission number "${record.payload.admission_number}" already exists for this school`
                        );
                        hasConflict = true;
                    }
                    if (
                        record.payload.upi_number &&
                        existingUPISet.has(record.payload.upi_number.toLowerCase())
                    ) {
                        newErrors.push(
                            `UPI number "${record.payload.upi_number}" already exists for this school`
                        );
                        hasConflict = true;
                    }
                    if (
                        record.payload.knec_assessment_number &&
                        existingKnecSet.has(record.payload.knec_assessment_number.toLowerCase())
                    ) {
                        newErrors.push(
                            `KNEC assessment number "${record.payload.knec_assessment_number}" already exists for this school`
                        );
                        hasConflict = true;
                    }

                    if (hasConflict && record.id !== undefined) {
                        const updated = {
                            ...record,
                            status: "duplicate" as const,
                            errors: [...record.errors, ...newErrors],
                        };
                        await updateStagedRecord(updated);
                    }
                }

                existingCheckDone.current = true;
            } catch (err) {
                console.error("Failed to check existing duplicates:", err);
                existingCheckDone.current = true;
            } finally {
                // Reload to reflect any changes
                const [paginated, statusCounts] = await Promise.all([
                    getStagedRecordsPaginated(schoolId, page, PAGE_SIZE, filter),
                    getStagedCountByStatus(schoolId),
                ]);
                setRecords(paginated.records);
                setTotal(paginated.total);
                setCounts(statusCounts);
            }
        })();
    }, [page, filter, schoolId]);

    const hasBlockingErrors = React.useMemo(
        () => counts.error > 0 || counts.duplicate > 0,
        [counts]
    );

    const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

    const handleFilterChange = React.useCallback((newFilter: string) => {
        setFilter(newFilter as "all" | "valid" | "error");
        setPage(1);
    }, []);

    // ─── Render ────────────────────────────────────────────────────────

    return (
        <div className="space-y-4">
            <div>
                <h3 className="font-medium">Review Staged Records</h3>
                <p className="text-muted-foreground mt-1 text-xs">
                    {counts.total} records loaded —{" "}
                    <span className="text-emerald-600">{counts.valid} valid</span>
                    {counts.error > 0 && (
                        <span className="text-destructive"> · {counts.error} with errors</span>
                    )}
                    {counts.duplicate > 0 && (
                        <span className="text-amber-600"> · {counts.duplicate} duplicates</span>
                    )}
                    . Click the edit icon to fix values inline.
                </p>
            </div>

            {/* Blocking errors alert */}
            {hasBlockingErrors && (
                <Alert variant="destructive" className="py-2">
                    <AlertCircle className="size-4" />
                    <AlertTitle>Records need attention</AlertTitle>
                    <AlertDescription>
                        {counts.error} record(s) with validation errors and {counts.duplicate}{" "}
                        duplicate(s) must be resolved before importing.
                    </AlertDescription>
                </Alert>
            )}

            {/* Filter tabs */}
            <div className="flex items-center gap-1">
                {(["all", "valid", "error"] as const).map((f) => (
                    <Button
                        key={f}
                        variant={filter === f ? "secondary" : "ghost"}
                        size="sm"
                        className="h-7 text-xs"
                        onClick={() => handleFilterChange(f)}
                    >
                        {f === "all" && `All (${counts.total})`}
                        {f === "valid" && `Valid (${counts.valid})`}
                        {f === "error" && `Errors (${counts.error + counts.duplicate})`}
                    </Button>
                ))}
            </div>

            {/* Records list */}
            <div className="rounded-md border">
                {/* Fixed header */}
                <div className="bg-muted/50 text-muted-foreground flex items-center gap-2 border-b px-2 py-1.5 text-[10px] font-medium tracking-wider uppercase">
                    <div className="min-w-0 flex-1">Full Name</div>
                    <div className="w-20 shrink-0">Gender</div>
                    <div className="w-24 shrink-0">UPI</div>
                    <div className="w-24 shrink-0">KNEC</div>
                    <div className="w-32 shrink-0">DOB</div>
                    <div className="w-16 shrink-0">Status</div>
                    <div className="w-16 shrink-0">Edit</div>
                </div>
                {/* Scrollable rows */}
                <div className="max-h-80 overflow-y-auto">
                    {records.length === 0 && total === 0 ? (
                        <div className="text-muted-foreground p-4 text-center">
                            Loading records...
                        </div>
                    ) : records.length === 0 ? (
                        <div className="text-muted-foreground p-4 text-center">
                            No records found.
                        </div>
                    ) : (
                        records.map((record) => (
                            <EditableRow
                                key={record.id}
                                record={record}
                                onSave={handleSaveRecord}
                            />
                        ))
                    )}
                </div>
            </div>

            {/* Pagination */}
            {totalPages > 1 && (
                <div className="flex items-center justify-between">
                    <p className="text-muted-foreground text-xs">
                        Page {page} of {totalPages} ({total} total)
                    </p>
                    <div className="flex items-center gap-1">
                        <Button
                            variant="ghost"
                            size="icon"
                            className="size-7"
                            disabled={page <= 1}
                            onClick={() => setPage((p) => Math.max(1, p - 1))}
                        >
                            <ChevronLeft className="size-4" />
                        </Button>
                        <Button
                            variant="ghost"
                            size="icon"
                            className="size-7"
                            disabled={page >= totalPages}
                            onClick={() => setPage((p) => p + 1)}
                        >
                            <ChevronRight className="size-4" />
                        </Button>
                    </div>
                </div>
            )}

            {/* Actions */}
            <div className="flex items-center justify-between pt-2">
                <Button variant="ghost" size="sm" onClick={onBack}>
                    Back
                </Button>
                <Button
                    size="sm"
                    onClick={onProceed}
                    disabled={hasBlockingErrors || counts.total === 0}
                >
                    {hasBlockingErrors
                        ? "Fix Errors to Continue"
                        : `Import ${counts.valid} Records`}
                </Button>
            </div>
        </div>
    );
}
