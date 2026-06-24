/**
 * Student Import API calls.
 *
 * Endpoints:
 *   POST /api/v1/imports/students         — bulk import students
 *   GET  /api/v1/students/parents         — list parents for lookup
 *   GET  /api/v1/students/classes         — list active classes
 *   GET  /api/v1/students                 — list existing students (for duplicate detection)
 */

import { api } from "@/lib/api/client";
import { getErrorMessage } from "@/lib/errors";
import type {
    StudentImportPayload,
    ImportResultSummary,
    ImportResponseRow,
    ParentRecord,
    ClassRecord,
    ExistingStudent,
} from "../types";

// ─── Constants ─────────────────────────────────────────────────────────────

const POST_TIMEOUT_MS = 30_000;

// ─── Parent Lookup ────────────────────────────────────────────────────────

export async function fetchParents(): Promise<ParentRecord[]> {
    return api.get<ParentRecord[]>("/api/v1/students/parents");
}

// ─── Class Lookup ─────────────────────────────────────────────────────────

export async function fetchClasses(): Promise<ClassRecord[]> {
    return api.get<ClassRecord[]>("/api/v1/students/classes");
}

// ─── Existing Students (for duplicate detection) ─────────────────────────

export async function fetchExistingStudents(): Promise<ExistingStudent[]> {
    return api.get<ExistingStudent[]>("/api/v1/students");
}

// ─── Bulk POST with timeout ───────────────────────────────────────────────

export interface BulkPostResult {
    summary: ImportResultSummary;
    rawResponse?: unknown;
}

export async function submitBulkImport(payload: StudentImportPayload[]): Promise<BulkPostResult> {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), POST_TIMEOUT_MS);

    try {
        const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "";
        const url = `${API_BASE}/api/v1/imports/students`;

        const res = await fetch(url, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            credentials: "include",
            body: JSON.stringify(payload),
            signal: controller.signal,
        });

        clearTimeout(timeoutId);

        // -- 200/201 Full Success --
        if (res.status === 200 || res.status === 201) {
            return {
                summary: {
                    total: payload.length,
                    successCount: payload.length,
                    failureCount: 0,
                    failures: [],
                    status: "success",
                },
            };
        }

        // -- 207 Multi-Status (partial success) --
        if (res.status === 207) {
            const rows: ImportResponseRow[] = await res.json();
            const successes = rows.filter((r) => r.status === "success");
            const failures = rows.filter((r) => r.status === "error");
            return {
                summary: {
                    total: rows.length,
                    successCount: successes.length,
                    failureCount: failures.length,
                    failures,
                    status: "partial",
                },
                rawResponse: rows,
            };
        }

        // -- 4xx/5xx --
        let body: { message?: string; code?: string } = {};
        try {
            body = await res.json();
        } catch {
            // ignore parse failure
        }

        return {
            summary: {
                total: payload.length,
                successCount: 0,
                failureCount: payload.length,
                failures: payload.map((p, i) => ({
                    index: i,
                    status: "error" as const,
                    full_name: p.full_name,
                    error_message: body.message ?? `HTTP ${res.status}: ${res.statusText}`,
                })),
                status: "error",
                message: body.message ?? `HTTP ${res.status}: ${res.statusText}`,
            },
        };
    } catch (err: unknown) {
        clearTimeout(timeoutId);

        if ((err as DOMException)?.name === "AbortError") {
            return {
                summary: {
                    total: payload.length,
                    successCount: 0,
                    failureCount: payload.length,
                    failures: payload.map((p, i) => ({
                        index: i,
                        status: "error" as const,
                        full_name: p.full_name,
                        error_message: "Request timed out after 30 seconds",
                    })),
                    status: "error",
                    message: "Request timed out after 30 seconds",
                },
            };
        }

        return {
            summary: {
                total: payload.length,
                successCount: 0,
                failureCount: payload.length,
                failures: payload.map((p, i) => ({
                    index: i,
                    status: "error" as const,
                    full_name: p.full_name,
                    error_message: getErrorMessage(err),
                })),
                status: "error",
                message: getErrorMessage(err),
            },
        };
    }
}
