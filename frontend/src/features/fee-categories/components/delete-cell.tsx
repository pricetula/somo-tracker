"use client";

import { useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { RowActions } from "@/components/shared/data-table/row-actions";
import { getErrorMessage } from "@/lib/errors";
import { deleteFeeCategory, type FeeCategory } from "@/lib/api/billing";

export function DeleteCell({ category }: { category: FeeCategory }) {
    const queryClient = useQueryClient();

    const handleDelete = useCallback(async () => {
        try {
            await deleteFeeCategory(category.id);
            await queryClient.invalidateQueries({ queryKey: ["fee-categories"] });
            toast.success("Fee category deleted.");
        } catch (err) {
            toast.error(getErrorMessage(err));
        }
    }, [category.id, queryClient]);

    return <RowActions rowId={category.id} label={category.name} onDelete={handleDelete} />;
}
