/**
 * File Upload Panel — drag-and-drop zone with Web Worker parsing.
 *
 * Detects CSV vs XLSX by file extension/MIME type.
 * Parses off the main thread and streams results into local state.
 * Enforces maximum 5,000 rows client-side.
 */

"use client";

import * as React from "react";
import { useVirtualizer } from "@tanstack/react-virtual";
import { Upload, AlertCircle, CheckCircle2, Loader2 } from "lucide-react";

import { saveDraft, type ImportDraftRow } from "@/lib/db";
import type { AllowedRole } from "./bulk-staff-import-dialog";

// ─── Types ─────────────────────────────────────────────────────────────────

interface FileUploadPanelProps {
    onRowsReady: (rows: ImportDraftRow[]) => void;
    role: AllowedRole;
    tenantID: string;
    userID: string;
    context: string;
}

type UploadState = "idle" | "dragging" | "parsing" | "complete" | "error";

// ─── Component ─────────────────────────────────────────────────────────────

export function FileUploadPanel({ onRowsReady, tenantID, userID, context }: FileUploadPanelProps) {
    const [uploadState, setUploadState] = React.useState<UploadState>("idle");
    const [rows, setRows] = React.useState<ImportDraftRow[]>([]);
    const [errorMessage, setErrorMessage] = React.useState<string>("");
    const [parsedCount, setParsedCount] = React.useState(0);
    const workerRef = React.useRef<Worker | null>(null);
    const parentRef = React.useRef<HTMLDivElement>(null);

    // Cleanup worker on unmount
    React.useEffect(() => {
        return () => {
            if (workerRef.current) {
                workerRef.current.terminate();
            }
        };
    }, []);

    // Persist rows to IndexedDB as they arrive
    React.useEffect(() => {
        if (rows.length > 0) {
            saveDraft(tenantID, userID, context, rows);
        }
    }, [rows, tenantID, userID, context]);

    function handleDragOver(e: React.DragEvent) {
        e.preventDefault();
        e.stopPropagation();
        setUploadState("dragging");
    }

    function handleDragLeave(e: React.DragEvent) {
        e.preventDefault();
        e.stopPropagation();
        setUploadState("idle");
    }

    function handleDrop(e: React.DragEvent) {
        e.preventDefault();
        e.stopPropagation();
        setUploadState("idle");

        const file = e.dataTransfer.files[0];
        if (file) {
            processFile(file);
        }
    }

    function handleFileSelect(e: React.ChangeEvent<HTMLInputElement>) {
        const file = e.target.files?.[0];
        if (file) {
            processFile(file);
        }
    }

    function processFile(file: File) {
        setRows([]);
        setParsedCount(0);
        setErrorMessage("");

        // Validate file type by extension
        const ext = file.name.split(".").pop()?.toLowerCase();
        if (!ext || !["csv", "xlsx", "xls"].includes(ext)) {
            setUploadState("error");
            setErrorMessage("Please upload a CSV (.csv) or Excel (.xlsx) file.");
            return;
        }

        if (ext === "csv" && file.size > 10 * 1024 * 1024) {
            setUploadState("error");
            setErrorMessage("CSV files must be under 10MB.");
            return;
        }

        if ((ext === "xlsx" || ext === "xls") && file.size > 20 * 1024 * 1024) {
            setUploadState("error");
            setErrorMessage("Excel files must be under 20MB.");
            return;
        }

        setUploadState("parsing");

        // Read as ArrayBuffer and send to Web Worker
        const reader = new FileReader();
        reader.onload = (e) => {
            const arrayBuffer = e.target?.result as ArrayBuffer;

            // Terminate previous worker if any
            if (workerRef.current) {
                workerRef.current.terminate();
            }

            const worker = new Worker(new URL("@/workers/xlsx-parser", import.meta.url), {
                type: "module",
            });
            workerRef.current = worker;

            worker.onmessage = (msg) => {
                const data = msg.data;

                switch (data.type) {
                    case "row":
                        setRows((prev) => [...prev, data.row]);
                        setParsedCount((prev) => prev + 1);
                        break;
                    case "error":
                        setUploadState("error");
                        setErrorMessage(data.message);
                        worker.terminate();
                        break;
                    case "complete":
                        setUploadState("complete");
                        worker.terminate();
                        break;
                }
            };

            worker.postMessage({ type: "parse", file: arrayBuffer, fileName: file.name });
        };
        reader.readAsArrayBuffer(file);
    }

    function handleProceed() {
        if (rows.length === 0) return;
        onRowsReady(rows);
    }

    // Virtualized preview when parsing is complete
    // eslint-disable-next-line react-hooks/incompatible-library
    const virtualizer = useVirtualizer({
        count: Math.min(rows.length, 50), // preview up to 50 rows
        getScrollElement: () => parentRef.current,
        estimateSize: () => 32,
        overscan: 5,
    });

    return (
        <div className="flex flex-col gap-4 p-2">
            {uploadState === "idle" || uploadState === "dragging" ? (
                <div
                    onDragOver={handleDragOver}
                    onDragLeave={handleDragLeave}
                    onDrop={handleDrop}
                    className={`flex cursor-pointer flex-col items-center justify-center rounded-lg border-2 border-dashed p-12 transition-colors ${
                        uploadState === "dragging"
                            ? "border-primary bg-primary/5"
                            : "border-muted-foreground/25 hover:border-muted-foreground/50"
                    }`}
                    onClick={() => document.getElementById("file-input")?.click()}
                >
                    <Upload className="text-muted-foreground mb-3 size-10" />
                    <p className="text-muted-foreground mb-1 text-sm font-medium">
                        Drop your file here, or click to browse
                    </p>
                    <p className="text-muted-foreground/60 text-xs">CSV or XLSX up to 5,000 rows</p>
                    <input
                        id="file-input"
                        type="file"
                        accept=".csv,.xlsx,.xls"
                        onChange={handleFileSelect}
                        className="hidden"
                    />
                </div>
            ) : uploadState === "parsing" ? (
                <div className="flex flex-col items-center justify-center gap-3 py-12">
                    <Loader2 className="text-primary size-8 animate-spin" />
                    <p className="text-muted-foreground text-sm">Parsing file...</p>
                    {parsedCount > 0 && (
                        <p className="text-muted-foreground/60 text-xs">{parsedCount} rows found</p>
                    )}
                </div>
            ) : uploadState === "error" ? (
                <div className="flex flex-col items-center justify-center gap-3 py-8">
                    <AlertCircle className="text-destructive size-8" />
                    <p className="text-destructive text-sm font-medium">{errorMessage}</p>
                    <button
                        onClick={() => setUploadState("idle")}
                        className="text-muted-foreground hover:text-foreground text-xs underline"
                    >
                        Try another file
                    </button>
                </div>
            ) : uploadState === "complete" ? (
                <div className="flex flex-col gap-3">
                    <div className="flex items-center gap-2">
                        <CheckCircle2 className="size-5 text-emerald-600" />
                        <p className="text-sm font-medium">
                            {rows.length} row{rows.length !== 1 ? "s" : ""} parsed successfully
                        </p>
                    </div>

                    {rows.length > 0 && (
                        <div ref={parentRef} className="max-h-72 overflow-auto rounded-md border">
                            <table className="w-full text-left text-xs">
                                <thead className="bg-muted/50 sticky top-0">
                                    <tr>
                                        <th className="px-2 py-1.5 font-medium">#</th>
                                        <th className="px-2 py-1.5 font-medium">Email</th>
                                        <th className="px-2 py-1.5 font-medium">First Name</th>
                                        <th className="px-2 py-1.5 font-medium">Last Name</th>
                                        <th className="px-2 py-1.5 font-medium">Phone</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {virtualizer.getVirtualItems().map((virtualItem) => {
                                        const row = rows[virtualItem.index];
                                        return (
                                            <tr
                                                key={virtualItem.index}
                                                className="border-b last:border-0"
                                                style={{
                                                    height: `${virtualItem.size}px`,
                                                    transform: `translateY(${virtualItem.start - virtualItem.index * virtualItem.size}px)`,
                                                }}
                                            >
                                                <td className="text-muted-foreground px-2 py-1">
                                                    {virtualItem.index + 1}
                                                </td>
                                                <td className="px-2 py-1">{row.email}</td>
                                                <td className="px-2 py-1">{row.first_name}</td>
                                                <td className="px-2 py-1">{row.last_name}</td>
                                                <td className="px-2 py-1">{row.phone || "—"}</td>
                                            </tr>
                                        );
                                    })}
                                </tbody>
                            </table>
                        </div>
                    )}

                    <div className="flex items-center justify-between">
                        <button
                            onClick={() => {
                                setRows([]);
                                setUploadState("idle");
                            }}
                            className="text-muted-foreground hover:text-foreground text-xs underline"
                        >
                            Import different file
                        </button>
                        <button
                            onClick={handleProceed}
                            className="bg-primary text-primary-foreground hover:bg-primary/90 rounded-md px-4 py-1.5 text-sm font-medium"
                        >
                            Review & Submit
                        </button>
                    </div>
                </div>
            ) : null}
        </div>
    );
}
