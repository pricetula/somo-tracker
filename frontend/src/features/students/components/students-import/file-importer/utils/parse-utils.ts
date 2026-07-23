/**
 * File parsing utilities for CSV/Excel/TSV/ODS files.
 *
 * Uses:
 *  - papaparse for CSV/TSV parsing
 *  - xlsx for Excel/ODS parsing with serial date conversion
 */

import Papa from "papaparse";
import * as XLSX from "xlsx";
import type { ParsedFileResult } from "../types";

const MAX_FILE_SIZE_BYTES = 15 * 1024 * 1024; // 15MB hard cap
const STREAMING_THRESHOLD = 1000; // rows above this use streaming/worker parse

// ─── BOM stripping ────────────────────────────────────────────────────────

function stripBom(text: string): string {
    if (text.charCodeAt(0) === 0xfeff) {
        return text.slice(1);
    }
    return text;
}

// ─── Excel serial date conversion ─────────────────────────────────────────

/**
 * Convert an Excel serial date number to an ISO date string.
 * Excel serial date 1 = 1900-01-01 (with the Lotus 123 leap year bug).
 */
function excelSerialToDate(serial: number): string | null {
    if (serial < 1 || !Number.isFinite(serial)) return null;
    // Excel epoch: 1900-01-01 (but serial 60 = 1900-02-29 due to Lotus bug)
    const utcDays = serial - 1;
    const ms = utcDays * 86400000 - (serial > 60 ? 0 : 0); // skip the bug correction for simplicity
    const date = new Date(Date.UTC(1899, 11, 31) + ms);
    if (isNaN(date.getTime())) return null;
    return date.toISOString().split("T")[0];
}

/**
 * Check if a value looks like an Excel serial date number.
 * Must be a finite number between 1 and ~150000 (covers dates up to year ~2310).
 */
function isExcelSerialDate(value: unknown): value is number {
    return typeof value === "number" && Number.isFinite(value) && value >= 1 && value <= 150000;
}

// ─── CSV Parsing ──────────────────────────────────────────────────────────

function parseCsv(
    text: string,
    delimiter?: string
): { headers: string[]; rows: Record<string, string>[] } {
    const cleaned = stripBom(text);

    const result = Papa.parse<Record<string, string>>(cleaned, {
        header: true,
        skipEmptyLines: true,
        dynamicTyping: false, // keep everything as strings
        delimiter,
    });

    if (result.errors.length > 0) {
        const critical = result.errors.filter(
            (e) => e.type === "FieldMismatch" || e.type === "Quotes"
        );
        if (critical.length > 0) {
            console.warn("CSV parse warnings:", critical);
        }
    }

    const headers = result.meta.fields ?? [];
    const rows = result.data.filter((row) => headers.some((h) => (row[h] ?? "").trim().length > 0));

    return { headers, rows };
}

// ─── Excel / ODS Parsing ──────────────────────────────────────────────────

interface ExcelParseResult {
    headers: string[];
    rows: Record<string, string>[];
    sheetNames: string[];
}

function parseExcel(
    data: Uint8Array,
    sheetName?: string
): ExcelParseResult & { selectedSheet: string } {
    const workbook = XLSX.read(data, {
        type: "array",
        cellDates: false, // we handle date conversion manually
        raw: true, // get raw values (numbers for dates)
    });

    const sheetNames = workbook.SheetNames;

    let targetSheet: string;
    if (sheetName && sheetNames.includes(sheetName)) {
        targetSheet = sheetName;
    } else {
        // Find first non-empty sheet
        targetSheet = sheetNames[0];
        for (const name of sheetNames) {
            const sheet = workbook.Sheets[name];
            if (sheet && XLSX.utils.sheet_to_json(sheet, { header: 1 }).length > 1) {
                targetSheet = name;
                break;
            }
        }
    }

    const sheet = workbook.Sheets[targetSheet];
    if (!sheet) {
        throw new Error(`Sheet "${targetSheet}" not found in workbook`);
    }

    // Get all rows as arrays (raw values)
    const rows: unknown[][] = XLSX.utils.sheet_to_json(sheet, {
        header: 1,
        defval: "",
        raw: true,
    });

    if (rows.length < 2) {
        return {
            headers: [],
            rows: [],
            sheetNames,
            selectedSheet: targetSheet,
        };
    }

    // First row is headers
    const rawHeaders = rows[0] as string[];
    const headers = rawHeaders.map((h) => String(h ?? "").trim());

    // Detect UPI/KNEC columns — treat as text
    const numberColumns = new Set<number>();
    headers.forEach((h, i) => {
        const lower = h.toLowerCase().replace(/[^a-z0-9]/g, "");
        if (lower.includes("upi") || lower.includes("knec") || lower.includes("assessment")) {
            numberColumns.add(i);
        }
    });

    // Process data rows
    const dataRows: Record<string, string>[] = [];
    for (let i = 1; i < rows.length; i++) {
        const row = rows[i] as unknown[];
        // Skip completely empty rows
        if (!row || row.every((cell) => cell === "" || cell === undefined || cell === null)) {
            continue;
        }

        const record: Record<string, string> = {};
        let hasData = false;

        for (let j = 0; j < headers.length; j++) {
            let value = row[j];
            if (value === undefined || value === null) {
                value = "";
            }

            const header = headers[j];

            // Excel serial date conversion for any column
            if (isExcelSerialDate(value)) {
                const isoDate = excelSerialToDate(value);
                record[header] = isoDate ?? String(value);
            } else if (numberColumns.has(j) && typeof value === "number") {
                // Force text representation for UPI/KNEC columns to preserve leading zeros
                record[header] = String(value);
            } else {
                record[header] = String(value).trim();
            }

            if (record[header].length > 0) hasData = true;
        }

        if (hasData) {
            dataRows.push(record);
        }
    }

    return { headers, rows: dataRows, sheetNames, selectedSheet: targetSheet };
}

// ─── File Type Detection ──────────────────────────────────────────────────

export type FileType = "csv" | "tsv" | "excel" | "ods";

function detectFileType(file: File): FileType {
    const name = file.name.toLowerCase();
    const ext = name.split(".").pop() ?? "";

    if (["xlsx", "xls", "xlsm"].includes(ext)) return "excel";
    if (ext === "ods") return "ods";
    if (ext === "tsv" || ext === "tab") return "tsv";
    return "csv"; // default to csv
}

// ─── Main Parse Function ──────────────────────────────────────────────────

export interface ParseOptions {
    /** For multi-sheet workbooks, which sheet to parse. */
    sheetName?: string;
    /** CSV delimiter override. Auto-detected if not set. */
    delimiter?: string;
}

export interface ParseResult {
    success: true;
    data: ParsedFileResult;
    /** If the workbook has multiple sheets, list them here. */
    availableSheets?: string[];
    /** True if rows exceeded streaming threshold (main thread might block). */
    largeFile: boolean;
}

export interface ParseError {
    success: false;
    error: string;
}

export type FileParseResult = ParseResult | ParseError;

/**
 * Parse a file into headers and rows.
 * Handles CSV, TSV, Excel (.xlsx, .xls, .xlsm), and ODS.
 */
export async function parseFile(file: File, options?: ParseOptions): Promise<FileParseResult> {
    // File size check
    if (file.size > MAX_FILE_SIZE_BYTES) {
        return {
            success: false,
            error: `File exceeds the ${MAX_FILE_SIZE_BYTES / 1024 / 1024}MB size limit. Please split your file into smaller chunks.`,
        };
    }

    // Empty file check
    if (file.size === 0) {
        return {
            success: false,
            error: "The file is empty.",
        };
    }

    const fileType = detectFileType(file);

    try {
        if (fileType === "csv" || fileType === "tsv") {
            const text = await file.text();
            const delimiter = options?.delimiter ?? (fileType === "tsv" ? "\t" : undefined);
            const { headers, rows } = parseCsv(text, delimiter);

            if (headers.length === 0) {
                return {
                    success: false,
                    error: "No headers found in the file. Make sure the first row contains column names.",
                };
            }

            if (rows.length === 0) {
                return {
                    success: false,
                    error: "No data rows found in this file.",
                };
            }

            return {
                success: true,
                data: {
                    file_name: file.name,
                    headers,
                    rows,
                    total_rows: rows.length,
                },
                largeFile: rows.length >= STREAMING_THRESHOLD,
            };
        }

        // Excel or ODS
        const buffer = await file.arrayBuffer();
        const data = new Uint8Array(buffer);
        const result = parseExcel(data, options?.sheetName);

        if (result.headers.length === 0) {
            return {
                success: false,
                error: "No headers found in the file. Make sure the first row contains column names.",
            };
        }

        if (result.rows.length === 0) {
            return {
                success: false,
                error: "No data rows found in this file.",
            };
        }

        return {
            success: true,
            data: {
                file_name: file.name,
                sheet_name: result.selectedSheet,
                headers: result.headers,
                rows: result.rows,
                total_rows: result.rows.length,
            },
            availableSheets: result.sheetNames.length > 1 ? result.sheetNames : undefined,
            largeFile: result.rows.length >= STREAMING_THRESHOLD,
        };
    } catch (err) {
        const message =
            err instanceof Error
                ? err.message
                : "Failed to parse the file. Please check the format and try again.";
        return { success: false, error: message };
    }
}

/**
 * Get available sheet names from an Excel/ODS file without parsing all rows.
 */
export async function getSheetNames(file: File): Promise<string[]> {
    if (!["excel", "ods"].includes(detectFileType(file))) {
        return [];
    }

    try {
        const buffer = await file.arrayBuffer();
        const data = new Uint8Array(buffer);
        const workbook = XLSX.read(data, { type: "array", raw: true });
        return workbook.SheetNames;
    } catch {
        return [];
    }
}
