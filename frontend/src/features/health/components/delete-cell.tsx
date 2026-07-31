"use client";

import { useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { RowActions } from "@/components/shared/data-table/row-actions";
import { deleteMedicalIncident } from "@/lib/api/health";
import { type MedicalIncident } from "@/lib/api/health";
import { getErrorMessage } from "@/lib/errors";

export function DeleteCell({ incident }: { incident: MedicalIncident }) {
    const queryClient = useQueryClient();

    const handleDelete = useCallback(async () => {
        try {
            await deleteMedicalIncident(incident.id);
            await queryClient.invalidateQueries({ queryKey: ["health", "incidents"] });
            toast.success("Incident deleted.");
        } catch (err) {
            toast.error(getErrorMessage(err));
        }
    }, [incident.id, queryClient]);

    return <RowActions rowId={incident.id} label="medical incident" onDelete={handleDelete} />;
}
