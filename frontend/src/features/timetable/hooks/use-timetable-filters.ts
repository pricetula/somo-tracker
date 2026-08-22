"use client";

import { useState, useCallback, useMemo } from "react";
import type { TimetableFilters } from "../types";

/** Hook for managing timetable view filters */
export function useTimetableFilters(initialFilters: TimetableFilters = {}) {
    const [filters, setFilters] = useState<TimetableFilters>(initialFilters);

    const updateFilter = useCallback(
        <K extends keyof TimetableFilters>(key: K, value: TimetableFilters[K]) => {
            setFilters((prev) => ({ ...prev, [key]: value }));
        },
        []
    );

    const clearFilters = useCallback(() => {
        setFilters({});
    }, []);

    const clearFilter = useCallback(<K extends keyof TimetableFilters>(key: K) => {
        setFilters((prev) => {
            const next = { ...prev };
            delete next[key];
            return next;
        });
    }, []);

    const hasActiveFilters = useMemo(
        () => Object.values(filters).some((v) => v !== undefined && v !== ""),
        [filters]
    );

    return {
        filters,
        setFilters,
        updateFilter,
        clearFilters,
        clearFilter,
        hasActiveFilters,
    };
}
