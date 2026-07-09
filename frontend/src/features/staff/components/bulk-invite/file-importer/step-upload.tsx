"use client";

/**
 * StepUpload — file upload & parsing for staff bulk invitation files.
 * Matches the student import StepUpload pattern.
 */

import * as React from "react";
import { Upload, FileText, AlertCircle, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Progress } from "@/components/ui/progress";
import { getErrorMessage } from "@/lib/errors";
import { parseFile, getSheetNames, type ParseOptions } from "./parse-utils";
import { getStorageEstimate } from "./db";
import type { ParsedFileResult } from "./types";

// MAX_INVITE_ROWS must stay in sync with backend imports.MaxImportRows (5000).
const MAX_INVITE_ROWS = 5000;

interface StepUploadProps {
    onParsed: (result: ParsedFileResult) => void;
    onBack?: () => void;
    isResuming?: boolean;
    resumeFileName?: string;
}

export function StepUpload({ onParsed, onBack, isResuming, resumeFileName }: StepUploadProps) {
    const [file, setFile] = React.useState<File | null>(null);
    const [parsing, setParsing] = React.useState(false);
    const [parseProgress, setParseProgress] = React.useState(0);
    const [error, setError] = React.useState<string | null>(null);
    const [sheets, setSheets] = React.useState<string[]>([]);
    const [showSheetPicker, setShowSheetPicker] = React.useState(false);
    const [parseResult, setParseResult] = React.useState<ParsedFileResult | null>(null);
    const [nearQuota, setNearQuota] = React.useState(false);
    const [rowLimitExceeded, setRowLimitExceeded] = React.useState(false);
    const inputRef = React.useRef<HTMLInputElement>(null);

    // Check storage quota on mount
    React.useEffect(() => {
        getStorageEstimate()
            .then((est) => {
                if (est.nearQuota) {
                    setNearQuota(true);
                }
            })
            .catch(() => {});
    }, []);

    const handleFileSelect = React.useCallback(
        async (selectedFile: File | null) => {
            if (!selectedFile) return;
            setFile(selectedFile);
            setError(null);
            setParsing(true);
            setParseProgress(10);
            setParseResult(null);
            setRowLimitExceeded(false);

            try {
                // Check for multiple sheets
                const sheetNames = await getSheetNames(selectedFile);
                if (sheetNames.length > 1) {
                    setSheets(sheetNames);
                    setShowSheetPicker(true);
                    setParsing(false);
                    setParseProgress(60);
                    return;
                }

                setParseProgress(40);
                const result = await parseFile(selectedFile);
                setParseProgress(100);

                setParseResult(result);
                setParsing(false);

                // Block progression if row count exceeds the limit
                if (result.total_rows > MAX_INVITE_ROWS) {
                    setRowLimitExceeded(true);
                    return;
                }

                // Auto-proceed after brief delay
                setTimeout(() => {
                    onParsed(result);
                }, 300);
            } catch (err) {
                setError(getErrorMessage(err));
                setParsing(false);
            }
        },
        [onParsed]
    );

    const handleSheetSelect = React.useCallback(
        async (sheetName: string) => {
            if (!file) return;
            setShowSheetPicker(false);
            setParsing(true);
            setParseProgress(40);

            try {
                const result = await parseFile(file, { sheetName } as ParseOptions);
                setParseProgress(100);

                setParseResult(result);
                setParsing(false);

                if (result.total_rows > MAX_INVITE_ROWS) {
                    setRowLimitExceeded(true);
                    return;
                }

                setTimeout(() => onParsed(result), 300);
            } catch (err) {
                setError(getErrorMessage(err));
                setParsing(false);
            }
        },
        [file, onParsed]
    );

    const handleDrop = React.useCallback(
        (e: React.DragEvent) => {
            e.preventDefault();
            const droppedFile = e.dataTransfer.files[0];
            handleFileSelect(droppedFile);
        },
        [handleFileSelect]
    );

    const handleDragOver = React.useCallback((e: React.DragEvent) => {
        e.preventDefault();
    }, []);

    const handleInputChange = React.useCallback(
        (e: React.ChangeEvent<HTMLInputElement>) => {
            handleFileSelect(e.target.files?.[0] ?? null);
        },
        [handleFileSelect]
    );

    return (
        <div className="space-y-4">
            {/* Resume notice */}
            {isResuming && resumeFileName && (
                <Alert>
                    <AlertCircle className="size-4" />
                    <AlertTitle>Resuming saved import draft</AlertTitle>
                    <AlertDescription>
                        Resuming saved import draft for &ldquo;{resumeFileName}&rdquo;
                    </AlertDescription>
                </Alert>
            )}

            {/* Near quota warning */}
            {nearQuota && (
                <Alert variant="destructive">
                    <AlertCircle className="size-4" />
                    <AlertTitle>Browser storage nearly full</AlertTitle>
                    <AlertDescription>
                        Your browser&apos;s storage is nearly full. This may affect the import.
                        Consider freeing up space before proceeding.
                    </AlertDescription>
                </Alert>
            )}

            {/* Error */}
            {error && (
                <Alert variant="destructive">
                    <AlertCircle className="size-4" />
                    <AlertTitle>Error</AlertTitle>
                    <AlertDescription>{error}</AlertDescription>
                </Alert>
            )}

            {/* Sheet picker */}
            {showSheetPicker && (
                <div className="space-y-2">
                    <p className="text-sm font-medium">Select a sheet to import:</p>
                    <div className="flex flex-wrap gap-2">
                        {sheets.map((name) => (
                            <Button
                                key={name}
                                variant="outline"
                                onClick={() => handleSheetSelect(name)}
                                disabled={parsing}
                            >
                                <FileText className="mr-2 size-4" />
                                {name}
                            </Button>
                        ))}
                    </div>
                </div>
            )}

            {/* Upload zone */}
            {!showSheetPicker && !parsing && !parseResult && (
                <div
                    onDrop={handleDrop}
                    onDragOver={handleDragOver}
                    className="border-muted-foreground/25 hover:border-muted-foreground/50 flex cursor-pointer flex-col items-center gap-3 rounded-lg border-2 border-dashed p-12 transition-colors"
                    onClick={() => inputRef.current?.click()}
                >
                    <Upload className="text-muted-foreground size-8" />
                    <div className="text-center">
                        <p className="text-sm font-medium">
                            Drop your file here, or click to browse
                        </p>
                        <p className="text-muted-foreground mt-1 text-xs">
                            Supports CSV, TSV, Excel (.xlsx, .xls), and ODS files (max 15MB)
                        </p>
                    </div>
                    <input
                        ref={inputRef}
                        type="file"
                        className="hidden"
                        accept=".csv,.tsv,.xlsx,.xls,.xlsm,.ods,.tab"
                        onChange={handleInputChange}
                    />
                </div>
            )}

            {/* Parse progress */}
            {parsing && (
                <div className="space-y-2">
                    <div className="text-muted-foreground flex items-center gap-2 text-sm">
                        <Loader2 className="size-4 animate-spin" />
                        <span>Parsing file...</span>
                    </div>
                    <Progress value={parseProgress} className="h-1.5" />
                </div>
            )}

            {/* Row limit exceeded error */}
            {parseResult && !parsing && rowLimitExceeded && (
                <Alert variant="destructive">
                    <AlertCircle className="size-4" />
                    <AlertTitle>Row limit exceeded</AlertTitle>
                    <AlertDescription>
                        This file contains {parseResult.total_rows.toLocaleString()} rows; the
                        maximum is {MAX_INVITE_ROWS.toLocaleString()}. Please split into smaller
                        files and import each separately.
                    </AlertDescription>
                </Alert>
            )}

            {/* Quick parse result */}
            {parseResult && !parsing && !rowLimitExceeded && (
                <Alert>
                    <FileText className="size-4" />
                    <AlertTitle>File parsed successfully</AlertTitle>
                    <AlertDescription>
                        {parseResult.total_rows.toLocaleString()} /{" "}
                        {MAX_INVITE_ROWS.toLocaleString()} rows with {parseResult.headers.length}{" "}
                        columns
                        {parseResult.sheet_name ? ` (sheet: ${parseResult.sheet_name})` : ""}
                    </AlertDescription>
                </Alert>
            )}

            {/* Back button */}
            {onBack && !parsing && (
                <Button variant="ghost" size="sm" onClick={onBack}>
                    Back to import options
                </Button>
            )}
        </div>
    );
}
