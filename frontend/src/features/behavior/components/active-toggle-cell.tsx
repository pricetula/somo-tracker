"use client";

import { useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Switch } from "@/components/ui/switch";
import { useUpdateBehaviorCategory } from "../hooks/use-behavior";
import { type BehaviorCategory } from "@/lib/api/behavior";

export function ActiveToggleCell({ category }: { category: BehaviorCategory }) {
    const updateCategory = useUpdateBehaviorCategory();
    const queryClient = useQueryClient();

    const handleToggle = useCallback(() => {
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
    }, [category.id, category.is_active, updateCategory, queryClient]);

    return <Switch checked={category.is_active} onCheckedChange={handleToggle} />;
}
