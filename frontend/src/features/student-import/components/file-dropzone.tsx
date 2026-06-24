/**
 * File dropzone for CSV/Excel upload (Pattern B).
 *
 * Validates file size (10MB max) before parsing.
 * Uses PapaParse with web worker for non-blocking parsing.
 * Transfers headers + 5 preview rows first, then full data deferred.
 */

"use client";

import * as React from "react";
import { Upload, FileSpreadsheet } from "lucide-react";
import Papa from "papaparse";
import { toast } from "sonner";
import * as XLSX from "xlsx";
import type { ParsedFileResult } from "../types";

// ─── Constants ─────────────────────────────────────────────────────────────

const MAX_FILE_SIZE_BYTES = 10 * 1024 * 1024; // 10MB
const MAX_ROW_COUNT = 5_000;
const PREVIEW_ROW_COUNT = 5;

interface FileDropzoneProps {
    onFileParsed: (result: ParsedFileResult) => void;
    onBack: () => void;
}

export function FileDropzone({ onFileParsed, onBack }: FileDropzoneProps) {
    const [dragOver, setDragOver] = React.useState(false);
    const [parsing, setParsing] = React.useState(false);
    const [parseProgress, setParseProgress] = React.useState<string | null>(null);
    const fileInputRef = React.useRef<HTMLInputElement>(null);

    async function parseCSV(file: File) {
        return new Promise<void>((resolve, reject) => {
            // Step 1: Parse headers + preview with worker
            setParseProgress("Parsing file headers…");

            Papa.parse(file, {
                worker: true,
                header: true,
                skipEmptyLines: true,
                preview: PREVIEW_ROW_COUNT + 1, // header + 5 rows
                complete: async (previewResult) => {
                    if (!previewResult.data || previewResult.data.length === 0) {
                        toast.error("File appears to be empty");
                        setParsing(false);
                        resolve();
                        return;
                    }

                    const headers = previewResult.meta.fields ?? [];
                    const previewRows = previewResult.data as Record<string, string>[];

                    // Check row count before full parse
                    setParseProgress("Counting rows…");

                    // We need total row count. Do a quick line count
                    const lineCount = await countCSVRows(file);

                    if (lineCount > MAX_ROW_COUNT) {
                        toast.error(
                            `This file contains ${lineCount.toLocaleString()} rows. The maximum supported per import is ${MAX_ROW_COUNT.toLocaleString()}. Please split the file and import in batches.`
                        );
                        setParsing(false);
                        resolve();
                        return;
                    }

                    // Step 2: Deferred full parse
                    setParseProgress(`Parsing ${lineCount.toLocaleString()} rows…`);

                    Papa.parse(file, {
                        worker: true,
                        header: true,
                        skipEmptyLines: true,
                        complete: (fullResult) => {
                            const fullData = fullResult.data as Record<string, string>[];
                            onFileParsed({
                                headers,
                                previewRows: previewRows.slice(0, PREVIEW_ROW_COUNT),
                                totalRows: fullData.length,
                                fullData,
                                fileName: file.name,
                            });
                            setParsing(false);
                            resolve();
                        },
                        error: () => {
                            toast.error("Failed to parse CSV file");
                            setParsing(false);
                            reject();
                        },
                    });
                },
                error: () => {
                    toast.error("Failed to parse CSV file");
                    setParsing(false);
                    reject();
                },
            });
        });
    }

    async function parseExcel(file: File) {
        setParseProgress("Parsing Excel file…");

        const buffer = await file.arrayBuffer();
        const workbook = XLSX.read(buffer, { type: "array" });
        const sheetName = workbook.SheetNames[0];
        if (!sheetName) {
            toast.error("Excel file has no sheets");
            setParsing(false);
            return;
        }

        const sheet = workbook.Sheets[sheetName];
        const jsonData = XLSX.utils.sheet_to_json<Record<string, string>>(sheet, {
            defval: "",
        });

        if (jsonData.length > MAX_ROW_COUNT) {
            toast.error(
                `This file contains ${jsonData.length.toLocaleString()} rows. The maximum supported per import is ${MAX_ROW_COUNT.toLocaleString()}. Please split the file and import in batches.`
            );
            setParsing(false);
            return;
        }

        const headers = jsonData.length > 0 ? Object.keys(jsonData[0]) : [];
        const previewRows = jsonData.slice(0, PREVIEW_ROW_COUNT);

        onFileParsed({
            headers,
            previewRows,
            totalRows: jsonData.length,
            fullData: jsonData,
            fileName: file.name,
        });
        setParsing(false);
    }

    async function countCSVRows(file: File): Promise<number> {
        return new Promise((resolve) => {
            Papa.parse(file, {
                worker: true,
                header: true,
                skipEmptyLines: true,
                complete: (result) => {
                    resolve(result.data.length);
                },
                error: () => {
                    resolve(0);
                },
            });
        });
    }

    function handleDrop(e: React.DragEvent) {
        e.preventDefault();
        setDragOver(false);
        const file = e.dataTransfer.files?.[0];
        if (file) handleFile(file);
    }

    function handleDragOver(e: React.DragEvent) {
        e.preventDefault();
        setDragOver(true);
    }

    function handleDragLeave() {
        setDragOver(false);
    }

    function handleClick() {
        fileInputRef.current?.click();
    }

    function handleFileInput(e: React.ChangeEvent<HTMLInputElement>) {
        const file = e.target.files?.[0];
        if (file) handleFile(file);
        // Reset so the same file can be re-selected
        e.target.value = "";
    }

    async function handleFile(file: File) {
        // Validate file size
        if (file.size > MAX_FILE_SIZE_BYTES) {
            toast.error(
                `File too large (${(file.size / 1024 / 1024).toFixed(1)}MB). Maximum is 10MB.`
            );
            return;
        }

        const ext = file.name.split(".").pop()?.toLowerCase();

        setParsing(true);
        setParseProgress("Reading file…");

        try {
            if (ext === "csv") {
                await parseCSV(file);
            } else if (ext === "xlsx" || ext === "xls") {
                await parseExcel(file);
            } else {
                toast.error("Unsupported file format. Please upload CSV or Excel (.xlsx).");
                setParsing(false);
            }
        } catch {
            toast.error("Failed to parse file");
            setParsing(false);
        }
    }

    return (
        <div className="space-y-4">
            <h3 className="text-sm font-medium">Upload File</h3>
            <p className="text-muted-foreground text-xs">
                Supported formats: CSV, XLSX. Maximum file size: 10MB. Maximum rows: 5,000.
            </p>

            <div
                onDrop={handleDrop}
                onDragOver={handleDragOver}
                onDragLeave={handleDragLeave}
                onClick={handleClick}
                className={`flex cursor-pointer flex-col items-center justify-center gap-3 rounded-md border-2 border-dashed px-6 py-12 transition-colors ${
                    dragOver
                        ? "border-primary bg-primary/5"
                        : "border-muted-foreground/20 hover:border-muted-foreground/40"
                }`}
            >
                {parsing ? (
                    <>
                        <FileSpreadsheet className="text-primary size-10 animate-pulse" />
                        <p className="text-muted-foreground text-sm">
                            {parseProgress ?? "Parsing…"}
                        </p>
                    </>
                ) : (
                    <>
                        <Upload className="text-muted-foreground size-10" />
                        <p className="text-muted-foreground text-sm">
                            Drop a CSV or Excel file here, or click to browse
                        </p>
                    </>
                )}
            </div>

            <input
                ref={fileInputRef}
                type="file"
                accept=".csv,.xlsx,.xls"
                className="hidden"
                onChange={handleFileInput}
            />

            <div className="flex items-center justify-between">
                <button
                    onClick={onBack}
                    className="text-muted-foreground hover:text-foreground text-sm"
                >
                    Back
                </button>
            </div>
        </div>
    );
}
