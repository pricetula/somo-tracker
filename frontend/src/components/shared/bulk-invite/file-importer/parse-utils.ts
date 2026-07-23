/**
 * File parsing utilities for CSV/Excel/TSV/ODS staff invitation files.
 * Matches the student import parse-utils pattern.
 */

import Papa from "papaparse";
import * as XLSX from "xlsx";
import type { ParsedFileResult } from "./types";

const MAX_FILE_SIZE_BYTES = 15 * 1024 * 1024; // 15MB

// ─── BOM stripping ────────────────────────────────────────────────────────

function stripBom(text: string): string {
    if (text.charCodeAt(0) === 0xfeff) return text.slice(1);
    return text;
}

// ─── File type detection ──────────────────────────────────────────────────

type FileType = "csv" | "tsv" | "excel" | "ods";

function detectFileType(file: File): FileType {
    const name = file.name.toLowerCase();
    const ext = name.split(".").pop() ?? "";
    if (["xlsx", "xls", "xlsm"].includes(ext)) return "excel";
    if (ext === "ods") return "ods";
    if (ext === "tsv" || ext === "tab") return "tsv";
    return "csv";
}

// ─── Excel serial date conversion ─────────────────────────────────────────

function excelSerialToDate(serial: number): string | null {
    if (serial < 1 || !Number.isFinite(serial)) return null;
    const ms = (serial - 1) * 86400000;
    const date = new Date(Date.UTC(1899, 11, 31) + ms);
    if (isNaN(date.getTime())) return null;
    return date.toISOString().split("T")[0];
}

// ─── Get sheet names ──────────────────────────────────────────────────────

/**
 * Get the names of all sheets in a workbook file.
 * Returns [firstSheetName] for CSV/TSV.
 */
export async function getSheetNames(file: File): Promise<string[]> {
    const fileType = detectFileType(file);
    if (fileType === "csv" || fileType === "tsv") {
        return ["Sheet1"];
    }

    const buffer = await file.arrayBuffer();
    const workbook = XLSX.read(new Uint8Array(buffer), { type: "array" });
    return workbook.SheetNames;
}

// ─── Main parse function ──────────────────────────────────────────────────

export interface ParseOptions {
    sheetName?: string;
}

/**
 * Parse a file (CSV, TSV, Excel, ODS) into a ParsedFileResult.
 * Throws on size limits or parsing failures.
 */
export async function parseFile(file: File, options?: ParseOptions): Promise<ParsedFileResult> {
    if (file.size > MAX_FILE_SIZE_BYTES) {
        throw new Error(
            `File is too large (${(file.size / 1024 / 1024).toFixed(1)}MB). Maximum is 15MB.`
        );
    }

    const fileType = detectFileType(file);

    if (fileType === "csv" || fileType === "tsv") {
        return parseDelimited(file, fileType);
    }

    return parseWorkbook(file, options?.sheetName);
}

// ─── CSV/TSV parsing ──────────────────────────────────────────────────────

function parseDelimited(file: File, fileType: FileType): Promise<ParsedFileResult> {
    return new Promise((resolve, reject) => {
        const reader = new FileReader();

        reader.onload = (e) => {
            try {
                const text = stripBom(e.target?.result as string);
                const delimiter = fileType === "tsv" ? "\t" : "";

                const result = Papa.parse(text, {
                    header: true,
                    dynamicTyping: false,
                    skipEmptyLines: true,
                    delimiter,
                });

                if (result.errors.length > 0 && result.data.length === 0) {
                    reject(new Error(`Failed to parse CSV: ${result.errors[0].message}`));
                    return;
                }

                const headers = result.meta.fields ?? [];
                const rows = result.data as Record<string, string>[];

                if (rows.length === 0) {
                    reject(new Error("The file is empty — no data rows found."));
                    return;
                }

                resolve({
                    headers,
                    rows,
                    total_rows: rows.length,
                    file_name: file.name,
                });
            } catch (err) {
                reject(err instanceof Error ? err : new Error("Failed to parse file"));
            }
        };

        reader.onerror = () => reject(new Error("Failed to read file"));

        if (fileType === "csv" || fileType === "tsv") {
            reader.readAsText(file);
        }
    });
}

// ─── Excel/ODS parsing ────────────────────────────────────────────────────

function parseWorkbook(file: File, sheetName?: string): Promise<ParsedFileResult> {
    return new Promise((resolve, reject) => {
        const reader = new FileReader();

        reader.onload = (e) => {
            try {
                const data = new Uint8Array(e.target?.result as ArrayBuffer);
                const workbook = XLSX.read(data, { type: "array", cellDates: false });

                const targetSheetName = sheetName ?? workbook.SheetNames[0];

                if (!targetSheetName || !workbook.SheetNames.includes(targetSheetName)) {
                    reject(new Error(`Sheet "${targetSheetName}" not found in workbook.`));
                    return;
                }

                const sheet = workbook.Sheets[targetSheetName];
                const jsonData = XLSX.utils.sheet_to_json<Record<string, string>>(sheet, {
                    header: 0,
                    defval: "",
                    raw: false,
                });

                if (jsonData.length === 0) {
                    reject(new Error("The selected sheet is empty."));
                    return;
                }

                const headers = Object.keys(jsonData[0]);

                // Convert any date-like values
                const rows = jsonData.map((row) => {
                    const clean: Record<string, string> = {};
                    for (const [key, val] of Object.entries(row)) {
                        clean[key] =
                            typeof val === "string" ? val : (excelSerialToDate(val) ?? String(val));
                    }
                    return clean;
                });

                resolve({
                    headers,
                    rows,
                    total_rows: rows.length,
                    file_name: file.name,
                    sheet_name: targetSheetName,
                });
            } catch (err) {
                reject(err instanceof Error ? err : new Error("Failed to parse workbook"));
            }
        };

        reader.onerror = () => reject(new Error("Failed to read file"));
        reader.readAsArrayBuffer(file);
    });
}
