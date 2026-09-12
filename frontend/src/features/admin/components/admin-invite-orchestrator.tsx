"use client";

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

    const handleMappedList = async (rows: MappedRow[]) => {
        const invitations: InvitationRow[] = rows.map((r) => ({
            email: String(r.data.email ?? ""),
            full_name: String(r.data.full_name ?? ""),
        }));
        return mutation.mutateAsync({ invitations });
    };

    return (
        <ImportOrchestrator
            fieldDef={ADMIN_INVITE_FIELDS}
            onMappedList={handleMappedList}
            isSubmitting={mutation.isPending}
        />
    );
}
