/**
 * Teachers listing page — active teachers with extended educator fields.
 *
 * Uses its own dedicated teachers endpoint (GET /api/v1/teachers) with
 * TSC Number, KNEC Panel Assessor ID, and Core Assignment Role.
 * Curriculum filter: multi-select dropdown for Education Levels or Grade Levels.
 *
 * Invitations are listed on the dedicated /teachers/invitations page.
 */

"use client";

import * as React from "react";
import { useTeachers } from "@/features/staff";
import { TeachersTable } from "@/features/staff";

/** Omit a single key from a record. */
function withoutKey<V>(obj: Record<string, V>, key: string): Record<string, V> {
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    const { [key]: _omit, ...rest } = obj;
    return rest;
}

export default function TeachersPage() {
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

    const { data, isLoading } = useTeachers({
        search: search || undefined,
        filters: Object.keys(queryFilters).length > 0 ? queryFilters : undefined,
    });

    const teachers = data?.items ?? [];
    const total = data?.total ?? 0;

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
        <TeachersTable
            teachers={teachers}
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
