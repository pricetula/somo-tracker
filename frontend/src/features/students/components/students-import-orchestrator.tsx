"use client";

import React from "react";
import { ImportOrchestrator } from "@/features/import";
import { useBulkImportStudents } from "../hooks/use-student-import";
import type { MappedRow } from "@/features/import/components/upload/field-mapper";
import { api } from "@/lib/api/client";
import { Button } from "@/components/ui/button";

type BulkJobItem = {
  id: string;
  job_id: string;
  row_index: number;
  payload: string;
  result?: string;
  status: string;
  attempt_count: number;
  last_error?: string;
  created_at: string;
  updated_at: string;
};

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
    const [idempotencyKey, setIdempotencyKey] = React.useState<string | null>(null);
    const [jobId, setJobId] = React.useState<string | null>(null);
    const [items, setItems] = React.useState<BulkJobItem[]>([]);
    const [loadingItems, setLoadingItems] = React.useState(false);

    const handleMappedList = async (rows: MappedRow[]) => {
        const students = rows.map((r) => ({
            admission_number: String(r.data.admission_number ?? ""),
            full_name: String(r.data.full_name ?? ""),
            date_of_birth: String(r.data.date_of_birth ?? ""),
            gender: String(r.data.gender ?? ""),
            metadata: (r.data.metadata ?? {}) as Record<string, unknown>,
        }));
        const canonical = JSON.stringify(
            students.slice().sort((a, b) => a.admission_number.localeCompare(b.admission_number))
        );
        const hash = await sha256Hex(canonical);
        const key = `bulk-student-import:${hash}`;
        setIdempotencyKey(key);
        try {
            sessionStorage.setItem(`bulk-student-import-key:${hash}`, key);
        } catch {}
        const resp = await mutation.mutateAsync({ students, idempotencyKey: key });
        return { job_id: resp.job_id, total_records: resp.total_records, status: resp.status };
    };

    const handleReset = () => {
        setIdempotencyKey(null);
        setJobId(null);
        setItems([]);
    };

    const handleJobCompleted = async (jid: string) => {
        setJobId(jid);
        setLoadingItems(true);
        try {
            const data = await api.get<BulkJobItem[]>(`/api/students/jobs/${jid}/items`);
            setItems(data ?? []);
        } catch {
            setItems([]);
        } finally {
            setLoadingItems(false);
        }
    };

    return (
        <div className="space-y-4">
            {idempotencyKey && (
                <div className="text-muted-foreground text-xs">
                    Idempotency key:{" "}
                    <span className="text-foreground font-mono">{idempotencyKey}</span>
                </div>
            )}
            <ImportOrchestrator
                fieldDef={STUDENT_IMPORT_FIELDS}
                onMappedList={handleMappedList}
                isSubmitting={mutation.isPending}
                progressUrl={(jid) => `/backend/api/students/jobs/${jid}/events`}
                showProgress
                onReset={handleReset}
                onJobCompleted={handleJobCompleted}
            />
            {jobId && items.length > 0 && (
                <div className="space-y-2">
                    <div className="text-foreground font-medium">Failed rows</div>
                    <div className="text-muted-foreground text-sm">
                        Job {jobId} –{" "}
                        {
                            items.filter((i) => i.status === "FAILED" || i.status === "DEFERRED")
                                .length
                        }{" "}
                        failed items
                    </div>
                    <div className="space-y-2">
                        {items
                            .filter((i) => i.status === "FAILED" || i.status === "DEFERRED")
                            .map((item) => {
                                let payload: Record<string, unknown> = {};
                                try {
                                    payload = JSON.parse(item.payload);
                                } catch {}
                                return (
                                    <div
                                        key={item.id}
                                        className="border-muted flex items-center justify-between rounded border p-3"
                                    >
                                        <div className="text-sm">
                                            <div className="font-medium">
                                                {String(payload.full_name ?? "—")} –{" "}
                                                {String(payload.admission_number ?? "—")}
                                            </div>
                                            <div className="text-destructive text-xs">
                                                {item.last_error || "No error details"}
                                            </div>
                                        </div>
                                        <Button
                                            size="sm"
                                            variant="outline"
                                            onClick={() => {
                                                // Retry single row – re-submit payload as a mini import
                                                // For now, just reset to allow a new import
                                                handleReset();
                                            }}
                                        >
                                            Retry
                                        </Button>
                                    </div>
                                );
                            })}
                    </div>
                </div>
            )}
            {loadingItems && (
                <div className="text-muted-foreground text-sm">Loading failed rows…</div>
            )}
        </div>
    );
}
