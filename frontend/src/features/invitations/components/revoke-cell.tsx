"use client";

import { toast } from "sonner";
import { useQueryClient } from "@tanstack/react-query";
import { XCircle } from "lucide-react";
import { RowActions } from "@/components/shared/data-table/row-actions";
import { type RowAction } from "@/components/shared/data-table/row-actions";
import { revokeInvitation, type Invitation } from "@/lib/api/invitations";
import { getErrorMessage } from "@/lib/errors";

export function RevokeCell({
    invitation,
    queryKey,
}: {
    invitation: Invitation;
    queryKey: readonly unknown[];
}) {
    const queryClient = useQueryClient();

    if (invitation.status !== "pending") {
        return <div className="w-12" />;
    }

    const rowActions: RowAction[] = [
        {
            label: "Revoke",
            icon: XCircle,
            destructive: true,
            confirmTitle: "Revoke Invitation",
            confirmDescription: `Are you sure you want to revoke the invitation for "${invitation.email}"? They will no longer be able to accept it.`,
            onClick: async () => {
                try {
                    await revokeInvitation(invitation.id);
                    queryClient.invalidateQueries({ queryKey });
                    toast.success("Invitation revoked.");
                } catch (err) {
                    toast.error(getErrorMessage(err));
                }
            },
        },
    ];

    return <RowActions rowId={invitation.id} label={invitation.email} actions={rowActions} />;
}
