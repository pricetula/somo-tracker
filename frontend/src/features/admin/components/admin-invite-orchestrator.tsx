"use client";

import React from "react";
import { ImportOrchestrator } from "@/features/import";
import { useBulkInviteUsers } from "../hooks/use-invitations";
import type { MappedRow } from "@/features/import/components/upload/field-mapper";
import type { InvitationRow } from "@/lib/api/generated";

const ADMIN_INVITE_FIELDS = [
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

export function AdminInviteOrchestrator() {
    const mutation = useBulkInviteUsers();
    const [idempotencyKey, setIdempotencyKey] = React.useState<string | null>(null);

    const handleMappedList = async (rows: MappedRow[]) => {
        // MappedRow.data already contains fields matching InvitationRow
        const invitations: InvitationRow[] = rows.map((r) => ({
            email: String(r.data.email ?? ""),
            full_name: String(r.data.full_name ?? ""),
        }));
        // Generate key once per mapping session; reuse on retry
        const key = idempotencyKey ?? crypto.randomUUID();
        setIdempotencyKey(key);
        const resp = await mutation.mutateAsync({ invitations, idempotencyKey: key });
        // Return shape expected by ImportOrchestrator
        return { job_id: resp.job_id, total_records: resp.total_records, status: resp.status };
    };

    const handleReset = () => {
        setIdempotencyKey(null);
    };

    return (
        <ImportOrchestrator
            fieldDef={ADMIN_INVITE_FIELDS}
            onMappedList={handleMappedList}
            isSubmitting={mutation.isPending}
            progressUrl={(jobId) => `/backend/api/admins/invitations/jobs/${jobId}/events`}
            showProgress
            onReset={handleReset}
        />
    );
}
