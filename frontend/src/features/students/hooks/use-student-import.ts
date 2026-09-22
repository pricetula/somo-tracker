"use client";

import { useMutation } from "@tanstack/react-query";
import { api } from "@/lib/api/client";

export interface StudentImportRow {
    admission_number: string;
    full_name: string;
    date_of_birth: string;
    gender?: string;
    metadata?: Record<string, unknown>;
}

export interface BulkStudentImportPayload {
    students: StudentImportRow[];
    idempotencyKey: string;
}

export interface BulkStudentImportResponse {
    job_id: string;
    status: string;
    total_records: number;
    message?: string;
}

export async function createStudentImport(
    payload: BulkStudentImportPayload
): Promise<BulkStudentImportResponse> {
    return api.post<BulkStudentImportResponse>(
        "/api/students/add",
        { students: payload.students },
        {
            headers: { "Idempotency-Key": payload.idempotencyKey },
        }
    );
}

export function useBulkImportStudents() {
    return useMutation({
        mutationFn: async ({ students, idempotencyKey }: BulkStudentImportPayload) => {
            return createStudentImport({ students, idempotencyKey });
        },
    });
}
