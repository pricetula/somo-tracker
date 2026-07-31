"use client";

import { useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { RowActions } from "@/components/shared/data-table/row-actions";
import { getErrorMessage } from "@/lib/errors";
import { deleteFeeTemplate, type FeeTemplate } from "@/lib/api/billing";

export function DeleteCell({ template }: { template: FeeTemplate }) {
    const queryClient = useQueryClient();

    const handleDelete = useCallback(async () => {
        try {
            await deleteFeeTemplate(template.id);
            await queryClient.invalidateQueries({ queryKey: ["fee-templates"] });
            toast.success("Fee template deleted.");
        } catch (err) {
            toast.error(getErrorMessage(err));
        }
    }, [template.id, queryClient]);

    return (
        <RowActions
            rowId={template.id}
            label={`${template.grade_level} template`}
            onDelete={handleDelete}
        />
    );
}
