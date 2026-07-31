"use client";

import { RowActions } from "@/components/shared/data-table/row-actions";
import { useDeleteScaleProfile } from "../hooks/use-assessments";

export function DeleteCell({ profileId, profileName }: { profileId: string; profileName: string }) {
    const deleteMutation = useDeleteScaleProfile();

    return (
        <RowActions
            rowId={profileId}
            label={profileName}
            onDelete={() => deleteMutation.mutate(profileId)}
            disabled={deleteMutation.isPending}
        />
    );
}
