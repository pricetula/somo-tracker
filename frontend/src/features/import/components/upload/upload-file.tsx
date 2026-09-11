"use client";

import React, { useCallback, useRef, useState } from "react";
import { UploadCloud, Loader2 } from "lucide-react";

let papaPromise: Promise<typeof import("papaparse")> | null = null;
let xlsxPromise: Promise<typeof import("xlsx")> | null = null;

function loadPapa() {
    if (!papaPromise)
        papaPromise = import("papaparse").then(
            (m) => (m as unknown as { default?: typeof import("papaparse") }).default ?? m
        );
    return papaPromise;
}

function loadXLSX() {
    if (!xlsxPromise)
        xlsxPromise = import("xlsx").then(
            (m) => (m as unknown as { default?: typeof import("xlsx") }).default ?? m
        );
    return xlsxPromise;
}

interface UploadFileProps {
    onUploaded: (i: Record<string, object>[]) => void;
}

function isCsv(name: string) {
    return /\.csv$/i.test(name);
}

function isExcel(name: string) {
    return /\.(xlsx|xls)$/i.test(name);
}

export function UploadFile({ onUploaded }: UploadFileProps) {
    const inputRef = useRef<HTMLInputElement>(null);
    const [dragActive, setDragActive] = useState(false);
    const [parsing, setParsing] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const handleFiles = useCallback(
        async (selected: FileList | null) => {
            if (!selected) return;
            const accepted = Array.from(selected).filter((f) => isCsv(f.name) || isExcel(f.name));
            if (accepted.length === 0) {
                setError("Only CSV and Excel (.xlsx, .xls) files are supported.");
                return;
            }
            setError(null);
            setParsing(true);

            let Papa: typeof import("papaparse");
            let XLSX: typeof import("xlsx");
            try {
                [Papa, XLSX] = await Promise.all([loadPapa(), loadXLSX()]);
            } catch (err: unknown) {
                setParsing(false);
                setError(
                    `Failed to load parsing libraries: ${err instanceof Error ? err.message : "unknown error"}`
                );
                return;
            }

            const results: Record<string, object>[] = [];
            let completed = 0;

            const done = () => {
                completed += 1;
                if (completed >= accepted.length) {
                    setParsing(false);
                    onUploaded(results);
                }
            };

            accepted.forEach((file) => {
                const reader = new FileReader();

                reader.onload = (e) => {
                    try {
                        if (isCsv(file.name)) {
                            const text = e.target?.result as string;
                            Papa.parse(text, {
                                header: true,
                                skipEmptyLines: true,
                                complete: (res) => {
                                    if (res.errors && res.errors.length > 0) {
                                        setError(
                                            `Parse error in ${file.name}: ${res.errors[0].message}`
                                        );
                                    } else {
                                        results.push(...(res.data as Record<string, object>[]));
                                    }
                                    done();
                                },
                                error: (err: Error) => {
                                    setError(`Failed to parse ${file.name}: ${err.message}`);
                                    done();
                                },
                            });
                        } else {
                            const data = new Uint8Array(e.target?.result as ArrayBuffer);
                            const workbook = XLSX.read(data, { type: "array" });
                            const firstSheet = workbook.SheetNames[0];
                            const ws = workbook.Sheets[firstSheet];
                            const json = XLSX.utils.sheet_to_json(ws) as Record<string, object>[];
                            results.push(...json);
                            done();
                        }
                    } catch (err: unknown) {
                        setError(
                            `Unexpected error reading ${file.name}: ${err instanceof Error ? err.message : String(err)}`
                        );
                        done();
                    }
                };

                reader.onerror = () => {
                    setError(`Failed to read ${file.name}`);
                    done();
                };

                if (isCsv(file.name)) {
                    reader.readAsText(file);
                } else {
                    reader.readAsArrayBuffer(file);
                }
            });
        },
        [onUploaded]
    );

    const onDragEnter = useCallback((e: React.DragEvent) => {
        e.preventDefault();
        e.stopPropagation();
        setDragActive(true);
    }, []);

    const onDragLeave = useCallback((e: React.DragEvent) => {
        e.preventDefault();
        e.stopPropagation();
        setDragActive(false);
    }, []);

    const onDragOver = useCallback((e: React.DragEvent) => {
        e.preventDefault();
        e.stopPropagation();
        setDragActive(true);
    }, []);

    const onDrop = useCallback(
        (e: React.DragEvent) => {
            e.preventDefault();
            e.stopPropagation();
            setDragActive(false);
            handleFiles(e.dataTransfer.files);
        },
        [handleFiles]
    );

    return (
        <div>
            <div
                onDragEnter={onDragEnter}
                onDragLeave={onDragLeave}
                onDragOver={onDragOver}
                onDrop={onDrop}
                onClick={() => inputRef.current?.click()}
                role="button"
                tabIndex={0}
                aria-label="Drop CSV or Excel files here"
                className={`focus:ring-primary/40 flex cursor-pointer flex-col items-center justify-center rounded-xl border-2 border-dashed px-6 py-10 text-center transition-colors focus:ring-2 focus:outline-none ${
                    dragActive
                        ? "bg-primary/10 border-primary"
                        : "bg-muted/30 border-muted-foreground/30 hover:bg-muted/50"
                }`}
            >
                <input
                    ref={inputRef}
                    type="file"
                    accept=".csv,.xlsx,.xls"
                    multiple
                    className="hidden"
                    onChange={(e) => handleFiles(e.target.files)}
                />
                <div className="bg-primary/10 text-primary mb-3 flex h-12 w-12 items-center justify-center rounded-full">
                    <UploadCloud className="h-6 w-6" />
                </div>
                <p className="text-foreground font-medium">
                    {dragActive ? "Drop files here" : "Click or drag files here"}
                </p>
                <p className="text-muted-foreground mt-1">CSV and Excel (.xlsx, .xls)</p>
            </div>

            {error && (
                <div className="bg-destructive/10 text-destructive rounded-lg px-4 py-3">
                    {error}
                </div>
            )}

            {parsing && (
                <div className="text-muted-foreground flex items-center gap-3">
                    <Loader2 className="h-4 w-4 animate-spin" />
                    Parsing files…
                </div>
            )}
        </div>
    );
}
