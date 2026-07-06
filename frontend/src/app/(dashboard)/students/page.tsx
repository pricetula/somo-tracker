/**
 * Students listing page — active enrolled students.
 *
 * Maps to GET /api/v1/students/list with support for:
 *   - Search: Full Name, Admission Number, UPI Number, KNEC Assessment Number
 *   - Curriculum filters: Education Level, Grade Level (multi-select)
 *   - Lifecycle filter: Enrollment Status (Active, Suspended, Transferred)
 *
 * The Import button navigates to /students/import. When navigated from
 * within this page, the @modal parallel slot intercepts the route and
 * renders the import pipeline as a dialog overlay (keeping this listing
 * mounted underneath). Direct navigation to /students/import renders
 * as a full-page view.
 */

"use client";

import * as React from "react";
import { useStudents } from "@/features/students";
import { StudentsTable } from "@/features/students";

/** Omit a single key from a record. */
function withoutKey<V>(obj: Record<string, V>, key: string): Record<string, V> {
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    const { [key]: _omit, ...rest } = obj;
    return rest;
}

export default function StudentsPage() {
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

    // Extract enrollment_status for direct param (not a multi-value filter)
    const enrollmentStatus =
        typeof activeFilters.enrollment_status === "string"
            ? activeFilters.enrollment_status
            : undefined;

    // Filter out lifecycle enrollment_status from the filters object (sent as direct param)
    const curriculumFilters = React.useMemo(() => {
        const rest = withoutKey(queryFilters, "enrollment_status");
        return Object.keys(rest).length > 0 ? rest : undefined;
    }, [queryFilters]);

    const { data, isLoading } = useStudents({
        search: search || undefined,
        enrollment_status: enrollmentStatus,
        filters: curriculumFilters,
    });

    const students = data?.items ?? [];
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
        <StudentsTable
            students={students}
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
