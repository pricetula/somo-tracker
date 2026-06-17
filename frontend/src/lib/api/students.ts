/**
 * Students API functions.
 *
 * Endpoints:
 *   GET  /api/v1/students              — list students (paginated, searchable)
 *   POST /api/v1/students              — create a single student
 *   POST /api/v1/students/import       — upload CSV for bulk import
 *   GET  /api/v1/students/import/stream — SSE progress stream
 *   GET  /api/v1/students/import/errors — download error CSV
 */

import { api, ApiRequestError } from "./client";

// ─── Types ─────────────────────────────────────────────────────────────────

export interface Student {
    id: string;
    tenant_id: string;
    first_name: string;
    middle_name?: string;
    last_name: string;
    gender: "MALE" | "FEMALE" | "OTHER" | "PREFER_NOT_TO_SAY";
    date_of_birth: string;
    is_active: boolean;
    created_at: string;
}

export interface ListStudentsResponse {
    students: Student[];
    total: number;
}

export interface CreateStudentPayload {
    first_name: string;
    middle_name?: string;
    last_name: string;
    gender: string;
    date_of_birth: string;
}

export interface ImportResponse {
    import_id: string;
}

/** SSE progress event from the backend. */
export interface ImportProgressEvent {
    status: "processing" | "completed" | "error";
    current?: number;
    total?: number;
    success?: number;
    failed?: number;
    download_url?: string;
    error?: string;
}

// ─── API Functions ─────────────────────────────────────────────────────────

/** List students with pagination and optional search. */
export async function listStudents(
    params: { page?: number; per_page?: number; search?: string } = {}
): Promise<ListStudentsResponse> {
    const searchParams = new URLSearchParams();
    if (params.page) searchParams.set("page", String(params.page));
    if (params.per_page) searchParams.set("per_page", String(params.per_page));
    if (params.search) searchParams.set("search", params.search);

    const qs = searchParams.toString();
    return api.get<ListStudentsResponse>(`/api/v1/students${qs ? `?${qs}` : ""}`);
}

/** Create a single student manually. */
export async function createStudent(payload: CreateStudentPayload): Promise<Student> {
    return api.post<Student>("/api/v1/students", payload);
}

/** Upload a CSV file for bulk student import. Returns an import_id. */
export async function importStudentCSV(file: File): Promise<ImportResponse> {
    const formData = new FormData();
    formData.append("file", file);

    const url = process.env.NEXT_PUBLIC_API_URL ?? "";
    const res = await fetch(`${url}/api/v1/students/import`, {
        method: "POST",
        credentials: "include",
        body: formData,
    });

    if (!res.ok) {
        let errBody: { error?: string; message?: string };
        try {
            errBody = await res.json();
        } catch {
            errBody = { message: res.statusText };
        }
        throw new ApiRequestError(res.status, {
            error: errBody.error ?? "unknown",
            message: errBody.message,
        });
    }

    return res.json();
}

/** Download the error CSV for a failed import. */
export async function downloadErrorCSV(errorId: string): Promise<string> {
    const url = process.env.NEXT_PUBLIC_API_URL ?? "";
    const res = await fetch(
        `${url}/api/v1/students/import/errors?id=${encodeURIComponent(errorId)}`,
        { credentials: "include" }
    );

    if (!res.ok) {
        throw new Error("Failed to download error CSV");
    }

    return res.text();
}
