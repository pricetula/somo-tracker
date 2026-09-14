"use client";

import React from "react";
import { ImportOrchestrator } from "@/features/import";
import { useBulkInviteFinances } from "../hooks/use-finance-invitations";
import type { MappedRow } from "@/features/import/components/upload/field-mapper";

const FINANCEINVITE_FIELDS = [
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

export function FinanceInviteOrchestrator() {
    const mutation = useBulkInviteFinances();

    const handleMappedList = async (rows: MappedRow[]) => {
        const invitations = rows.map((r) => ({
            email: String(r.data.email ?? ""),
            full_name: String(r.data.full_name ?? ""),
        }));

        const canonical = JSON.stringify(
            invitations.slice().sort((a, b) => a.email.localeCompare(b.email))
        );
        const hash = await sha256Hex(canonical);
        const key = `bulk-finance-invite:${hash}`;

        try {
            sessionStorage.setItem(`bulk-finance-invite-key:${hash}`, key);
        } catch {}

        const resp = await mutation.mutateAsync({ invitations, idempotencyKey: key });
        return { job_id: resp.job_id, total_records: resp.total_records, status: resp.status };
    };

    const handleReset = () => {};

    return (
        <ImportOrchestrator
            fieldDef={FINANCEINVITE_FIELDS}
            onMappedList={handleMappedList}
            isSubmitting={mutation.isPending}
            progressUrl={(jobId) => `/backend/api/finance/invitations/jobs/${jobId}/events`}
            showProgress
            onReset={handleReset}
        />
    );
}
