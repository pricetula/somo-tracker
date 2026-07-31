"use client";

import { useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { useUpdateBehaviorCategory } from "../hooks/use-behavior";
import { type BehaviorCategory } from "@/lib/api/behavior";

export function SeverityCell({ category }: { category: BehaviorCategory }) {
    const updateCategory = useUpdateBehaviorCategory();
    const queryClient = useQueryClient();

    const handleSeverityChange = useCallback(
        (severity: string) => {
            updateCategory.mutate(
                {
                    id: category.id,
                    payload: {
                        default_severity:
                            severity === "__none__"
                                ? null
                                : (severity as "MINOR" | "NEEDS_FOLLOW_UP"),
                    },
                },
                {
                    onSuccess: () => {
                        queryClient.invalidateQueries({ queryKey: ["behavior", "categories"] });
                    },
                }
            );
        },
        [category.id, updateCategory, queryClient]
    );

    return (
        <Select
            value={category.default_severity ?? "__none__"}
            onValueChange={handleSeverityChange}
        >
            <SelectTrigger className="h-8 w-44">
                <SelectValue placeholder="None" />
            </SelectTrigger>
            <SelectContent>
                <SelectItem value="__none__">None</SelectItem>
                <SelectItem value="MINOR">Minor</SelectItem>
                <SelectItem value="NEEDS_FOLLOW_UP">Needs Follow-up</SelectItem>
            </SelectContent>
        </Select>
    );
}
