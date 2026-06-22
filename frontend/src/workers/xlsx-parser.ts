/**
 * Web Worker for parsing CSV and XLSX files off the main thread.
 *
 * Messages to worker:
 *   { type: "parse", file: ArrayBuffer, fileName: string }
 *
 * Messages from worker (streaming):
 *   { type: "progress", parsed: number, total: number }
 *   { type: "row", row: { temp_id, email, first_name, last_name, phone, registration_number } }
 *   { type: "error", message: string }
 *   { type: "complete", total: number }
 */

import Papa from "papaparse";
import * as XLSX from "xlsx";

// ─── Types ─────────────────────────────────────────────────────────────────

interface ParsedRow {
    temp_id: string;
    email: string;
    first_name: string;
    last_name: string;
    phone: string;
    registration_number: string;
}

// ─── Worker message handler ────────────────────────────────────────────────

self.onmessage = (e: MessageEvent<{ type: string; file: ArrayBuffer; fileName: string }>) => {
    const { type, file, fileName } = e.data;

    if (type !== "parse") return;

    try {
        const ext = fileName.split(".").pop()?.toLowerCase() || "";

        if (ext === "csv") {
            parseCSV(file);
        } else if (ext === "xlsx" || ext === "xls") {
            parseXLSX(file);
        } else {
            self.postMessage({
                type: "error",
                message: `Unsupported file type: .${ext}. Use CSV or XLSX.`,
            });
        }
    } catch (err) {
        self.postMessage({
            type: "error",
            message: err instanceof Error ? err.message : "Unknown parse error",
        });
    }
};

// ─── CSV Parser (streaming via Papa Parse) ─────────────────────────────────

function parseCSV(file: ArrayBuffer) {
    const text = new TextDecoder("utf-8").decode(file);
    let rowCount = 0;
    const MAX_ROWS = 5000;

    Papa.parse(text, {
        header: true,
        skipEmptyLines: true,
        fastMode: true,
        step: (results: Papa.ParseStepResult<Record<string, string>>) => {
            if (results.errors.length > 0) {
                // Log but continue
                return;
            }

            const data = results.data;
            if (!data) return;

            rowCount++;
            if (rowCount > MAX_ROWS) {
                self.postMessage({
                    type: "error",
                    message: `File exceeds maximum of ${MAX_ROWS} rows.`,
                });
                return;
            }

            const row = normalizeRow(data);
            if (row) {
                self.postMessage({ type: "row", row });
            }
        },
        complete: () => {
            self.postMessage({ type: "complete", total: rowCount });
        },
        error: (err: Error) => {
            self.postMessage({ type: "error", message: err.message });
        },
    });
}

// ─── XLSX Parser ───────────────────────────────────────────────────────────

function parseXLSX(file: ArrayBuffer) {
    const workbook = XLSX.read(file, { type: "array" });
    const firstSheet = workbook.Sheets[workbook.SheetNames[0]];
    if (!firstSheet) {
        self.postMessage({ type: "error", message: "No sheets found in workbook." });
        return;
    }

    const jsonData = XLSX.utils.sheet_to_json<Record<string, string>>(firstSheet, { defval: "" });
    const MAX_ROWS = 5000;

    if (jsonData.length > MAX_ROWS) {
        self.postMessage({ type: "error", message: `File exceeds maximum of ${MAX_ROWS} rows.` });
        return;
    }

    let validCount = 0;
    for (const data of jsonData) {
        const row = normalizeRow(data);
        if (row) {
            validCount++;
            self.postMessage({ type: "row", row });
        }
    }

    self.postMessage({ type: "complete", total: validCount });
}

// ─── Row Normalization ─────────────────────────────────────────────────────

/** Column name aliases for flexible import. */
const COLUMN_ALIASES: Record<string, string[]> = {
    email: ["email", "e-mail", "e_mail", "mail"],
    first_name: ["first_name", "firstname", "first name", "given_name", "given name"],
    last_name: ["last_name", "lastname", "last name", "surname", "family_name", "family name"],
    phone: [
        "phone",
        "phone_number",
        "phone number",
        "telephone",
        "tel",
        "mobile",
        "mobile_number",
        "cell",
    ],
    registration_number: [
        "registration_number",
        "registration number",
        "reg_number",
        "reg no",
        "tsc_number",
        "tsc no",
    ],
};

function normalizeRow(data: Record<string, string>): ParsedRow | null {
    // Normalize keys to lowercase
    const normalized: Record<string, string> = {};
    for (const [key, value] of Object.entries(data)) {
        normalized[key.toLowerCase().trim()] = (value ?? "").trim();
    }

    // Map aliased columns
    const row: Record<string, string> = {};
    for (const [canonical, aliases] of Object.entries(COLUMN_ALIASES)) {
        for (const alias of aliases) {
            if (normalized[alias] !== undefined && normalized[alias] !== "") {
                row[canonical] = normalized[alias];
                break;
            }
        }
    }

    // Must have at least email to create a row
    if (!row.email) return null;

    return {
        temp_id: crypto.randomUUID(),
        email: row.email || "",
        first_name: row.first_name || "",
        last_name: row.last_name || "",
        phone: row.phone || "",
        registration_number: row.registration_number || "",
    };
}
