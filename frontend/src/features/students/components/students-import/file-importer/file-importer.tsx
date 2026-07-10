"use client";

/**
 * Crash-Resistant Student Import Wizard
 *
 * Orchestrates the 6-step import flow:
 *   1. App hydration / state recovery
 *   2. File upload & dynamic column mapping
 *   3. Class normalization & token resolver
 *   4. Staging validation & IndexedDB generation
 *   5. Persistent live review & correction table
 *   6. Chunked streaming network layer
 *
 * Key IndexedDB crash-recovery features:
 *   - Session keyed by school_id (G4) — switching schools shows a foreign-session prompt
 *   - Parsed-file data persisted for column_mapping / class_resolving resume (G1)
 *   - Staleness threshold (24h) triggers resume prompt instead of silent auto-resume (G3)
 *   - Stale sessions cleared after any terminal import status (G2)
 *   - Quota pre-check and catch around bulk writes (G5)
 *   - Error handling around all save operations (G7)
 */

import * as React from "react";
import {
    AlertCircle,
    FileText,
    Upload,
    MapPin,
    CheckSquare,
    CloudUpload,
    Trash2,
    Clock,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
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
import { StepUpload } from "./step-upload";
import { StepColumnMapping } from "./step-column-mapping";
import { StepClassResolve } from "./step-class-resolve";
import { StepDataReview } from "./step-data-review";
import { StepStreaming } from "./step-streaming";
import {
    getSessionMeta,
    clearAllSessions,
    saveSessionMeta,
    updateSessionStep,
    bulkWriteStagedRecords,
    getStagedRecords,
    saveParsedFile,
    getParsedFile,
    deleteParsedFile,
    checkStorageForBulkWrite,
} from "./db";
import { validateAndDetectDuplicates } from "./utils/validation-utils";
import { useMe } from "@/hooks/use-auth";
import { toast } from "sonner";
import type { WizardStep, ParsedFileResult, StagedStudentRecord, ImportSessionMeta } from "./types";
import { SESSION_STALE_MS, MAX_PERSISTED_ROWS } from "./types";

// ─── Step indicators ──────────────────────────────────────────────────────

interface StepInfo {
    key: WizardStep;
    label: string;
    icon: React.ReactNode;
}

const STEPS: StepInfo[] = [
    { key: "upload", label: "Upload", icon: <Upload className="size-3.5" /> },
    { key: "column_mapping", label: "Map Columns", icon: <MapPin className="size-3.5" /> },
    { key: "class_resolving", label: "Resolve Classes", icon: <FileText className="size-3.5" /> },
    { key: "data_review", label: "Review", icon: <CheckSquare className="size-3.5" /> },
    { key: "streaming", label: "Import", icon: <CloudUpload className="size-3.5" /> },
];

// ─── Generate session ID for multi-tab detection ──────────────────────────

function generateTabId(): string {
    return `tab_${Date.now()}_${Math.random().toString(36).slice(2, 9)}`;
}

/** Format a date for the stale-session prompt. */
function formatSessionDate(iso: string): string {
    try {
        const d = new Date(iso);
        return d.toLocaleDateString(undefined, {
            month: "short",
            day: "numeric",
            hour: "2-digit",
            minute: "2-digit",
        });
    } catch {
        return iso;
    }
}

// ─── Props ────────────────────────────────────────────────────────────────

interface FileImporterProps {
    onReset: () => void;
    onJobCreated: (jobId: string, totalRecords: number) => void;
}

// ─── Main Component ───────────────────────────────────────────────────────

export function FileImporter({ onReset, onJobCreated }: FileImporterProps) {
    const { data: me } = useMe();
    const schoolId = me?.school_id ?? "";

    // ── State ──────────────────────────────────────────────────────────
    const [currentStep, setCurrentStep] = React.useState<WizardStep>("upload");
    const [tabId] = React.useState(generateTabId);
    const [multiTabWarning, setMultiTabWarning] = React.useState(false);
    const [initializing, setInitializing] = React.useState(true);

    // File parse result
    const [parsedFile, setParsedFile] = React.useState<ParsedFileResult | null>(null);

    // Resume state
    const [isResuming, setIsResuming] = React.useState(false);
    const [resumeFileName, setResumeFileName] = React.useState<string | undefined>();
    const [resumeMappings, setResumeMappings] = React.useState<
        Record<string, string | string[]> | undefined
    >();
    const [resumeClassMappings, setResumeClassMappings] = React.useState<
        Record<string, string> | undefined
    >();

    // G3: Stale-session prompt state (shown instead of silent auto-resume)
    const [staleSessionMeta, setStaleSessionMeta] = React.useState<ImportSessionMeta | null>(null);

    // G4: Foreign-school prompt state
    const [foreignSchoolMeta, setForeignSchoolMeta] = React.useState<ImportSessionMeta | null>(
        null
    );

    // G1: Message shown when a too-large parsed file prevents full resume
    const [resumeTooLargeMessage, setResumeTooLargeMessage] = React.useState<string | null>(null);

    // Track whether a job was submitted so we know to clear on unmount (G2)
    const jobSubmittedRef = React.useRef(false);

    // ── Broadcast channel for multi-tab detection ──────────────────────
    React.useEffect(() => {
        const channel = new BroadcastChannel("somo_student_import");
        channel.postMessage({ type: "tab_online", tabId });

        const handler = (event: MessageEvent) => {
            if (event.data?.type === "tab_online" && event.data?.tabId !== tabId) {
                setMultiTabWarning(true);
            }
        };

        channel.addEventListener("message", handler);

        // Also listen for storage events from other tabs
        const storageHandler = (e: StorageEvent) => {
            if (e.key === "somo_import_active_tab" && e.newValue !== tabId) {
                setMultiTabWarning(true);
            }
        };
        window.addEventListener("storage", storageHandler);

        // Store this tab's id
        try {
            localStorage.setItem("somo_import_active_tab", tabId);
        } catch {}

        return () => {
            channel.removeEventListener("message", handler);
            window.removeEventListener("storage", storageHandler);
            channel.close();
        };
    }, [tabId]);

    // G2: When the parent resets, also clear IndexedDB if a job was submitted
    // (This handles the case where the user clicks "Done" on ImportProgress.)
    // NOTE: The primary G2 cleanup happens in students-import.tsx via the
    // onTerminalStatus callback on ImportProgress. This is a secondary safety net.
    const handleParentReset = React.useCallback(() => {
        if (jobSubmittedRef.current) {
            clearAllSessions().catch(console.error);
            jobSubmittedRef.current = false;
        }
        onReset();
    }, [onReset]);

    // ── Session recovery on mount ──────────────────────────────────────
    React.useEffect(() => {
        if (!schoolId) return; // wait for auth to resolve

        async function recoverSession() {
            try {
                const meta = await getSessionMeta(schoolId);

                if (!meta || meta.current_step === "upload") {
                    setInitializing(false);
                    return;
                }

                // G4: Foreign-school detection — if the stored school_id doesn't match
                if (meta.school_id && meta.school_id !== schoolId) {
                    setForeignSchoolMeta(meta);
                    setInitializing(false);
                    return;
                }

                // G3: Staleness check
                const elapsed = Date.now() - new Date(meta.updated_at).getTime();
                if (elapsed > SESSION_STALE_MS) {
                    // Show prompt instead of auto-resuming
                    setStaleSessionMeta(meta);
                    setInitializing(false);
                    return;
                }

                // Check if staging has records for steps past upload
                const records = await getStagedRecords(schoolId);

                // G1: Restore parsed file for early steps
                if (
                    meta.current_step === "column_mapping" ||
                    meta.current_step === "class_resolving"
                ) {
                    const storedParsed = await getParsedFile(schoolId);
                    if (storedParsed) {
                        setParsedFile(storedParsed);
                    } else if (meta.parsed_file_too_large) {
                        // File was too large to persist — cannot resume these steps
                        setResumeTooLargeMessage(
                            "The previous import draft was too large to fully restore. " +
                                "Please start again from upload."
                        );
                        setResumeFileName(meta.file_name);
                        setInitializing(false);
                        return;
                    } else {
                        // Corrupted draft for early step — clear and restart
                        await clearAllSessions();
                        setInitializing(false);
                        return;
                    }
                }

                // G2: Guard against stale data — if the session says "streaming"
                // and records exist, this might be a completed import. Don't let
                // the user silently re-submit. Check if records are all "submitted"
                // (rarely set since assignBatchToRecords is unused) or if we see
                // a streaming session with no active job.
                if (meta.current_step === "streaming" && records.length > 0) {
                    const allSubmitted = records.every((r) => r.status === "submitted");
                    if (allSubmitted) {
                        // All records already submitted — treat as stale
                        await clearAllSessions();
                        setInitializing(false);
                        return;
                    }
                }

                // Normal resume: staging must have records
                if (records.length === 0 && meta.current_step !== "column_mapping") {
                    // Corrupted draft — clear and restart
                    await clearAllSessions();
                    setInitializing(false);
                    return;
                }

                setIsResuming(true);
                setResumeFileName(meta.file_name);
                setCurrentStep(meta.current_step);
                setResumeMappings(meta.column_mappings);
                setResumeClassMappings(meta.class_mappings);

                // G1: If resuming into data_review, parsed file data is no longer needed
                if (meta.current_step === "data_review" || meta.current_step === "streaming") {
                    await deleteParsedFile(schoolId).catch(() => {});
                }
            } catch (err) {
                console.error("Session recovery failed:", err);
            } finally {
                setInitializing(false);
            }
        }

        recoverSession();
    }, [schoolId]);

    // ── G3: Stale-session handlers ─────────────────────────────────────

    const handleDiscardAndRestart = React.useCallback(async () => {
        await clearAllSessions();
        setStaleSessionMeta(null);
        setForeignSchoolMeta(null);
        setResumeTooLargeMessage(null);
        setIsResuming(false);
        setResumeFileName(undefined);
        setResumeMappings(undefined);
        setResumeClassMappings(undefined);
        setParsedFile(null);
        setCurrentStep("upload");
    }, []);

    const handleResumeStaleSession = React.useCallback(async () => {
        if (!staleSessionMeta) return;
        try {
            const records = await getStagedRecords(schoolId);
            // G1: Restore parsed file for early steps
            if (
                staleSessionMeta.current_step === "column_mapping" ||
                staleSessionMeta.current_step === "class_resolving"
            ) {
                const storedParsed = await getParsedFile(schoolId);
                if (storedParsed) {
                    setParsedFile(storedParsed);
                } else {
                    // Can't resume — cleared or too large
                    await clearAllSessions();
                    setStaleSessionMeta(null);
                    setCurrentStep("upload");
                    return;
                }
            }

            if (
                records.length === 0 &&
                staleSessionMeta.current_step !== "column_mapping" &&
                staleSessionMeta.current_step !== "class_resolving"
            ) {
                // Corrupted — clear and restart
                await clearAllSessions();
                setStaleSessionMeta(null);
                setCurrentStep("upload");
                return;
            }

            setIsResuming(true);
            setResumeFileName(staleSessionMeta.file_name);
            setCurrentStep(staleSessionMeta.current_step);
            setResumeMappings(staleSessionMeta.column_mappings);
            setResumeClassMappings(staleSessionMeta.class_mappings);
        } catch {
            await clearAllSessions();
        }
        setStaleSessionMeta(null);
    }, [staleSessionMeta, schoolId]);

    // ── G4: Foreign-school handler ─────────────────────────────────────

    const handleDiscardForeignSession = React.useCallback(async () => {
        await clearAllSessions();
        setForeignSchoolMeta(null);
        setIsResuming(false);
        setCurrentStep("upload");
    }, []);

    // ── Gender normalization ──────────────────────────────────────────
    /** Normalize gender values from the file to "M" or "F". */
    function normalizeGender(raw: string | undefined | null): string | undefined {
        if (!raw) return undefined;
        const lower = raw.trim().toLowerCase();
        if (["m", "male", "boy", "masculine"].includes(lower)) return "M";
        if (["f", "female", "girl", "feminine"].includes(lower)) return "F";
        return raw.trim() || undefined;
    }

    // ── Build staged records from parsed file + mappings ───────────────

    const buildStagedRecords = React.useCallback(
        (
            file: ParsedFileResult,
            mappings: Record<string, string | string[]>,
            classMappings: Record<string, string>
        ): StagedStudentRecord[] => {
            return file.rows.map((row) => {
                const payload: Record<string, unknown> = {};

                // Process each mapped field
                for (const [targetKey, sourceValue] of Object.entries(mappings)) {
                    if (targetKey === "class_id") {
                        // Class resolution
                        const rawClass = (row[sourceValue as string] ?? "").trim();
                        payload.class_id = classMappings[rawClass] ?? null;
                    } else if (targetKey === "gender") {
                        // Normalize gender to M or F
                        payload.gender = normalizeGender(row[sourceValue as string] as string);
                    } else if (Array.isArray(sourceValue)) {
                        // Multi-column concatenation (e.g., full_name)
                        const parts = sourceValue.map((h) => (row[h] ?? "").trim()).filter(Boolean);
                        payload[targetKey] = parts.join(" ").replace(/\s{2,}/g, " ");
                    } else {
                        payload[targetKey] = (row[sourceValue] ?? "").trim() || undefined;
                    }
                }

                // Ensure full_name exists
                const fn = payload.full_name;
                if (typeof fn !== "string" || fn.trim().length === 0) {
                    payload.full_name = "";
                }

                return {
                    payload: payload as unknown as StagedStudentRecord["payload"],
                    school_id: schoolId,
                    raw_row_data: row,
                    status: "valid" as const,
                    errors: [],
                };
            });
        },
        [schoolId]
    );

    // ── Step handlers ─────────────────────────────────────────────────

    const handleUploadComplete = React.useCallback(
        async (result: ParsedFileResult) => {
            setParsedFile(result);
            setCurrentStep("column_mapping");

            // G1: Persist parsed file for crash recovery of early steps.
            // G1 size guard: skip persisting full row data for very large files.
            const fileTooLarge = result.total_rows > MAX_PERSISTED_ROWS;
            if (!fileTooLarge) {
                try {
                    await saveParsedFile(schoolId, {
                        file_name: result.file_name,
                        sheet_name: result.sheet_name,
                        headers: result.headers,
                        rows: result.rows,
                        total_rows: result.total_rows,
                    });
                } catch (err) {
                    // G7: Log but don't block the wizard
                    console.error("Failed to persist parsed file:", err);
                    toast.error(
                        "Couldn't save your progress — if you leave this page you may need to re-upload."
                    );
                }
            }

            // G7: Error handling around saveSessionMeta
            const meta: Partial<ImportSessionMeta> = {
                current_step: "column_mapping",
                file_name: result.file_name,
                source_sheet_name: result.sheet_name,
                total_rows: result.total_rows,
                parsed_file_too_large: fileTooLarge || undefined,
            };
            try {
                await saveSessionMeta(schoolId, meta);
            } catch (err) {
                console.error("Failed to save session meta:", err);
                toast.error(
                    "Couldn't save your progress — if you leave this page you may need to re-upload."
                );
            }
        },
        [schoolId]
    );

    const handleMappingComplete = React.useCallback(
        async (mappings: Record<string, string | string[]>) => {
            setResumeMappings(mappings);
            setCurrentStep("class_resolving");

            // G7: Error handling around saveSessionMeta
            const meta: Partial<ImportSessionMeta> = {
                current_step: "class_resolving",
                column_mappings: mappings,
            };
            try {
                await saveSessionMeta(schoolId, meta);
            } catch (err) {
                console.error("Failed to save session meta:", err);
                toast.error(
                    "Couldn't save your progress — if you leave this page you may need to re-upload."
                );
            }
        },
        [schoolId]
    );

    const handleClassResolveComplete = React.useCallback(
        async (classMappings: Record<string, string>) => {
            if (!parsedFile) return;

            try {
                const staged = buildStagedRecords(parsedFile, resumeMappings ?? {}, classMappings);
                const validated = validateAndDetectDuplicates(staged);

                // G5: Pre-check storage before bulk write
                const storageMsg = await checkStorageForBulkWrite(validated.length);
                if (storageMsg) {
                    toast.error(storageMsg);
                    return; // Don't proceed — user needs to free space or use a smaller file
                }

                // G5: Wrap bulk write in try/catch for QuotaExceededError
                try {
                    await bulkWriteStagedRecords(validated);
                } catch (writeErr) {
                    console.error("Bulk write failed:", writeErr);
                    toast.error(
                        "Not enough browser storage available to save this many records. " +
                            "Try a smaller file or free up browser storage."
                    );
                    return; // Don't advance to next step with partial data
                }

                // G1: Parsed file data is no longer needed now that staged records exist
                await deleteParsedFile(schoolId).catch(() => {});

                // G7: Error handling around saveSessionMeta
                const meta: Partial<ImportSessionMeta> = {
                    current_step: "data_review",
                    class_mappings: classMappings,
                };
                try {
                    await saveSessionMeta(schoolId, meta);
                } catch (err) {
                    console.error("Failed to save session meta:", err);
                    toast.error(
                        "Couldn't save your progress — if you leave this page you may need to re-upload."
                    );
                }

                setCurrentStep("data_review");
            } catch (err) {
                console.error("Failed to build staged records:", err);
                toast.error("An error occurred while processing the records. Please try again.");
            }
        },
        [parsedFile, resumeMappings, schoolId, buildStagedRecords]
    );

    const handleProceedToStreaming = React.useCallback(async () => {
        setCurrentStep("streaming");
        // G7: Error handling around updateSessionStep
        try {
            await updateSessionStep(schoolId, "streaming");
        } catch (err) {
            console.error("Failed to update session step:", err);
        }
    }, [schoolId]);

    const handleImportComplete = React.useCallback(() => {
        // G2: Mark job as submitted so cleanup happens on reset
        jobSubmittedRef.current = true;
        handleParentReset();
    }, [handleParentReset]);

    const handleImportStarted = React.useCallback(
        (jobId: string, totalRecords: number) => {
            // G2: Mark that a job was submitted from this wizard
            jobSubmittedRef.current = true;
            onJobCreated(jobId, totalRecords);
        },
        [onJobCreated]
    );

    const handleImportError = React.useCallback((error: string) => {
        console.error("Import error:", error);
    }, []);

    const handleDiscardDraft = React.useCallback(async () => {
        // G2+G4: Clear everything for this school
        await clearAllSessions();
        setIsResuming(false);
        setResumeFileName(undefined);
        setResumeMappings(undefined);
        setResumeClassMappings(undefined);
        setParsedFile(null);
        setCurrentStep("upload");
        setStaleSessionMeta(null);
        setForeignSchoolMeta(null);
        setResumeTooLargeMessage(null);
    }, []);

    // ── Guard helpers ─────────────────────────────────────────────────

    const allHeaders = React.useMemo(() => parsedFile?.headers ?? [], [parsedFile]);

    const rows = React.useMemo(() => parsedFile?.rows ?? [], [parsedFile]);

    // ── Step navigation ────────────────────────────────────────────────

    const handleStepBack = React.useCallback(async (fromStep: WizardStep) => {
        switch (fromStep) {
            case "column_mapping":
                setCurrentStep("upload");
                break;
            case "class_resolving":
                setCurrentStep("column_mapping");
                break;
            case "data_review":
                setCurrentStep("class_resolving");
                break;
            default:
                setCurrentStep("upload");
        }
    }, []);

    // ── Render ─────────────────────────────────────────────────────────

    if (!schoolId) {
        return (
            <div className="flex items-center justify-center py-12">
                <p className="text-muted-foreground">Loading user session...</p>
            </div>
        );
    }

    if (initializing) {
        return (
            <div className="flex items-center justify-center py-12">
                <p className="text-muted-foreground">Restoring session...</p>
            </div>
        );
    }

    // G3: Stale-session prompt (shown instead of auto-resume)
    if (staleSessionMeta) {
        return (
            <Alert>
                <Clock className="size-4" />
                <AlertTitle>
                    Unfinished import from {formatSessionDate(staleSessionMeta.updated_at)}
                </AlertTitle>
                <AlertDescription>
                    You have an unfinished import for &ldquo;{staleSessionMeta.file_name}&rdquo;
                    from {formatSessionDate(staleSessionMeta.updated_at)}. This draft is more than
                    24 hours old. You can resume it or discard it and start fresh.
                </AlertDescription>
                <div className="mt-3 flex items-center gap-2">
                    <Button size="sm" onClick={handleResumeStaleSession}>
                        Resume Import
                    </Button>
                    <Button size="sm" variant="ghost" onClick={handleDiscardAndRestart}>
                        Start Fresh
                    </Button>
                </div>
            </Alert>
        );
    }

    // G4: Foreign-school prompt
    if (foreignSchoolMeta) {
        return (
            <Alert>
                <AlertCircle className="size-4" />
                <AlertTitle>Unfinished import for a different school</AlertTitle>
                <AlertDescription>
                    You have an unfinished import for &ldquo;{foreignSchoolMeta.file_name}&rdquo;
                    from a different school. This session cannot be resumed here. Discard it to
                    start a new import.
                </AlertDescription>
                <div className="mt-3 flex items-center gap-2">
                    <Button size="sm" variant="destructive" onClick={handleDiscardForeignSession}>
                        Discard &amp; Start New
                    </Button>
                </div>
            </Alert>
        );
    }

    // G1: Message for too-large files that can't be fully resumed
    if (resumeTooLargeMessage) {
        return (
            <Alert variant="destructive">
                <AlertCircle className="size-4" />
                <AlertTitle>Draft too large to restore</AlertTitle>
                <AlertDescription>{resumeTooLargeMessage}</AlertDescription>
                <div className="mt-3 flex items-center gap-2">
                    <Button size="sm" onClick={handleDiscardAndRestart}>
                        Start New Import
                    </Button>
                </div>
            </Alert>
        );
    }

    const stepIndex = STEPS.findIndex((s) => s.key === currentStep);

    return (
        <section className="space-y-6">
            {/* Multi-tab warning */}
            {multiTabWarning && (
                <Alert variant="destructive" className="py-2">
                    <AlertCircle className="size-4" />
                    <AlertTitle>Import open in another tab</AlertTitle>
                    <AlertDescription>
                        This import session is also open in another browser tab. Editing here may
                        cause conflicts.
                    </AlertDescription>
                </Alert>
            )}

            {/* Step indicator */}
            <div className="flex items-center gap-0">
                {STEPS.map((step, idx) => (
                    <React.Fragment key={step.key}>
                        <div
                            className={`flex items-center gap-1.5 text-xs ${idx < stepIndex ? "text-emerald-600" : ""} ${idx === stepIndex ? "text-foreground font-medium" : ""} ${idx > stepIndex ? "text-muted-foreground" : ""} `}
                        >
                            {step.icon}
                            <span className="hidden sm:inline">{step.label}</span>
                        </div>
                        {idx < STEPS.length - 1 && (
                            <div
                                className={`mx-2 h-px w-6 ${
                                    idx < stepIndex ? "bg-emerald-300" : "bg-border"
                                }`}
                            />
                        )}
                    </React.Fragment>
                ))}
            </div>

            {/* Discard draft button */}
            <div className="flex justify-end">
                <AlertDialog>
                    <AlertDialogTrigger asChild>
                        <Button
                            variant="ghost"
                            size="sm"
                            className="text-muted-foreground h-7 text-xs"
                        >
                            <Trash2 className="mr-1 size-3" />
                            Discard Draft
                        </Button>
                    </AlertDialogTrigger>
                    <AlertDialogContent>
                        <AlertDialogHeader>
                            <AlertDialogTitle>Discard import draft?</AlertDialogTitle>
                            <AlertDialogDescription>
                                This will permanently delete all imported data and mapping
                                configurations. This action cannot be undone.
                            </AlertDialogDescription>
                        </AlertDialogHeader>
                        <AlertDialogFooter>
                            <AlertDialogCancel>Cancel</AlertDialogCancel>
                            <AlertDialogAction variant="destructive" onClick={handleDiscardDraft}>
                                Discard
                            </AlertDialogAction>
                        </AlertDialogFooter>
                    </AlertDialogContent>
                </AlertDialog>
            </div>

            {/* Step content */}
            {currentStep === "upload" && (
                <StepUpload
                    onParsed={handleUploadComplete}
                    onBack={() => onReset()}
                    isResuming={isResuming}
                    resumeFileName={resumeFileName}
                />
            )}

            {currentStep === "column_mapping" && parsedFile && (
                <StepColumnMapping
                    headers={parsedFile.headers}
                    onMappingComplete={handleMappingComplete}
                    onBack={() => handleStepBack("column_mapping")}
                    initialMappings={resumeMappings}
                />
            )}

            {currentStep === "class_resolving" && (
                <StepClassResolve
                    rows={rows}
                    classColumn={
                        typeof resumeMappings?.class_id === "string"
                            ? resumeMappings.class_id
                            : null
                    }
                    allHeaders={allHeaders}
                    onResolveComplete={handleClassResolveComplete}
                    onBack={() => handleStepBack("class_resolving")}
                    initialMappings={resumeClassMappings}
                />
            )}

            {currentStep === "data_review" && (
                <StepDataReview
                    onProceed={handleProceedToStreaming}
                    onBack={() => handleStepBack("data_review")}
                    schoolId={schoolId}
                />
            )}

            {currentStep === "streaming" && (
                <StepStreaming
                    onComplete={handleImportComplete}
                    onError={handleImportError}
                    onJobCreated={handleImportStarted}
                    schoolId={schoolId}
                />
            )}
        </section>
    );
}
