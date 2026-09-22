"use client";

import React from "react";
import { ImportOrchestrator } from "@/features/import";
import { useBulkImportStudents } from "../hooks/use-student-import";
import type { MappedRow } from "@/features/import/components/upload/field-mapper";

const STUDENT_IMPORT_FIELDS = [
    {
        key: "admission_number",
        label: "Admission number",
        aliases: ["admission", "adm no", "admission no", "student id"],
        required: true,
    },
    {
        key: "full_name",
        label: "Full name",
        aliases: ["name", "student name", "full name"],
        required: true,
    },
    {
        key: "date_of_birth",
        label: "Date of birth",
        aliases: ["dob", "birth date", "date of birth"],
        required: true,
        validator: (value: unknown) => {
            if (typeof value !== "string" || value.trim() === "") return "Required";
            return null;
        },
    },
    {
        key: "gender",
        label: "Gender",
        aliases: ["sex", "gender"],
        required: false,
    },
];

async function sha256Hex(input: string): Promise<string> {
    const data = new TextEncoder().encode(input);
    const hashBuffer = await crypto.subtle.digest("SHA-256", data);
    const hashArray = Array.from(new Uint8Array(hashBuffer));
    return hashArray.map((b) => b.toString(16).padStart(2, "0")).join("");
}

export function StudentsImportOrchestrator() {
    const mutation = useBulkImportStudents();

    const handleMappedList = async (rows: MappedRow[]) => {
        const students = rows.map((r) => ({
            admission_number: String(r.data.admission_number ?? ""),
            full_name: String(r.data.full_name ?? ""),
            date_of_birth: String(r.data.date_of_birth ?? ""),
            gender: String(r.data.gender ?? ""),
            metadata: r.data.metadata ? r.data.metadata : {},
        }));
        const canonical = JSON.stringify(
            students.slice().sort((a, b) => a.admission_number.localeCompare(b.admission_number))
        );
        const hash = await sha256Hex(canonical);
        const key = `bulk-student-import:${hash}`;
        try {
            sessionStorage.setItem(`bulk-student-import-key:${hash}`, key);
        } catch {}
        const resp = await mutation.mutateAsync({ students, idempotencyKey: key });
        return { job_id: resp.job_id, total_records: resp.total_records, status: resp.status };
    };

    const handleReset = () => {};

    return (
        <ImportOrchestrator
            fieldDef={STUDENT_IMPORT_FIELDS}
            onMappedList={handleMappedList}
            isSubmitting={mutation.isPending}
            progressUrl={(jobId) => `/backend/api/students/jobs/${jobId}/events`}
            showProgress
            onReset={handleReset}
        />
    );
}
