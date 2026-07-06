/**
 * useClassList — TanStack Query hook for fetching classes.
 *
 * The backend defaults to the current academic year, so no params needed.
 * Stable query key + generous staleTime ensures N simultaneously mounted
 * <ClassCombobox /> instances produce exactly ONE network request.
 */

"use client";

import { useQuery } from "@tanstack/react-query";

import { listClasses } from "@/lib/api/classes";
import type { ClassOption } from "../types";

// ─── Query keys ───────────────────────────────────────────────────────────

export const classKeys = {
    all: ["classes"] as const,
    list: () => [...classKeys.all, "list"] as const,
};

// ─── Hook ─────────────────────────────────────────────────────────────────

export function useClassList() {
    return useQuery({
        queryKey: classKeys.list(),
        queryFn: () => listClasses({ limit: 500 }),
        staleTime: 5 * 60 * 1000,
        placeholderData: (prev) => prev,
        select: (data): { items: ClassOption[] } => ({
            items: data.items.map((c) => ({
                value: c.id,
                label: c.display_label,
            })),
        }),
    });
}
