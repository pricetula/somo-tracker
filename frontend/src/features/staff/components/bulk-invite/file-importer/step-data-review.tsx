"use client";

/**
 * StepDataReview — review and edit staged invitation records.
 * Matches the student import StepDataReview pattern — paginated, IndexedDB-backed,
 * inline editing, filter tabs.
 */

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
import { Badge } from "@/components/ui/badge";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { cn } from "@/lib/utils";
import type { StagedInviteRecord } from "./types";
import { validateAndDetectDuplicates } from "./validation-utils";
import { getStagedRecordsPaginated, getStagedCountByStatus, updateStagedRecord } from "./db";

const PAGE_SIZE = 25;

interface StepDataReviewProps {
    onProceed: () => void;
    onBack: () => void;
    schoolId: string;
}

// ─── Status Badge ─────────────────────────────────────────────────────────

function StatusBadge({ status }: { status: StagedInviteRecord["status"] }) {
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
    record: StagedInviteRecord;
    onSave: (updated: StagedInviteRecord) => void;
}) {
    const [editing, setEditing] = React.useState(false);
    const [email, setEmail] = React.useState(record.email);
    const [fullName, setFullName] = React.useState(record.full_name);
    const [saving, setSaving] = React.useState(false);

    const handleSave = React.useCallback(() => {
        setSaving(true);
        const updated: StagedInviteRecord = {
            ...record,
            email: email.trim(),
            full_name: fullName.trim(),
            errors: [],
        };
        onSave(updated);
        setEditing(false);
        setSaving(false);
    }, [record, email, fullName, onSave]);

    const handleCancel = React.useCallback(() => {
        setEmail(record.email);
        setFullName(record.full_name);
        setEditing(false);
    }, [record]);

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
                <div className="min-w-0 flex-[2]">
                    <Input
                        value={email}
                        onChange={(e) => setEmail(e.target.value)}
                        onKeyDown={handleKeyDown}
                        className="h-7 text-xs"
                        autoFocus
                    />
                </div>
                <div className="min-w-0 flex-[2]">
                    <Input
                        value={fullName}
                        onChange={(e) => setFullName(e.target.value)}
                        onKeyDown={handleKeyDown}
                        className="h-7 text-xs"
                        placeholder="(optional)"
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
            <div className="min-w-0 flex-[2] truncate">
                {record.email || <span className="text-muted-foreground italic">empty</span>}
            </div>
            <div className="text-muted-foreground min-w-0 flex-[2] truncate">
                {record.full_name || <span className="italic">—</span>}
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

// ─── Error Details Popover ────────────────────────────────────────────────

function ErrorPopover({ errors }: { errors: string[] }) {
    if (errors.length === 0) return null;
    return (
        <div className="flex items-center gap-1">
            <AlertCircle className="text-destructive size-3 shrink-0" />
            <span className="text-destructive truncate text-xs">{errors[0]}</span>
            {errors.length > 1 && (
                <span className="text-muted-foreground text-[10px]">+{errors.length - 1}</span>
            )}
        </div>
    );
}

// ─── Main Component ───────────────────────────────────────────────────────

export function StepDataReview({ onProceed, onBack, schoolId }: StepDataReviewProps) {
    const [records, setRecords] = React.useState<StagedInviteRecord[]>([]);
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
        (updated: StagedInviteRecord) => {
            const recordId = updated.id;
            if (recordId === undefined) return;

            // Debounce: cancel any pending save for this record
            const existing = debounceTimers.current.get(recordId);
            if (existing) clearTimeout(existing);

            const timer = setTimeout(async () => {
                try {
                    const allRecords = await (await import("./db")).getStagedRecords(schoolId);

                    // Re-run validation + dup check against all records
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
                        duplicate(s) must be resolved before inviting.
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
                    <div className="min-w-0 flex-[2]">Email</div>
                    <div className="min-w-0 flex-[2]">Full Name</div>
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
                            <div key={record.id}>
                                <EditableRow record={record} onSave={handleSaveRecord} />
                                {/* Show first error inline */}
                                {record.errors.length > 0 && (
                                    <div className="border-destructive/10 border-t px-2 pb-1">
                                        <ErrorPopover errors={record.errors} />
                                    </div>
                                )}
                            </div>
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
                        : `Send ${counts.valid} Invitation${counts.valid !== 1 ? "s" : ""}`}
                </Button>
            </div>
        </div>
    );
}
