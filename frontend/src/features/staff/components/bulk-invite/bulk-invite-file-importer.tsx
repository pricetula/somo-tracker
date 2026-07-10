"use client";

/**
 * Crash-Resistant Staff Bulk Invite Wizard
 *
 * Matches the student import FileImporter pattern with IndexedDB crash recovery,
 * resume/discard draft, multi-tab detection, and storage quota checks.
 *
 * Steps:
 *   1. App hydration / state recovery
 *   2. File upload & dynamic column mapping
 *   3. Staging validation & IndexedDB generation
 *   4. Persistent live review & correction table
 *   5. Submit — POST to /api/v1/staff/invite
 */

import * as React from "react";
import { AlertCircle, Upload, MapPin, CheckSquare, Send, Trash2, Clock } from "lucide-react";
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
import { StepUpload } from "./file-importer/step-upload";
import { StepColumnMapping } from "./file-importer/step-column-mapping";
import { StepDataReview } from "./file-importer/step-data-review";
import { StepStreaming } from "./file-importer/step-streaming";
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
} from "./file-importer/db";
import { validateAndDetectDuplicates } from "./file-importer/validation-utils";
import { useMe } from "@/hooks/use-auth";
import { toast } from "sonner";
import type {
    WizardStep,
    ParsedFileResult,
    StagedInviteRecord,
    ImportSessionMeta,
} from "./file-importer/types";
import { SESSION_STALE_MS, MAX_PERSISTED_ROWS } from "./file-importer/types";

// ─── Step indicators ──────────────────────────────────────────────────────

interface StepInfo {
    key: WizardStep;
    label: string;
    icon: React.ReactNode;
}

const STEPS: StepInfo[] = [
    { key: "upload", label: "Upload", icon: <Upload className="size-3.5" /> },
    { key: "column_mapping", label: "Map Columns", icon: <MapPin className="size-3.5" /> },
    { key: "data_review", label: "Review", icon: <CheckSquare className="size-3.5" /> },
    { key: "streaming", label: "Send", icon: <Send className="size-3.5" /> },
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

interface BulkInviteFileImporterProps {
    role: string;
    onReset: () => void;
    onJobCreated: (jobId: string, totalRecords: number) => void;
}

// ─── Main Component ───────────────────────────────────────────────────────

export function BulkInviteFileImporter({
    role,
    onReset,
    onJobCreated,
}: BulkInviteFileImporterProps) {
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

    // G3: Stale-session prompt state
    const [staleSessionMeta, setStaleSessionMeta] = React.useState<ImportSessionMeta | null>(null);

    // G4: Foreign-school prompt state
    const [foreignSchoolMeta, setForeignSchoolMeta] = React.useState<ImportSessionMeta | null>(
        null
    );

    // G1: Message shown when a too-large parsed file prevents full resume
    const [resumeTooLargeMessage, setResumeTooLargeMessage] = React.useState<string | null>(null);

    // Track whether a job was submitted so we know to clear on unmount
    const jobSubmittedRef = React.useRef(false);

    // ── Broadcast channel for multi-tab detection ──────────────────────
    React.useEffect(() => {
        const channel = new BroadcastChannel("somo_staff_invite");
        channel.postMessage({ type: "tab_online", tabId });

        const handler = (event: MessageEvent) => {
            if (event.data?.type === "tab_online" && event.data?.tabId !== tabId) {
                setMultiTabWarning(true);
            }
        };

        channel.addEventListener("message", handler);

        const storageHandler = (e: StorageEvent) => {
            if (e.key === "somo_staff_invite_active_tab" && e.newValue !== tabId) {
                setMultiTabWarning(true);
            }
        };
        window.addEventListener("storage", storageHandler);

        try {
            localStorage.setItem("somo_staff_invite_active_tab", tabId);
        } catch {}

        return () => {
            channel.removeEventListener("message", handler);
            window.removeEventListener("storage", storageHandler);
            channel.close();
        };
    }, [tabId]);

    // ── Session recovery on mount ──────────────────────────────────────
    React.useEffect(() => {
        if (!schoolId) return;

        async function recoverSession() {
            try {
                const meta = await getSessionMeta(schoolId);

                if (!meta || meta.current_step === "upload") {
                    setInitializing(false);
                    return;
                }

                // G4: Foreign-school detection
                if (meta.school_id && meta.school_id !== schoolId) {
                    setForeignSchoolMeta(meta);
                    setInitializing(false);
                    return;
                }

                // G3: Staleness check
                const elapsed = Date.now() - new Date(meta.updated_at).getTime();
                if (elapsed > SESSION_STALE_MS) {
                    setStaleSessionMeta(meta);
                    setInitializing(false);
                    return;
                }

                // Check if staging has records for steps past upload
                const records = await getStagedRecords(schoolId);

                // G1: Restore parsed file for early steps
                if (meta.current_step === "column_mapping") {
                    const storedParsed = await getParsedFile(schoolId);
                    if (storedParsed) {
                        setParsedFile(storedParsed);
                    } else if (meta.parsed_file_too_large) {
                        setResumeTooLargeMessage(
                            "The previous import draft was too large to fully restore. " +
                                "Please start again from upload."
                        );
                        setResumeFileName(meta.file_name);
                        setInitializing(false);
                        return;
                    } else {
                        await clearAllSessions();
                        setInitializing(false);
                        return;
                    }
                }

                // Guard against stale data
                if (meta.current_step === "streaming" && records.length > 0) {
                    const allSubmitted = records.every((r) => r.status === "submitted");
                    if (allSubmitted) {
                        await clearAllSessions();
                        setInitializing(false);
                        return;
                    }
                }

                // Normal resume: staging must have records
                if (records.length === 0 && meta.current_step !== "column_mapping") {
                    await clearAllSessions();
                    setInitializing(false);
                    return;
                }

                setIsResuming(true);
                setResumeFileName(meta.file_name);
                setCurrentStep(meta.current_step);
                setResumeMappings(meta.column_mappings);

                // G1: If resuming into data_review or streaming, parsed file no longer needed
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
        setParsedFile(null);
        setCurrentStep("upload");
    }, []);

    const handleResumeStaleSession = React.useCallback(async () => {
        if (!staleSessionMeta) return;
        try {
            const records = await getStagedRecords(schoolId);
            if (staleSessionMeta.current_step === "column_mapping") {
                const storedParsed = await getParsedFile(schoolId);
                if (storedParsed) {
                    setParsedFile(storedParsed);
                } else {
                    await clearAllSessions();
                    setStaleSessionMeta(null);
                    setCurrentStep("upload");
                    return;
                }
            }

            if (records.length === 0 && staleSessionMeta.current_step !== "column_mapping") {
                await clearAllSessions();
                setStaleSessionMeta(null);
                setCurrentStep("upload");
                return;
            }

            setIsResuming(true);
            setResumeFileName(staleSessionMeta.file_name);
            setCurrentStep(staleSessionMeta.current_step);
            setResumeMappings(staleSessionMeta.column_mappings);
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

    // ── Build staged records from parsed file + mappings ───────────────

    const buildStagedRecords = React.useCallback(
        (
            file: ParsedFileResult,
            mappings: Record<string, string | string[]>
        ): StagedInviteRecord[] => {
            // Determine required fields: at minimum we need an "email" column mapped
            const emailCol = mappings.email;
            if (!emailCol) return [];

            return file.rows.map((row) => {
                let email = "";
                let full_name = "";

                // Extract email
                if (typeof emailCol === "string") {
                    email = (row[emailCol] ?? "").trim();
                }

                // Extract full_name (optional)
                const nameCol = mappings.full_name;
                if (nameCol) {
                    if (Array.isArray(nameCol)) {
                        const parts = nameCol.map((h) => (row[h] ?? "").trim()).filter(Boolean);
                        full_name = parts.join(" ").replace(/\s{2,}/g, " ");
                    } else if (typeof nameCol === "string") {
                        full_name = (row[nameCol] ?? "").trim();
                    }
                }

                return {
                    email,
                    full_name,
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
                    console.error("Failed to persist parsed file:", err);
                    toast.error(
                        "Couldn't save your progress — if you leave this page you may need to re-upload."
                    );
                }
            }

            const meta: Partial<ImportSessionMeta> = {
                current_step: "column_mapping",
                file_name: result.file_name,
                source_sheet_name: result.sheet_name,
                total_rows: result.total_rows,
                parsed_file_too_large: fileTooLarge || undefined,
                role,
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
        [schoolId, role]
    );

    const handleMappingComplete = React.useCallback(
        async (mappings: Record<string, string | string[]>) => {
            if (!parsedFile) return;

            try {
                const staged = buildStagedRecords(parsedFile, mappings);
                const validated = validateAndDetectDuplicates(staged);

                // G5: Pre-check storage before bulk write
                const storageMsg = await checkStorageForBulkWrite(validated.length);
                if (storageMsg) {
                    toast.error(storageMsg);
                    return;
                }

                try {
                    await bulkWriteStagedRecords(validated);
                } catch (writeErr) {
                    console.error("Bulk write failed:", writeErr);
                    toast.error(
                        "Not enough browser storage available to save this many records. " +
                            "Try a smaller file or free up browser storage."
                    );
                    return;
                }

                // G1: Parsed file data is no longer needed now that staged records exist
                await deleteParsedFile(schoolId).catch(() => {});

                setResumeMappings(mappings);

                const meta: Partial<ImportSessionMeta> = {
                    current_step: "data_review",
                    column_mappings: mappings,
                    role,
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
        [parsedFile, schoolId, role, buildStagedRecords]
    );

    const handleProceedToStreaming = React.useCallback(async () => {
        setCurrentStep("streaming");
        try {
            await updateSessionStep(schoolId, "streaming");
        } catch (err) {
            console.error("Failed to update session step:", err);
        }
    }, [schoolId]);

    const handleImportStarted = React.useCallback(
        (jobId: string, totalRecords: number) => {
            jobSubmittedRef.current = true;
            onJobCreated(jobId, totalRecords);
        },
        [onJobCreated]
    );

    const handleImportError = React.useCallback((error: string) => {
        console.error("Import error:", error);
    }, []);

    const handleDiscardDraft = React.useCallback(async () => {
        await clearAllSessions();
        setIsResuming(false);
        setResumeFileName(undefined);
        setResumeMappings(undefined);
        setParsedFile(null);
        setCurrentStep("upload");
        setStaleSessionMeta(null);
        setForeignSchoolMeta(null);
        setResumeTooLargeMessage(null);
    }, []);

    // ── Step navigation ────────────────────────────────────────────────

    const handleStepBack = React.useCallback(async (fromStep: WizardStep) => {
        switch (fromStep) {
            case "column_mapping":
                setCurrentStep("upload");
                break;
            case "data_review":
                setCurrentStep("column_mapping");
                break;
            default:
                setCurrentStep("upload");
        }
    }, []);

    // ── Guard helpers ─────────────────────────────────────────────────

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

    // G3: Stale-session prompt
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
                            className={`flex items-center gap-1.5 text-xs ${
                                idx < stepIndex ? "text-emerald-600" : ""
                            } ${idx === stepIndex ? "text-foreground font-medium" : ""} ${
                                idx > stepIndex ? "text-muted-foreground" : ""
                            } `}
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

            {currentStep === "data_review" && (
                <StepDataReview
                    onProceed={handleProceedToStreaming}
                    onBack={() => handleStepBack("data_review")}
                    schoolId={schoolId}
                />
            )}

            {currentStep === "streaming" && (
                <StepStreaming
                    role={role}
                    onError={handleImportError}
                    onJobCreated={handleImportStarted}
                    schoolId={schoolId}
                />
            )}
        </section>
    );
}
