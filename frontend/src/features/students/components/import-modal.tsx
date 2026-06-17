/**
 * Import Modal — CSV upload dropzone with client-side validation.
 *
 * On successful upload, the modal collapses and a mini progress widget
 * (import-progress.tsx) takes over in the bottom-right corner.
 */

"use client";

import * as React from "react";
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
    DialogClose,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Upload, FileText, AlertCircle, X } from "lucide-react";
import { cn } from "@/lib/utils";

import { useImportCSV } from "@/features/students/hooks/use-students";
import { ImportProgress } from "./import-progress";

// ─── Types ─────────────────────────────────────────────────────────────────

interface ImportModalProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
}

type DropState = "idle" | "dragging" | "validating" | "invalid" | "uploading";

// ─── Component ─────────────────────────────────────────────────────────────

export function ImportModal({ open, onOpenChange }: ImportModalProps) {
    const [dropState, setDropState] = React.useState<DropState>("idle");
    const [file, setFile] = React.useState<File | null>(null);
    const [validationError, setValidationError] = React.useState<string>("");
    const [importId, setImportId] = React.useState<string | null>(null);
    const [showProgress, setShowProgress] = React.useState(false);

    const importCSV = useImportCSV();
    const inputRef = React.useRef<HTMLInputElement>(null);

    // Handle dialog open/close — reset state when closing
    function handleOpenChange(newOpen: boolean) {
        if (!newOpen) {
            setDropState("idle");
            setFile(null);
            setValidationError("");
            setImportId(null);
            setShowProgress(false);
        }
        onOpenChange(newOpen);
    }

    function validateFile(f: File): string | null {
        // Check file extension
        const ext = f.name.split(".").pop()?.toLowerCase();
        if (ext !== "csv") {
            return "Only CSV files are accepted.";
        }
        // Check file size (max 32MB)
        if (f.size > 32 * 1024 * 1024) {
            return "File exceeds the 32MB maximum size.";
        }
        if (f.size === 0) {
            return "File is empty.";
        }
        return null;
    }

    function handleFile(f: File) {
        const error = validateFile(f);
        if (error) {
            setValidationError(error);
            setDropState("invalid");
            setFile(null);
            return;
        }

        setFile(f);
        setValidationError("");
        setDropState("idle");
    }

    function handleDrop(e: React.DragEvent) {
        e.preventDefault();
        e.stopPropagation();
        setDropState("idle");

        const droppedFile = e.dataTransfer.files[0];
        if (droppedFile) {
            handleFile(droppedFile);
        }
    }

    function handleDragOver(e: React.DragEvent) {
        e.preventDefault();
        e.stopPropagation();
        setDropState("dragging");
    }

    function handleDragLeave(e: React.DragEvent) {
        e.preventDefault();
        e.stopPropagation();
        setDropState("idle");
    }

    function handleInputChange(e: React.ChangeEvent<HTMLInputElement>) {
        const selectedFile = e.target.files?.[0];
        if (selectedFile) {
            handleFile(selectedFile);
        }
    }

    async function handleUpload() {
        if (!file) return;

        setDropState("uploading");
        try {
            const result = await importCSV.mutateAsync(file);
            setImportId(result.import_id);
            setShowProgress(true);
            // Keep the dialog open briefly, or close it
            // The progress widget is independent
        } catch {
            setDropState("idle");
        }
    }

    function handleImportComplete() {
        // The user can dismiss the progress widget manually
    }

    // Download standard CSV template
    function downloadTemplate() {
        const template =
            "first_name,middle_name,last_name,gender,date_of_birth\nJohn,,Doe,MALE,2010-01-15\nJane,Marie,Smith,FEMALE,2011-06-22";
        const blob = new Blob([template], { type: "text/csv" });
        const url = URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        a.download = "student_import_template.csv";
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
    }

    const isProcessing = dropState === "uploading";

    return (
        <>
            <Dialog open={open} onOpenChange={handleOpenChange}>
                <DialogContent className="sm:max-w-lg">
                    <DialogHeader>
                        <DialogTitle>Import Students</DialogTitle>
                        <DialogDescription>
                            Upload a CSV file to bulk-import student records.
                        </DialogDescription>
                    </DialogHeader>

                    <div className="p-4">
                        {/* Dropzone */}
                        <div
                            onDrop={handleDrop}
                            onDragOver={handleDragOver}
                            onDragLeave={handleDragLeave}
                            onClick={() => inputRef.current?.click()}
                            className={cn(
                                "relative flex cursor-pointer flex-col items-center justify-center gap-2 rounded-lg border-2 border-dashed p-10 transition-colors",
                                dropState === "dragging" && "border-primary bg-primary/5",
                                dropState === "invalid" && "border-destructive bg-destructive/5",
                                dropState !== "dragging" &&
                                    dropState !== "invalid" &&
                                    "border-muted-foreground/20 hover:border-muted-foreground/40"
                            )}
                        >
                            <input
                                ref={inputRef}
                                type="file"
                                accept=".csv"
                                className="hidden"
                                onChange={handleInputChange}
                            />

                            {dropState === "invalid" ? (
                                <AlertCircle className="text-destructive size-8" />
                            ) : (
                                <Upload className="text-muted-foreground size-8" />
                            )}

                            <div className="text-center">
                                <p className="text-sm font-medium">
                                    {dropState === "invalid"
                                        ? "Invalid file"
                                        : "Drop CSV here or click to browse"}
                                </p>
                                <p className="text-muted-foreground mt-1 text-xs">
                                    Only .csv files up to 32MB
                                </p>
                            </div>

                            {validationError && (
                                <p className="text-destructive max-w-xs text-center text-xs">
                                    {validationError}
                                </p>
                            )}
                        </div>

                        {/* Selected file */}
                        {file && dropState !== "invalid" && (
                            <div className="bg-muted/50 mt-4 flex items-center gap-3 rounded-md p-3">
                                <FileText className="text-muted-foreground size-5 shrink-0" />
                                <div className="min-w-0 flex-1">
                                    <p className="truncate text-sm font-medium">{file.name}</p>
                                    <p className="text-muted-foreground text-xs">
                                        {(file.size / 1024).toFixed(1)} KB
                                    </p>
                                </div>
                                <button
                                    onClick={() => {
                                        setFile(null);
                                        setDropState("idle");
                                        setValidationError("");
                                    }}
                                    disabled={isProcessing}
                                    className="text-muted-foreground hover:text-foreground transition-colors"
                                >
                                    <X className="size-4" />
                                </button>
                            </div>
                        )}

                        {/* Template download */}
                        <div className="mt-3 text-center">
                            <button
                                onClick={downloadTemplate}
                                className="text-muted-foreground hover:text-foreground text-xs underline underline-offset-2 transition-colors"
                            >
                                Download standard CSV template
                            </button>
                        </div>

                        {/* Actions */}
                        <div className="mt-6 flex justify-end gap-3">
                            <DialogClose asChild>
                                <Button variant="outline" type="button">
                                    Cancel
                                </Button>
                            </DialogClose>
                            <Button
                                onClick={handleUpload}
                                disabled={!file || isProcessing || importCSV.isPending}
                            >
                                {isProcessing ? "Uploading..." : "Import Students"}
                            </Button>
                        </div>
                    </div>
                </DialogContent>
            </Dialog>

            {/* Mini progress widget (bottom-right) */}
            {showProgress && importId && (
                <ImportProgress
                    importId={importId}
                    onDismiss={() => setShowProgress(false)}
                    onComplete={handleImportComplete}
                />
            )}
        </>
    );
}
