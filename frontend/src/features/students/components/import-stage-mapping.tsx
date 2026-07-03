/**
 * Stage 1: Upload, Column Mapping & Academic Term Selection.
 *
 * - Parses file headers from row 1
 * - Auto-matches columns using header dictionary + fuzzy matching
 * - Multi-select for full_name (split-name files)
 * - Academic term selection (auto-select if single active term)
 * - Persists to IndexedDB on every change (debounced ~300ms)
 * - On resume, loads column_mapping and academic_term_id as initial state
 */

"use client";

import * as React from "react";
import { Button } from "@/components/ui/button";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import {
    Command,
    CommandEmpty,
    CommandGroup,
    CommandInput,
    CommandItem,
} from "@/components/ui/command";
import { Badge } from "@/components/ui/badge";
import { Label } from "@/components/ui/label";
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Check, ChevronsUpDown, Loader2, Upload, X } from "lucide-react";
import { cn } from "@/lib/utils";
import { getErrorMessage } from "@/lib/errors";
import { useImportStore } from "../hooks/use-import-store";
import { listTerms, type AcademicTerm } from "@/lib/api/academic-terms";
import { listClasses } from "@/lib/api/classes";
import { matchAllHeaders, type HeaderField } from "@/lib/import-data/matching";
import { setImportMeta, clearAll, getImportMeta } from "@/lib/import-data/db";
import type { ColumnMapping, ImportStage } from "@/lib/import-data/types";
import * as XLSX from "xlsx";

// ─── Props ─────────────────────────────────────────────────────────────────

interface ImportStageMappingProps {
    onStageChange: (stage: ImportStage) => void;
    onClose: () => void;
}

// ─── Types ─────────────────────────────────────────────────────────────────

interface ParsedHeader {
    original: string;
    normalized: string;
    mappedField: HeaderField | null;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function ImportStageMapping({ onStageChange, onClose }: ImportStageMappingProps) {
    const store = useImportStore();

    const [parsedHeaders, setParsedHeaders] = React.useState<ParsedHeader[]>([]);
    const [rawRows, setRawRows] = React.useState<Record<string, unknown>[]>([]);
    const [fileName, setFileName] = React.useState<string>("");

    // Column mapping state
    const [columnMapping, setColumnMappingLocal] = React.useState<ColumnMapping>({
        full_name: [],
        gender: null,
        date_of_birth: null,
        class_room: null,
        nemis_number: null,
        assessment_number: null,
        birth_certificate_number: null,
    });

    // Academic term state
    const [academicYearId, setAcademicYearIdLocal] = React.useState("");
    const [academicTermId, setAcademicTermIdLocal] = React.useState("");
    const [terms, setTerms] = React.useState<AcademicTerm[]>([]);
    const [termsLoading, setTermsLoading] = React.useState(false);

    // UI state
    const [parsing, setParsing] = React.useState(false);
    const [error, setError] = React.useState<string | null>(null);
    const [showStaleDialog, setShowStaleDialog] = React.useState(false);
    const [showClearConfirm, setShowClearConfirm] = React.useState(false);
    const [initialized, setInitialized] = React.useState(false);

    // Debounce ref for persisting mapping
    const debounceRef = React.useRef<ReturnType<typeof setTimeout> | undefined>(undefined);

    // ─── Initialize / Resume ──────────────────────────────────────────────
    React.useEffect(() => {
        const resume = async () => {
            const schoolId = getSchoolIdFromAuth(); // We'll read it from meta if already stored
            if (!schoolId) {
                setInitialized(true);
                return;
            }

            await store.initialize(schoolId);

            if (store.meta) {
                // Check stale (7+ days) — the store sets isStale
                if (store.isStale) {
                    setShowStaleDialog(true);
                    return;
                }

                // If the meta exists and has a stage beyond MAPPING, skip
                if (
                    store.meta.current_stage === "PREVIEW" ||
                    store.meta.current_stage === "READY" ||
                    store.meta.current_stage === "SUBMITTING"
                ) {
                    // Redirect to the right stage
                    onStageChange(store.meta.current_stage);
                    return;
                }

                // Resume MAPPING state
                if (store.meta.column_mapping) {
                    setColumnMappingLocal(store.meta.column_mapping);
                }
                if (store.meta.academic_term_id) {
                    setAcademicYearIdLocal(store.meta.academic_year_id);
                    setAcademicTermIdLocal(store.meta.academic_term_id);
                }
                if (store.meta.total_rows > 0) {
                    // Already have rows — skip to PREVIEW
                    onStageChange("PREVIEW");
                    return;
                }
            }

            setInitialized(true);
        };

        resume();
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    // ─── Fetch terms when year changes ────────────────────────────────────
    React.useEffect(() => {
        if (!academicYearId) return;

        const fetchTerms = async () => {
            setTermsLoading(true);
            try {
                const res = await listTerms({ academic_year_id: academicYearId });
                const allTerms = res.data ?? [];

                // Filter to active/current terms (is_current or within date range)
                const now = new Date();
                const eligible = allTerms.filter((t) => {
                    if (t.is_current) return true;
                    const start = new Date(t.start_date);
                    const end = new Date(t.end_date);
                    return start <= now && end >= now;
                });

                setTerms(allTerms);

                // Auto-select if exactly one eligible term
                if (eligible.length === 1) {
                    setAcademicTermIdLocal(eligible[0].id);
                    store.setAcademicTerm(eligible[0].id);
                }
            } catch {
                setTerms([]);
            } finally {
                setTermsLoading(false);
            }
        };

        fetchTerms();
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [academicYearId]);

    // ─── Debounced persist ────────────────────────────────────────────────
    const persistMapping = React.useCallback(
        (mapping: ColumnMapping) => {
            if (debounceRef.current) clearTimeout(debounceRef.current);
            debounceRef.current = setTimeout(() => {
                store.setColumnMapping(mapping);
            }, 300);
        },
        [store]
    );

    // ─── File upload handler ──────────────────────────────────────────────
    const handleFileUpload = React.useCallback(
        async (e: React.ChangeEvent<HTMLInputElement>) => {
            const file = e.target.files?.[0];
            if (!file) return;

            setParsing(true);
            setError(null);
            setFileName(file.name);

            try {
                const buffer = await file.arrayBuffer();
                const workbook = XLSX.read(buffer, { type: "array" });
                const sheet = workbook.Sheets[workbook.SheetNames[0]];
                const jsonData = XLSX.utils.sheet_to_json<Record<string, unknown>>(sheet, {
                    defval: "",
                });

                if (jsonData.length === 0) {
                    setError("File appears to be empty. Please check your file and try again.");
                    setParsing(false);
                    return;
                }

                // Extract headers from the first row's keys
                const headerKeys = Object.keys(jsonData[0]);
                const matched = matchAllHeaders(headerKeys);

                const headers: ParsedHeader[] = headerKeys.map((h) => ({
                    original: h,
                    normalized: h.toLowerCase().trim(),
                    mappedField: matched[h.toLowerCase().trim()] ?? null,
                }));

                setParsedHeaders(headers);
                setRawRows(jsonData);

                // Auto-populate mapping from matches
                const mapping: ColumnMapping = {
                    full_name: [],
                    gender: null,
                    date_of_birth: null,
                    class_room: null,
                    nemis_number: null,
                    assessment_number: null,
                    birth_certificate_number: null,
                };

                for (const h of headers) {
                    if (h.mappedField === "full_name") {
                        mapping.full_name.push(h.original);
                    } else if (h.mappedField === "gender") {
                        mapping.gender = h.original;
                    } else if (h.mappedField === "class_room") {
                        mapping.class_room = h.original;
                    } else if (h.mappedField === "date_of_birth") {
                        mapping.date_of_birth = h.original;
                    } else if (h.mappedField === "nemis_number") {
                        mapping.nemis_number = h.original;
                    } else if (h.mappedField === "assessment_number") {
                        mapping.assessment_number = h.original;
                    } else if (h.mappedField === "birth_certificate_number") {
                        mapping.birth_certificate_number = h.original;
                    }
                }

                setColumnMappingLocal(mapping);
                store.setColumnMapping(mapping);

                // Save initial meta to IndexedDB
                const schoolId = getSchoolIdFromAuth();
                if (schoolId) {
                    await setImportMeta({
                        school_id: schoolId,
                        current_stage: "MAPPING",
                        column_mapping: mapping,
                        academic_year_id: academicYearId,
                        academic_term_id: academicTermId,
                        total_rows: jsonData.length,
                        schema_version: 2,
                        created_at: new Date().toISOString(),
                        classes_last_fetched_at: null,
                        idempotency_key: null,
                        import_job_id: null,
                        file_name: file.name,
                    });
                }
            } catch (err) {
                setError(getErrorMessage(err));
            } finally {
                setParsing(false);
            }
        },
        [academicYearId, academicTermId, store]
    );

    // ─── Mapping update handlers ──────────────────────────────────────────
    const updateSingleMapping = React.useCallback(
        (field: keyof ColumnMapping, value: string | null | string[]) => {
            setColumnMappingLocal((prev) => {
                // TypeScript-safe field assignment
                const next: ColumnMapping = {
                    ...prev,
                    [field]: value as string[] & string & null,
                };
                persistMapping(next);
                return next;
            });
        },
        [persistMapping]
    );

    const toggleFullNameColumn = React.useCallback(
        (header: string) => {
            setColumnMappingLocal((prev) => {
                const current = prev.full_name;
                const next = current.includes(header)
                    ? current.filter((h) => h !== header)
                    : [...current, header];
                const updated = { ...prev, full_name: next };
                persistMapping(updated);
                return updated;
            });
        },
        [persistMapping]
    );

    // ─── Continue validation ──────────────────────────────────────────────
    const canContinue =
        columnMapping.full_name.length >= 1 &&
        columnMapping.gender !== null &&
        academicTermId !== "";

    const getGateReason = (): string | null => {
        if (columnMapping.full_name.length === 0) return "Map at least one column to Full Name";
        if (columnMapping.gender === null) return "Map a column to Gender";
        if (!academicTermId) return "Select an academic term";
        return null;
    };

    // ─── Continue handler ─────────────────────────────────────────────────
    const handleContinue = async () => {
        if (!canContinue) return;

        const schoolId = getSchoolIdFromAuth();
        if (!schoolId) return;

        // Fetch classes for the selected year/term
        try {
            const classRes = await listClasses({
                academic_year_id: academicYearId,
                academic_term_id: academicTermId,
                limit: 500,
            });

            // Store classes_last_fetched_at
            await store.setAcademicYear(academicYearId);
            await store.setAcademicTerm(academicTermId);

            // Store classes in the meta (we'll use them during processing)
            const meta = await getImportMeta(schoolId);
            if (meta) {
                await setImportMeta({
                    ...meta,
                    classes_last_fetched_at: new Date().toISOString(),
                });
            }

            // Build class lookup and store for the worker
            const classEntries: Array<{
                id: string;
                grade_level: string;
                stream_name: string;
                display_label: string;
            }> = (classRes.data ?? []).map((c) => ({
                id: c.id,
                grade_level: c.grade_level,
                stream_name: c.stream_name,
                display_label: c.display_label,
            }));

            // Launch worker to process rows
            processRowsInWorker(rawRows, columnMapping, classEntries, store, () =>
                onStageChange("PREVIEW")
            );
        } catch (err) {
            setError(getErrorMessage(err));
        }
    };

    // ─── Clear import ─────────────────────────────────────────────────────
    const handleClear = async () => {
        const schoolId = getSchoolIdFromAuth();
        if (schoolId) {
            await store.clearImport(schoolId);
        }
        setParsedHeaders([]);
        setRawRows([]);
        setColumnMappingLocal({
            full_name: [],
            gender: null,
            date_of_birth: null,
            class_room: null,
            nemis_number: null,
            assessment_number: null,
            birth_certificate_number: null,
        });
        setFileName("");
        setShowClearConfirm(false);
    };

    // ─── Stale dialog handlers ────────────────────────────────────────────
    const handleResumeStale = async () => {
        setShowStaleDialog(false);
        if (store.meta) {
            onStageChange(store.meta.current_stage);
        }
    };

    const handleDiscardStale = async () => {
        const schoolId = getSchoolIdFromAuth();
        if (schoolId) {
            await clearAll(schoolId);
        }
        setShowStaleDialog(false);
        setInitialized(true);
    };

    // ─── Render ───────────────────────────────────────────────────────────
    if (!initialized) {
        return (
            <div className="flex items-center justify-center py-16">
                <Loader2 className="text-muted-foreground size-6 animate-spin" />
            </div>
        );
    }

    return (
        <div className="flex flex-1 flex-col gap-6 overflow-y-auto px-1 py-4">
            {/* Stale import dialog */}
            <AlertDialog open={showStaleDialog} onOpenChange={setShowStaleDialog}>
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>Resume previous import?</AlertDialogTitle>
                        <AlertDialogDescription>
                            This import was started over 7 days ago. You can resume where you left
                            off or discard it and start fresh.
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel onClick={handleDiscardStale}>Discard</AlertDialogCancel>
                        <AlertDialogAction onClick={handleResumeStale}>Resume</AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>

            {/* Clear confirmation */}
            <AlertDialog open={showClearConfirm} onOpenChange={setShowClearConfirm}>
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>Clear current import?</AlertDialogTitle>
                        <AlertDialogDescription>
                            This will discard all mapping choices and parsed rows. This action
                            cannot be undone.
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel>Cancel</AlertDialogCancel>
                        <AlertDialogAction onClick={handleClear}>Clear Import</AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>

            {/* Error display */}
            {error && (
                <div className="text-destructive bg-destructive/10 rounded-md px-3 py-2 text-sm">
                    {error}
                    <Button
                        variant="ghost"
                        size="sm"
                        className="ml-2"
                        onClick={() => setError(null)}
                    >
                        <X className="size-3" />
                    </Button>
                </div>
            )}

            {/* File upload (only show if no file loaded) */}
            {parsedHeaders.length === 0 && (
                <div className="flex flex-col items-center gap-4 py-8">
                    <div className="flex flex-col items-center gap-2">
                        <Upload className="text-muted-foreground size-8" />
                        <p className="text-foreground text-sm font-medium">
                            Upload a CSV or Excel file
                        </p>
                        <p className="text-muted-foreground text-xs">
                            .csv, .xlsx, or .xls files up to 10MB
                        </p>
                    </div>
                    <label className="cursor-pointer">
                        <div className="bg-primary text-primary-foreground hover:bg-primary/90 rounded-md px-4 py-2 text-sm font-medium transition-colors">
                            {parsing ? (
                                <>
                                    <Loader2 className="mr-1.5 inline size-4 animate-spin" />
                                    Parsing…
                                </>
                            ) : (
                                "Choose File"
                            )}
                        </div>
                        <input
                            type="file"
                            accept=".csv,.xlsx,.xls"
                            className="sr-only"
                            onChange={handleFileUpload}
                            disabled={parsing}
                        />
                    </label>
                </div>
            )}

            {/* Column mapping section */}
            {parsedHeaders.length > 0 && (
                <>
                    <div className="flex items-center justify-between">
                        <div>
                            <p className="text-foreground text-sm font-medium">{fileName}</p>
                            <p className="text-muted-foreground text-xs">
                                {rawRows.length} rows · {parsedHeaders.length} columns
                            </p>
                        </div>
                        <div className="flex items-center gap-2">
                            <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => setShowClearConfirm(true)}
                            >
                                Clear
                            </Button>
                        </div>
                    </div>

                    {/* Academic Year & Term */}
                    <div className="space-y-4">
                        <div className="space-y-1.5">
                            <Label>Academic Term</Label>
                            <Select
                                value={academicTermId}
                                onValueChange={(v) => {
                                    setAcademicTermIdLocal(v);
                                    store.setAcademicTerm(v);
                                }}
                            >
                                <SelectTrigger className="w-72">
                                    <SelectValue
                                        placeholder={
                                            termsLoading
                                                ? "Loading terms…"
                                                : "Select an academic term"
                                        }
                                    />
                                </SelectTrigger>
                                <SelectContent>
                                    {terms.length === 0 ? (
                                        <SelectItem value="__no_terms__" disabled>
                                            No terms available
                                        </SelectItem>
                                    ) : (
                                        terms.map((t) => (
                                            <SelectItem key={t.id} value={t.id}>
                                                {t.name} {t.is_current ? "(Current)" : ""}
                                            </SelectItem>
                                        ))
                                    )}
                                </SelectContent>
                            </Select>
                        </div>

                        {/* Column mappings */}
                        <div className="space-y-3">
                            <p className="text-foreground text-sm font-medium">Column Mapping</p>
                            <p className="text-muted-foreground text-xs">
                                Map each CSV column to the correct import field.
                            </p>

                            {/* Full Name (multi-select) */}
                            <div className="space-y-1.5">
                                <Label>
                                    Full Name
                                    <span className="text-destructive ml-1">*</span>
                                </Label>
                                <Popover>
                                    <PopoverTrigger asChild>
                                        <Button
                                            variant="outline"
                                            role="combobox"
                                            className="w-72 justify-between"
                                        >
                                            {columnMapping.full_name.length > 0
                                                ? `${columnMapping.full_name.length} column${
                                                      columnMapping.full_name.length > 1 ? "s" : ""
                                                  } selected`
                                                : "Select columns…"}
                                            <ChevronsUpDown className="ml-2 size-4 shrink-0 opacity-50" />
                                        </Button>
                                    </PopoverTrigger>
                                    <PopoverContent className="w-72 p-0">
                                        <Command>
                                            <CommandInput placeholder="Search columns…" />
                                            <CommandEmpty>No column found.</CommandEmpty>
                                            <CommandGroup>
                                                {parsedHeaders.map((h) => (
                                                    <CommandItem
                                                        key={h.original}
                                                        value={h.original}
                                                        onSelect={() =>
                                                            toggleFullNameColumn(h.original)
                                                        }
                                                    >
                                                        <Check
                                                            className={cn(
                                                                "mr-2 size-4",
                                                                columnMapping.full_name.includes(
                                                                    h.original
                                                                )
                                                                    ? "opacity-100"
                                                                    : "opacity-0"
                                                            )}
                                                        />
                                                        {h.original}
                                                    </CommandItem>
                                                ))}
                                            </CommandGroup>
                                        </Command>
                                    </PopoverContent>
                                </Popover>
                                {columnMapping.full_name.length > 0 && (
                                    <div className="flex flex-wrap gap-1 pt-1">
                                        {columnMapping.full_name.map((col) => (
                                            <Badge
                                                key={col}
                                                variant="secondary"
                                                className="text-xs"
                                            >
                                                {col}
                                                <button
                                                    className="ml-1"
                                                    onClick={() => toggleFullNameColumn(col)}
                                                >
                                                    <X className="size-3" />
                                                </button>
                                            </Badge>
                                        ))}
                                    </div>
                                )}
                            </div>

                            {/* Single-select fields */}
                            {(
                                [
                                    ["gender" as const, "Gender"],
                                    ["class_room" as const, "Class Room"],
                                    ["date_of_birth" as const, "Date of Birth"],
                                    ["nemis_number" as const, "NEMIS Number"],
                                    ["assessment_number" as const, "Assessment Number"],
                                    ["birth_certificate_number" as const, "Birth Certificate"],
                                ] as [
                                    (
                                        | "gender"
                                        | "class_room"
                                        | "date_of_birth"
                                        | "nemis_number"
                                        | "assessment_number"
                                        | "birth_certificate_number"
                                    ),
                                    string,
                                ][]
                            ).map(([field, label]) => (
                                <div key={field} className="space-y-1.5">
                                    <Label>
                                        {label}
                                        {field === "gender" && (
                                            <span className="text-destructive ml-1">*</span>
                                        )}
                                    </Label>
                                    <Select
                                        value={columnMapping[field] ?? ""}
                                        onValueChange={(v) => updateSingleMapping(field, v || null)}
                                    >
                                        <SelectTrigger className="w-72">
                                            <SelectValue placeholder="Unmapped" />
                                        </SelectTrigger>
                                        <SelectContent>
                                            {parsedHeaders.map((h) => (
                                                <SelectItem key={h.original} value={h.original}>
                                                    {h.original}
                                                </SelectItem>
                                            ))}
                                        </SelectContent>
                                    </Select>
                                </div>
                            ))}
                        </div>
                    </div>
                </>
            )}

            {/* Actions bar */}
            {parsedHeaders.length > 0 && (
                <div className="flex items-center justify-between border-t pt-4">
                    <div>
                        {!canContinue && (
                            <p className="text-muted-foreground text-xs">{getGateReason()}</p>
                        )}
                    </div>
                    <div className="flex items-center gap-2">
                        <Button variant="ghost" onClick={onClose}>
                            Cancel
                        </Button>
                        <Button onClick={handleContinue} disabled={!canContinue || parsing}>
                            {parsing ? (
                                <>
                                    <Loader2 className="mr-1.5 size-4 animate-spin" />
                                    Processing…
                                </>
                            ) : (
                                "Continue"
                            )}
                        </Button>
                    </div>
                </div>
            )}
        </div>
    );
}

// ─── Auth helper ──────────────────────────────────────────────────────────

/**
 * Read the active school_id from the auth context/locals.
 * This is a simplified version — in production it would read from
 * a useAuth() hook or similar.
 */
function getSchoolIdFromAuth(): string | null {
    if (typeof window === "undefined") return null;
    // Attempt to read from a meta tag or session store
    const meta = document.querySelector('meta[name="school-id"]');
    if (meta) return meta.getAttribute("content");

    // Fallback: try to read from localStorage (set by auth context)
    try {
        const stored = localStorage.getItem("somo-active-school-id");
        return stored ?? null;
    } catch {
        return null;
    }
}

// ─── Worker processing ───────────────────────────────────────────────────

function processRowsInWorker(
    rawRows: Record<string, unknown>[],
    columnMapping: ColumnMapping,
    classEntries: Array<{
        id: string;
        grade_level: string;
        stream_name: string;
        display_label: string;
    }>,
    store: ReturnType<typeof useImportStore>,
    onDone: () => void
) {
    // Build class lookup map for the worker
    const classLookup: Map<
        string,
        {
            class_id: string;
            grade_level: string;
            stream_name: string;
            display_label: string;
        }
    > = new Map();

    for (const entry of classEntries) {
        classLookup.set(entry.id, {
            class_id: entry.id,
            grade_level: entry.grade_level,
            stream_name: entry.stream_name,
            display_label: entry.display_label,
        });
    }

    // Try to use Web Worker
    let worker: Worker | null = null;

    try {
        worker = new Worker(new URL("../../../workers/student-import.worker.ts", import.meta.url), {
            type: "module",
        });
    } catch {
        // Worker creation failed — fall back to main-thread processing
        processOnMainThread(rawRows, columnMapping, classLookup, store, onDone);
        return;
    }

    worker.onmessage = (e) => {
        const data = e.data;

        if (data.type === "progress") {
            // Store progress is updated as chunks arrive
        } else if (data.type === "chunk") {
            // Map worker output to StagedRow format
            const stagedRows = data.rows.map((r: Record<string, unknown>) => ({
                row_number: r.row_number as number,
                raw_data: rawRows[r.row_number as number] ?? {},
                processed_data: {
                    full_name: r.full_name as string,
                    gender: r.gender as "M" | "F" | "",
                    date_of_birth: (r.date_of_birth as string) ?? null,
                    class_id: (r.class_id as string) ?? null,
                    grade_level: (r.grade_level as string) ?? "",
                    stream_name: (r.stream_name as string) ?? "",
                    nemis_number: (r.nemis_number as string) ?? null,
                    assessment_number: (r.assessment_number as string) ?? null,
                    birth_certificate_number: (r.birth_certificate_number as string) ?? null,
                },
                ui_meta: {
                    has_error: r.has_error as boolean,
                    skipped: false,
                    submitted: false,
                    errors: {
                        missing_required: (r.errors as Record<string, unknown>).missing_required as
                            | string
                            | null,
                        invalid_class: (r.errors as Record<string, unknown>).invalid_class as
                            | string
                            | null,
                        invalid_date: (r.errors as Record<string, unknown>).invalid_date as
                            | string
                            | null,
                        server_rejected: null,
                        server_error_type: null,
                    },
                },
            }));

            store.writeRowsBatch(stagedRows);
        } else if (data.type === "done") {
            store.setStage("PREVIEW");
            onDone();
        } else if (data.type === "error") {
            console.error("Import worker error:", data.message);
            // Fall back to main-thread processing
            processOnMainThread(rawRows, columnMapping, classLookup, store, onDone);
        }
    };

    worker.onerror = () => {
        // Fall back to main-thread processing
        processOnMainThread(rawRows, columnMapping, classLookup, store, onDone);
    };

    // Send data to worker
    worker.postMessage({
        type: "process",
        rows: rawRows,
        column_mapping: columnMapping,
        classLookup: Array.from(classLookup.entries()),
    });
}

// ─── Main-thread fallback (no worker available) ──────────────────────────

function processOnMainThread(
    rawRows: Record<string, unknown>[],
    columnMapping: ColumnMapping,
    classLookup: Map<
        string,
        { class_id: string; grade_level: string; stream_name: string; display_label: string }
    >,
    store: ReturnType<typeof useImportStore>,
    onDone: () => void
) {
    const CHUNK_SIZE = 100;
    const total = rawRows.length;

    // Re-import processRow for main-thread processing
    import("../../../lib/import-data/matching").then(({ processRow }) => {
        for (let start = 0; start < total; start += CHUNK_SIZE) {
            const end = Math.min(start + CHUNK_SIZE, total);
            const chunk = [];

            for (let i = start; i < end; i++) {
                const processed = processRow(
                    i,
                    rawRows[i],
                    columnMapping,
                    classLookup as Map<
                        string,
                        { class_id: string; grade_level: string; stream_name: string }
                    >
                );
                chunk.push({
                    row_number: processed.row_number,
                    raw_data: rawRows[i],
                    processed_data: {
                        full_name: processed.full_name,
                        gender: processed.gender,
                        date_of_birth: processed.date_of_birth,
                        class_id: processed.class_id,
                        grade_level: processed.grade_level,
                        stream_name: processed.stream_name,
                        nemis_number: processed.nemis_number,
                        assessment_number: processed.assessment_number,
                        birth_certificate_number: processed.birth_certificate_number,
                    },
                    ui_meta: {
                        has_error: processed.has_error,
                        skipped: false,
                        submitted: false,
                        errors: processed.errors,
                    },
                });
            }

            store.writeRowsBatch(chunk);
        }

        store.setStage("PREVIEW");
        onDone();
    });
}
