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
} from "./db";
import { validateAndDetectDuplicates } from "./utils/validation-utils";
import type { WizardStep, ParsedFileResult, StagedStudentRecord, ImportSessionMeta } from "./types";
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

// ─── Props ────────────────────────────────────────────────────────────────

interface FileImporterProps {
    onReset: () => void;
    onJobCreated: (jobId: string, totalRecords: number) => void;
}

// ─── Main Component ───────────────────────────────────────────────────────

export function FileImporter({ onReset, onJobCreated }: FileImporterProps) {
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

    // ── Session recovery on mount ──────────────────────────────────────
    React.useEffect(() => {
        async function recoverSession() {
            try {
                const meta = await getSessionMeta();

                if (meta && meta.current_step !== "upload") {
                    // Check if staging has records for steps past upload
                    const records = await getStagedRecords();
                    if (records.length === 0) {
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
                }
            } catch (err) {
                console.error("Session recovery failed:", err);
            } finally {
                setInitializing(false);
            }
        }

        recoverSession();
    }, []);

    // ── Handlers for step transitions ──────────────────────────────────

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
                    raw_row_data: row,
                    status: "valid" as const,
                    errors: [],
                };
            });
        },
        []
    );

    // ── Step handlers ─────────────────────────────────────────────────

    const handleUploadComplete = React.useCallback(
        async (result: ParsedFileResult) => {
            setParsedFile(result);
            setCurrentStep("column_mapping");

            const meta: Partial<ImportSessionMeta> = {
                current_step: "column_mapping",
                file_name: result.file_name,
                source_sheet_name: result.sheet_name,
                total_rows: result.total_rows,
                last_active_tab_id: tabId,
            };
            await saveSessionMeta(meta);
        },
        [tabId]
    );

    const handleMappingComplete = React.useCallback(
        async (mappings: Record<string, string | string[]>) => {
            setResumeMappings(mappings);
            setCurrentStep("class_resolving");
            const meta: Partial<ImportSessionMeta> = {
                current_step: "class_resolving",
                column_mappings: mappings,
                last_active_tab_id: tabId,
            };
            await saveSessionMeta(meta);
        },
        [tabId]
    );

    const handleClassResolveComplete = React.useCallback(
        async (classMappings: Record<string, string>) => {
            if (!parsedFile) return;

            try {
                const staged = buildStagedRecords(parsedFile, resumeMappings ?? {}, classMappings);
                const validated = validateAndDetectDuplicates(staged);

                await bulkWriteStagedRecords(validated);
                const meta: Partial<ImportSessionMeta> = {
                    current_step: "data_review",
                    class_mappings: classMappings,
                    last_active_tab_id: tabId,
                };
                await saveSessionMeta(meta);

                setCurrentStep("data_review");
            } catch (err) {
                console.error("Failed to build staged records:", err);
            }
        },
        [parsedFile, resumeMappings, tabId, buildStagedRecords]
    );

    const handleProceedToStreaming = React.useCallback(async () => {
        setCurrentStep("streaming");
        await updateSessionStep("streaming");
    }, []);

    const handleImportComplete = React.useCallback(() => {
        onReset();
    }, [onReset]);

    const handleImportError = React.useCallback((error: string) => {
        console.error("Import error:", error);
    }, []);

    const handleDiscardDraft = React.useCallback(async () => {
        await clearAllSessions();
        setIsResuming(false);
        setResumeFileName(undefined);
        setResumeMappings(undefined);
        setResumeClassMappings(undefined);
        setParsedFile(null);
        setCurrentStep("upload");
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

    if (initializing) {
        return (
            <div className="flex items-center justify-center py-12">
                <p className="text-muted-foreground text-sm">Restoring session...</p>
            </div>
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
                />
            )}

            {currentStep === "streaming" && (
                <StepStreaming
                    onComplete={handleImportComplete}
                    onError={handleImportError}
                    onJobCreated={onJobCreated}
                />
            )}
        </section>
    );
}
