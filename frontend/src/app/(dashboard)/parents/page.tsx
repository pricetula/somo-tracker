/**
 * Parents listing page.
 *
 * Shows all parent/guardian profiles with search and curriculum filter.
 * Maps to GET /api/v1/parents.
 *
 * Curriculum filter filters parents whose linked children are in
 * selected education levels or grades.
 */

"use client";

import * as React from "react";
import { useParents, ParentsTable } from "@/features/parents";
import type { ListParentsResponse } from "@/features/parents";

/** Omit a single key from a record. */
function withoutKey<V>(obj: Record<string, V>, key: string): Record<string, V> {
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    const { [key]: _omit, ...rest } = obj;
    return rest;
}

export default function ParentsPage() {
    const [search, setSearch] = React.useState("");
    const [activeFilters, setActiveFilters] = React.useState<Record<string, string | string[]>>({});

    // Derive filters object for the query hook
    const queryFilters = React.useMemo(() => {
        const filters: Record<string, string[]> = {};
        for (const [key, value] of Object.entries(activeFilters)) {
            if (Array.isArray(value) && value.length > 0) {
                filters[key] = value;
            }
        }
        return filters;
    }, [activeFilters]);

    const { data, isLoading } = useParents({
        search: search || undefined,
        filters: Object.keys(queryFilters).length > 0 ? queryFilters : undefined,
    });

    const parents = (data as ListParentsResponse | undefined)?.items ?? [];
    const total = (data as ListParentsResponse | undefined)?.total ?? 0;

    const handleToggleButton = React.useCallback((itemId: string, itemValue: string) => {
        setActiveFilters((prev) => {
            if (typeof prev[itemId] === "string" && prev[itemId] === itemValue) {
                return withoutKey(prev, itemId);
            }
            return { ...prev, [itemId]: itemValue };
        });
    }, []);

    const handleSelectSingle = React.useCallback((itemId: string, subValue: string) => {
        setActiveFilters((prev) => {
            if (typeof prev[itemId] === "string" && prev[itemId] === subValue) {
                return withoutKey(prev, itemId);
            }
            return { ...prev, [itemId]: subValue };
        });
    }, []);

    const handleToggleMulti = React.useCallback((itemId: string, subValue: string) => {
        setActiveFilters((prev) => {
            const current = prev[itemId];
            const arr = Array.isArray(current) ? current : [];
            if (arr.includes(subValue)) {
                const next = arr.filter((v) => v !== subValue);
                if (next.length === 0) {
                    return withoutKey(prev, itemId);
                }
                return { ...prev, [itemId]: next };
            }
            return { ...prev, [itemId]: [...arr, subValue] };
        });
    }, []);

    return (
        <ParentsTable
            parents={parents}
            total={total}
            isLoading={isLoading}
            search={search}
            onSearchChange={setSearch}
            activeFilters={activeFilters}
            onToggleButton={handleToggleButton}
            onSelectSingle={handleSelectSingle}
            onToggleMulti={handleToggleMulti}
        />
    );
}
