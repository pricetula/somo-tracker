"use client";

import { useMutation } from "@tanstack/react-query";
import { getErrorMessage } from "@/lib/errors";
import { toast } from "sonner";
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

export const studentImportKeys = {
    import: ["students", "import"] as const,
};

export function useBulkImportStudents() {
    return useMutation({
        mutationKey: studentImportKeys.import,
        mutationFn: async ({ students, idempotencyKey }: BulkStudentImportPayload) => {
            return createStudentImport({ students, idempotencyKey });
        },
        onSuccess: () => {
            toast.success("Import started");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}
