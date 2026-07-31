"use client";

import { useQueryClient } from "@tanstack/react-query";
import { ToggleLeft, ToggleRight } from "lucide-react";
import { toast } from "sonner";
import { RowActions } from "@/components/shared/data-table/row-actions";
import { type RowAction } from "@/components/shared/data-table/row-actions";
import { useUpdateBehaviorCategory } from "../hooks/use-behavior";
import { type BehaviorCategory } from "@/lib/api/behavior";

export function ActionsCell({ category }: { category: BehaviorCategory }) {
    const updateCategory = useUpdateBehaviorCategory();
    const queryClient = useQueryClient();

    const actions: RowAction[] = [
        {
            label: category.is_active ? "Deactivate" : "Activate",
            icon: category.is_active ? ToggleLeft : ToggleRight,
            destructive: true,
            confirmTitle: category.is_active ? "Deactivate Category" : "Activate Category",
            confirmDescription: `Are you sure you want to ${
                category.is_active ? "deactivate" : "activate"
            } "${category.name}"?`,
            onClick: () => {
                updateCategory.mutate(
                    { id: category.id, payload: { is_active: !category.is_active } },
                    {
                        onSuccess: () => {
                            queryClient.invalidateQueries({ queryKey: ["behavior", "categories"] });
                            toast.success(
                                category.is_active ? "Category deactivated" : "Category activated"
                            );
                        },
                    }
                );
            },
        },
    ];

    return <RowActions rowId={category.id} label={category.name} actions={actions} />;
}
