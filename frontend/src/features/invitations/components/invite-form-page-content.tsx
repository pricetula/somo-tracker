/**
 * Invite Form Page Content — the standalone page variant of the invite form.
 */

"use client";

import { useRouter } from "next/navigation";
import { InviteFormContent } from "./invite-form-content";

// ─── Types ─────────────────────────────────────────────────────────────────

export interface InviteFormPageContentProps {
    onSuccess: () => void;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function InviteFormPageContent({ onSuccess }: InviteFormPageContentProps) {
    const router = useRouter();

    function handleSuccess() {
        onSuccess();
        router.push("/admins/invitations");
    }

    return (
        <div className="mx-auto flex w-full max-w-2xl flex-col gap-4 p-6">
            <div>
                <h1 className="text-2xl font-semibold tracking-tight">Invite Users</h1>
                <p className="text-muted-foreground mt-1 text-sm">
                    Send invitation emails to join your school. You can choose a role for each
                    person.
                </p>
            </div>
            <InviteFormContent onSuccess={handleSuccess} />
        </div>
    );
}
