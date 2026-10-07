"use client";

import React from "react";
import { ImportOrchestrator } from "@/features/import";
import { useBulkInviteTeachers } from "../hooks/use-teacher-invitations";
import type { MappedRow } from "@/features/import/components/upload/field-mapper";

const TEACHER_INVITE_FIELDS = [
    {
        key: "full_name",
        label: "Full name",
        aliases: ["name", "first name", "first_name", "full name"],
        required: true,
    },
    {
        key: "email",
        label: "Email",
        aliases: ["e-mail", "mail", "email address"],
        required: true,
        validator: (value: unknown) => {
            if (typeof value !== "string" || !value.includes("@")) return "Invalid email";
            return null;
        },
    },
];

async function sha256Hex(input: string): Promise<string> {
    const data = new TextEncoder().encode(input);
    const hashBuffer = await crypto.subtle.digest("SHA-256", data);
    const hashArray = Array.from(new Uint8Array(hashBuffer));
    return hashArray.map((b) => b.toString(16).padStart(2, "0")).join("");
}

export function TeacherInviteOrchestrator() {
    const mutation = useBulkInviteTeachers();

    const handleMappedList = async (rows: MappedRow[]) => {
        const invitations = rows.map((r) => ({
            email: String(r.data.email ?? ""),
            full_name: String(r.data.full_name ?? ""),
        }));

        const canonical = JSON.stringify(
            invitations.slice().sort((a, b) => a.email.localeCompare(b.email))
        );
        const hash = await sha256Hex(canonical);
        const key = `bulk-teacher-invite:${hash}`;

        try {
            sessionStorage.setItem(`bulk-teacher-invite-key:${hash}`, key);
        } catch {}

        const resp = await mutation.mutateAsync({ invitations, idempotencyKey: key });
        return { job_id: resp.job_id, total_records: resp.total_records, status: resp.status };
    };

    const handleReset = () => {};

    return (
        <ImportOrchestrator
            fieldDef={TEACHER_INVITE_FIELDS}
            onMappedList={handleMappedList}
            isSubmitting={mutation.isPending}
            progressUrl={(jobId) => `/backend/api/teachers/invitations/jobs/${jobId}/events`}
            showProgress
            onReset={handleReset}
        />
    );
}
