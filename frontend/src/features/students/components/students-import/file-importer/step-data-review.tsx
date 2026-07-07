"use client";

import * as React from "react";
import {
    AlertCircle,
    CheckCircle2,
    ChevronLeft,
    ChevronRight,
    Edit,
    XCircle,
    AlertTriangle,
    Save,
    X,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { cn } from "@/lib/utils";
import type { StagedStudentRecord } from "./types";
import { validateAndDetectDuplicates } from "./utils/validation-utils";
import { getStagedRecordsPaginated, getStagedCountByStatus, updateStagedRecord } from "./db";

const PAGE_SIZE = 25;

interface StepDataReviewProps {
    onProceed: () => void;
    onBack: () => void;
}

// ─── Status Badge ─────────────────────────────────────────────────────────

function StatusBadge({ status }: { status: StagedStudentRecord["status"] }) {
    switch (status) {
        case "valid":
            return (
                <Badge variant="default" className="bg-emerald-500/10 text-[10px] text-emerald-600">
                    <CheckCircle2 className="mr-0.5 size-3" />
                    Valid
                </Badge>
            );
        case "error":
            return (
                <Badge
                    variant="outline"
                    className="text-destructive border-destructive/30 text-[10px]"
                >
                    <XCircle className="mr-0.5 size-3" />
                    Error
                </Badge>
            );
        case "duplicate":
            return (
                <Badge variant="outline" className="border-amber-200 text-[10px] text-amber-600">
                    <AlertTriangle className="mr-0.5 size-3" />
                    Duplicate
                </Badge>
            );
        case "submitted":
            return (
                <Badge variant="outline" className="border-blue-200 text-[10px] text-blue-600">
                    Submitted
                </Badge>
            );
    }
}

// ─── Editable Row ─────────────────────────────────────────────────────────

function EditableRow({
    record,
    onSave,
}: {
    record: StagedStudentRecord;
    onSave: (updated: StagedStudentRecord) => void;
}) {
    const [editing, setEditing] = React.useState(false);
    const [fullName, setFullName] = React.useState(record.payload.full_name);
    const [gender, setGender] = React.useState(record.payload.gender ?? "none");
    const [upi, setUpi] = React.useState(record.payload.upi_number ?? "");
    const [knec, setKnec] = React.useState(record.payload.knec_assessment_number ?? "");
    const [dob, setDob] = React.useState(record.payload.date_of_birth ?? "");
    const [saving, setSaving] = React.useState(false);

    const handleSave = React.useCallback(() => {
        setSaving(true);
        const updated: StagedStudentRecord = {
            ...record,
            payload: {
                ...record.payload,
                full_name: fullName.trim(),
                gender: gender === "none" ? undefined : gender,
                upi_number: upi || undefined,
                knec_assessment_number: knec || undefined,
                date_of_birth: dob || undefined,
            },
            // Clear previous errors — re-validation happens in parent
            errors: [],
        };
        onSave(updated);
        setEditing(false);
        setSaving(false);
    }, [record, fullName, gender, upi, knec, dob, onSave]);

    const handleCancel = React.useCallback(() => {
        setFullName(record.payload.full_name);
        setGender(record.payload.gender ?? "none");
        setUpi(record.payload.upi_number ?? "");
        setKnec(record.payload.knec_assessment_number ?? "");
        setDob(record.payload.date_of_birth ?? "");
        setEditing(false);
    }, [record]);

    // Enter key saves, Escape cancels
    const handleKeyDown = React.useCallback(
        (e: React.KeyboardEvent) => {
            if (e.key === "Enter" && !e.shiftKey) {
                e.preventDefault();
                handleSave();
            } else if (e.key === "Escape") {
                handleCancel();
            }
        },
        [handleSave, handleCancel]
    );

    const rowBase = cn(
        "flex items-center gap-2 px-2 py-1.5 text-sm",
        record.status !== "valid" && !editing && "bg-destructive/5"
    );

    if (editing) {
        return (
            <div className={cn(rowBase, "bg-muted/30")}>
                <div className="min-w-0 flex-1">
                    <Input
                        value={fullName}
                        onChange={(e) => setFullName(e.target.value)}
                        onKeyDown={handleKeyDown}
                        className="h-7 text-xs"
                        autoFocus
                    />
                </div>
                <div className="w-20 shrink-0">
                    <Select value={gender} onValueChange={setGender}>
                        <SelectTrigger className="h-7 text-xs">
                            <SelectValue placeholder="-" />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectItem value="none">-</SelectItem>
                            <SelectItem value="M">Male</SelectItem>
                            <SelectItem value="F">Female</SelectItem>
                        </SelectContent>
                    </Select>
                </div>
                <div className="w-24 shrink-0">
                    <Input
                        value={upi}
                        onChange={(e) => setUpi(e.target.value)}
                        onKeyDown={handleKeyDown}
                        className="h-7 text-xs"
                        placeholder="-"
                    />
                </div>
                <div className="w-24 shrink-0">
                    <Input
                        value={knec}
                        onChange={(e) => setKnec(e.target.value)}
                        onKeyDown={handleKeyDown}
                        className="h-7 text-xs"
                        placeholder="-"
                    />
                </div>
                <div className="w-32 shrink-0">
                    <Input
                        value={dob}
                        onChange={(e) => setDob(e.target.value)}
                        onKeyDown={handleKeyDown}
                        className="h-7 text-xs"
                        placeholder="-"
                        type="date"
                    />
                </div>
                <div className="w-16 shrink-0">
                    <StatusBadge status={record.status} />
                </div>
                <div className="flex w-16 shrink-0 items-center gap-1">
                    <Button
                        variant="ghost"
                        size="icon"
                        className="size-6"
                        onClick={handleSave}
                        disabled={saving}
                    >
                        <Save className="size-3 text-emerald-500" />
                    </Button>
                    <Button variant="ghost" size="icon" className="size-6" onClick={handleCancel}>
                        <X className="text-muted-foreground size-3" />
                    </Button>
                </div>
            </div>
        );
    }

    return (
        <div className={cn(rowBase, "hover:bg-muted/30 transition-colors")}>
            <div className="min-w-0 flex-1 truncate">{record.payload.full_name}</div>
            <div className="text-muted-foreground w-20 shrink-0">
                {record.payload.gender ?? "-"}
            </div>
            <div className="text-muted-foreground w-24 shrink-0 truncate">
                {record.payload.upi_number ?? "-"}
            </div>
            <div className="text-muted-foreground w-24 shrink-0 truncate">
                {record.payload.knec_assessment_number ?? "-"}
            </div>
            <div className="text-muted-foreground w-32 shrink-0">
                {record.payload.date_of_birth ?? "-"}
            </div>
            <div className="w-16 shrink-0">
                <StatusBadge status={record.status} />
            </div>
            <div className="flex w-16 shrink-0 items-center">
                <Button
                    variant="ghost"
                    size="icon"
                    className="size-6"
                    onClick={() => setEditing(true)}
                >
                    <Edit className="text-muted-foreground size-3" />
                </Button>
            </div>
        </div>
    );
}

// ─── Main Component ───────────────────────────────────────────────────────

export function StepDataReview({ onProceed, onBack }: StepDataReviewProps) {
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
            getStagedRecordsPaginated(page, PAGE_SIZE, filter),
            getStagedCountByStatus(),
        ]).then(([paginated, statusCounts]) => {
            setRecords(paginated.records);
            setTotal(paginated.total);
            setCounts(statusCounts);
        });
    }, [page, filter]);

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
                    const allRecords = await (await import("./db")).getStagedRecords();

                    // Need to re-run dup check against all records
                    const otherRecords = allRecords.filter((r) => r.id !== recordId);
                    const validated = validateAndDetectDuplicates([updated, ...otherRecords]);
                    const finalRecord = validated.find((r) => r.id === recordId) ?? updated;

                    await updateStagedRecord(finalRecord);
                    debounceTimers.current.delete(recordId);

                    // Reload data after edit
                    const [paginated, statusCounts] = await Promise.all([
                        getStagedRecordsPaginated(page, PAGE_SIZE, filter),
                        getStagedCountByStatus(),
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
        [page, filter]
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
                <h3 className="text-sm font-medium">Review Staged Records</h3>
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
                        <div className="text-muted-foreground p-4 text-center text-sm">
                            Loading records...
                        </div>
                    ) : records.length === 0 ? (
                        <div className="text-muted-foreground p-4 text-center text-sm">
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
